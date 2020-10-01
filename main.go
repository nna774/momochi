package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/akrylysov/algnhsa"
)

var (
	table     = os.Getenv("DYNAMODB_TABLE")
	mgmtTable = os.Getenv("DYNAMODB_MGMT_TABLE")
)

func co2AddHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, table)
}
func co2LastHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, mgmtTable)
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "Hello!(momochi)") })
	http.HandleFunc("/co2/add", co2AddHandler)
	http.HandleFunc("/co2/last", co2LastHandler)
	if os.Getenv("MOMOCHI_ENV") == "development" {
		panic(http.ListenAndServe(":8000", nil))
	} else {
		algnhsa.ListenAndServe(nil, &algnhsa.Options{RequestType: algnhsa.RequestTypeAPIGateway})
	}
}
