package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"

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
	manticoreIndex string
	manticoreHost  string
	manticorePort  uint16
)

func main() {
	pflag.BoolVarP(&index, "index", "i", false, "index content.")
	pflag.StringVarP(&search, "search", "s", "", "search content.")
	pflag.StringVarP(&manticoreHost, "manticore-host", "m", "localhost", "manticore host")
	pflag.Uint16VarP(&manticorePort, "manticore-port", "p", 9307, "manticore-port")
	pflag.StringVarP(&manticoreIndex, "manticore-index", "", "rt_papers", "manticore host")

	pflag.StringVarP(&inputFile, "input-file", "f", "./manticore_papers.sql", "input-file")
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

		csvfile, err := os.Open(inputFile)
		if err != nil {
			log.Fatal(err)
		}
		defer csvfile.Close()

		r := csv.NewReader(csvfile)

		for {
			// Read each record from csv
			record, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("query", record[0])

			_, err = cl.Exec(record[0])
			if err != nil {
				log.Fatal(err)
			}
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

		res, err := cl.Query(search, manticoreIndex)
		if err != nil {
			log.Fatal(err)
		}
		pp.Println(res)
		fmt.Println("Search done !")
		os.Exit(1)
	}

	fmt.Println("exit.")
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
