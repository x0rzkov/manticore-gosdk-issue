package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/feeds"
	"github.com/gosimple/slug"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/k0kubun/pp"
	"github.com/manticoresoftware/go-sdk/manticore"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

var (
	help           bool
	index          bool
	search         string
	inputFile      string
	serverPort     string
	serverAddr     string
	manticoreIndex string
	manticoreHost  string
	manticorePort  uint16
)

func main() {
	pflag.BoolVarP(&index, "index", "i", false, "index content.")
	pflag.StringVarP(&search, "search", "s", "", "search content.")
	pflag.StringVarP(&serverPort, "server-port", "", "8086", "server port.")
	pflag.StringVarP(&serverAddr, "server-addr", "", "0.0.0.0", "server address.")
	pflag.StringVarP(&manticoreHost, "manticore-host", "m", "manticore", "manticore host.")
	pflag.Uint16VarP(&manticorePort, "manticore-port", "p", 9312, "manticore port.")
	pflag.StringVarP(&manticoreIndex, "manticore-index", "", "rt_papers", "manticore index name.")
	pflag.StringVarP(&inputFile, "input-file", "f", "./data/manticore_papers.sql", "input-file")
	pflag.BoolVarP(&help, "help", "h", false, "display help")
	pflag.Parse()
	if help {
		pflag.PrintDefaults()
		os.Exit(1)
	}

	if index {
		cl, err := sql.Open("mysql", fmt.Sprintf("@tcp(%s:%d)/", manticoreHost, manticorePort))
		if err != nil {
			log.Fatal(err)
		}

		cl.SetMaxIdleConns(0)
		cl.SetConnMaxLifetime(0)

		dat, err := ioutil.ReadFile(inputFile)
		check(err)

		_, err = cl.Exec(string(dat))
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Indexing done !")
		os.Exit(1)
	}

	if search != "" {
		// init manticore full-text index
		cl, _, err := initSphinx(manticoreHost, manticorePort)
		if err != nil {
			log.Fatal(err)
		}

		offset := 0
		perPage := 20
		matchQuery := "WHERE MATCH('" + search + "')"
		filterStr := ""
		orderBy := "ORDER BY stars DESC"

		queryStr := "SELECT id,stars,created_at,updated_at,deleted_at,published_time,link,pdf,abstract,publisher,codes,links,authors,tasks,subjects,referers,title,summary, HIGHLIGHT({passage_boundary='sentence'},'summary'),frameworks,languages FROM rt_papers " + matchQuery + " " + filterStr + " " + orderBy + " LIMIT " + fmt.Sprintf("%d", offset) + "," + fmt.Sprintf("%d", perPage) + " FACET published_time FACET subjects FACET tasks FACET publisher FACET authors FACET frameworks FACET languages;"
		pp.Println("queryStr", queryStr)
		res, err := cl.Sphinxql(queryStr)
		if err != nil {
			log.Fatal(err)
		}
		pp.Println(res)
		fmt.Println("Search done !")
		os.Exit(1)
	}

	r := setupRouter()
	// Listen and Server in 0.0.0.0:8080
	r.Run(fmt.Sprintf("%s:%s", serverAddr, serverPort))

}

