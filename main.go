package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/k0kubun/pp"
	"github.com/manticoresoftware/go-sdk/manticore"
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

	// Get user value
	r.GET("/search/:query", func(c *gin.Context) {
		query := c.Params.ByName("query")
		cl, _, err := initSphinx(manticoreHost, manticorePort)
		if err != nil {
			log.Fatal(err)
		}

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