func initSphinx(host string, port uint16) (manticore.Client, bool, error) {
	cl := manticore.NewClient()
	cl.SetServer(host, port)
	status, err := cl.Open()
	if err != nil {
		return cl, status, err
	}
	return cl, status, nil
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func setupRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/rss/", func(c *gin.Context) {

		payload := make(map[string]interface{})
		cl, _, err := initSphinx(manticoreHost, manticorePort)
		if err != nil {
			log.Fatal(err)
		}
		defer cl.Close()

		offset := 0
		perPage := 40

		if c.Query("page") != "" {
			pageQuery, err := strconv.Atoi(c.Query("page"))
			if err != nil {
				log.Fatal(err)
			}
			offset = perPage * pageQuery
		}

		var orderBy string //, orderCodeBy string
		if c.Query("order_by") != "" {
			switch c.Query("order_by") {
			//orderCodeBy = "ORDER BY id DESC" // add last update time for repo
			case "least_recent":
				orderBy = "ORDER BY published_time ASC"
				//orderCodeBy = "ORDER BY id ASC" // add last update time for repo
			case "least_popular":
				orderBy = "ORDER BY stars ASC"
				//orderCodeBy = "ORDER BY stars ASC"
			case "most_relevant":
				orderBy = ""
				//orderCodeBy = "ORDER BY stars DESC"
			case "most_popular":
				orderBy = "ORDER BY stars DESC"
			case "most_recent":
				fallthrough
			default:
				orderBy = "ORDER BY published_time DESC"
				//orderCodeBy = "ORDER BY stars DESC"
			}
		} else {
			//orderCodeBy = "ORDER BY stars DESC"
		}

		facets := map[string]string{
			"subjects":       "",
			"tasks":          "",
			"publisher":      "",
			"authors":        "",
			"stars":          "",
			"published_time": "",
			"frameworks":     "",
			"languages":      "",
		}

		for key, val := range c.Request.URL.Query() {
			if strings.HasPrefix(key, "f_") {
				key = strings.Replace(key, "f_", "", -1)
				if _, ok := facets[key]; ok {
					switch key {
					case "subjects":
						fallthrough
					case "tasks":
						fallthrough
					case "publisher":
						fallthrough
					case "authors":
						fallthrough
					case "frameworks":
						fallthrough
					case "languages":
						fallthrough
					case "stars":
						fallthrough
					case "published_time":
						if val[0] != "" {
							params := strings.Split(val[0], ",")
							for _, parameter := range params {
								if parameter != "" {
									facets[key] += key + "=" + escape(parameter) + " OR "
								}
							}
						}
					}
				}
			}
		}

		var facetSlice []string
		for _, facet := range facets {
			if facet != "" {
				facetSlice = append(facetSlice, facet)
			}
		}

		var filterStr, filterCodeStr string
		if len(facetSlice) > 0 {
			filterStr += strings.Join(facetSlice, " ")
		}

		if strings.HasSuffix(filterStr, " OR ") {
			sz := len(filterStr)
			filterStr = filterStr[:sz-4]
		}

		for _, facet := range facetSlice {
			if strings.Contains(facet, "language") || strings.Contains(facet, "framework") {
				filterCodeStr += facet
			}
		}

		if strings.HasSuffix(filterCodeStr, " OR ") {
			sz := len(filterCodeStr)
			filterCodeStr = filterCodeStr[:sz-4]
		}

		decodedQuery, err := url.QueryUnescape(c.Query("q"))
		if err != nil {
			log.Fatal("decodedQuery:", err)
		}

		var matchQuery, matchCodeQuery string
		if decodedQuery != "" {
			matchQuery = "WHERE MATCH('" + decodedQuery + "') "
			matchCodeQuery = "WHERE MATCH('" + decodedQuery + "') "
			if filterStr != "" {
				matchQuery += " AND "
			}
			if filterCodeStr != "" {
				matchCodeQuery += " AND "
			}
		}
		if c.Query("q") == "" && filterStr != "" {
			matchQuery += " WHERE "
		}

		feed := &feeds.Feed{
			Title:       "Paper2code",
			Link:        &feeds.Link{Href: c.Request.URL.Path},
			Description: "",
			Author:      &feeds.Author{Name: "Paper2code", Email: "bot@paper2code.com"},
		}

		// papers
		queryStr := "SELECT id,stars,created_at,updated_at,deleted_at,published_time,pdf,abstract,publisher,codes,links,authors,tasks,subjects,referers,title,summary,frameworks,languages FROM rt_papers " + matchQuery + " " + filterStr + " " + orderBy + " LIMIT " + fmt.Sprintf("%d", offset) + "," + fmt.Sprintf("%d", perPage)
		fmt.Println("queryStr", queryStr)
		res, err := cl.Sphinxql(queryStr)
		if err != nil {
			log.Infoln("GetLastWarning:", cl.GetLastWarning())
			log.Warnln("Sphinxql:", err)
			//Do what you need to get the cached html
			errSphinxQL := "<html><body>Error occured</body></html>"
			//Write your 200 header status (or other status codes, but only WriteHeader once)
			c.String(500, errSphinxQL)
			return
		}

		if len(res) > 0 {
			for _, r := range res[0].Rows {

				i, err := strconv.ParseInt(fmt.Sprintf("%d", int(r[5].(int32))), 10, 64)
				if err != nil {
					panic(err)
				}
				pubDate := time.Unix(i, 0) // .Format("2006-01-02")
				// fmt.Println(tm)
				// pubDate, err := time.Format("2006-01-02", tm)
				// if err != nil {
				// 	continue
				// }
				linkUrl := fmt.Sprintf("https://paper2code.com/paper/%d/%s", r[0].(int64), slug.Make(r[15].(string)))

				item := feeds.Item{
					Title:       r[15].(string),
					Link:        &feeds.Link{Href: linkUrl},
					Description: "",
					Content:     strings.Replace(r[16].(string), "\n", "", -1),
					Author:      &feeds.Author{Name: "Paper2code", Email: "bot@paper2code.com"},
					Created:     pubDate,
				}
				feed.Items = append(feed.Items, &item)
			}
		}

		rss, err := feed.ToRss()
		if err != nil {
			payload["error"] = "could not parse the paperID"
			c.JSON(http.StatusOK, payload)
			return
		}
		c.String(http.StatusOK, rss)

	})

	// Get user value
	r.GET("/search/:query", func(c *gin.Context) {
		query := c.Params.ByName("query")
		cl, _, err := initSphinx(manticoreHost, manticorePort)
		if err != nil {
			log.Fatal(err)
		}
		defer cl.Close()

		offset := 0
		perPage := 20
		matchQuery := "WHERE MATCH('" + query + "')"
		filterStr := ""
		orderBy := "ORDER BY published_time DESC"

		queryStr := "SELECT id,stars,created_at,updated_at,deleted_at,published_time,link,pdf,abstract,publisher,codes,links,authors,tasks,subjects,referers,title,summary, HIGHLIGHT({passage_boundary='sentence'},'summary'),frameworks,languages FROM rt_papers " + matchQuery + " " + filterStr + " " + orderBy + " LIMIT " + fmt.Sprintf("%d", offset) + "," + fmt.Sprintf("%d", perPage) + " FACET published_time FACET subjects FACET tasks FACET publisher FACET authors FACET frameworks FACET languages;"
		pp.Println("queryStr", queryStr)
		res, err := cl.Sphinxql(queryStr)
		if err != nil {
			log.Fatal(err)
		}
		pp.Println(res)
		fmt.Println("Search done !")
		c.IndentedJSON(http.StatusOK, res)
	})

	return r
}

func escape(sql string) string {
	dest := make([]byte, 0, 2*len(sql))
	var escape byte
	for i := 0; i < len(sql); i++ {
		c := sql[i]

		escape = 0

		switch c {
		case 0: /* Must be escaped for 'mysql' */
			escape = '0'
			break
		case '\n': /* Must be escaped for logs */
			escape = 'n'
			break
		case '\r':
			escape = 'r'
			break
		case '\\':
			escape = '\\'
			break
		case '\'':
			escape = '\''
			break
		case '"': /* Better safe than sorry */
			escape = '"'
			break
		case '\032': /* This gives problems on Win32 */
			escape = 'Z'
		}

		if escape != 0 {
			dest = append(dest, '\\', escape)
		} else {
			dest = append(dest, c)
		}
	}

	return string(dest)
}
