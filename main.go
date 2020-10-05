package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/akrylysov/algnhsa"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
)

var (
	endpoint  = os.Getenv("DYNAMODB_ENDPOINT")
	dataTable = os.Getenv("DYNAMODB_TABLE")
	mgmtTable = os.Getenv("DYNAMODB_MGMT_TABLE")

	HostID = os.Getenv("HOST_ID")
	APIKey = os.Getenv("API_KEY")

	mackerelEndpoint = "https://mackerel.io/api/v0/tsdb"
)

const (
	TypeCo2   = "co2"
	Co2MgmtID = "co2-latest"
)

type MgmtLastValue struct {
	ID   string `json:"id" dynamo:"id"`
	Time int64  `json:"time" dynamo:"time"`
}

type Co2 struct {
	Time int64  `json:"time" dynamo:"time"`
	Type string `json:"type" dynamo:"type"`
	PPM  int    `json:"co2" dynamo:"co2"`
}

func table(name string) dynamo.Table {
	cfg := aws.NewConfig()
	if endpoint != "" {
		cfg = cfg.WithEndpoint(endpoint)
	}
	db := dynamo.New(session.New(), cfg)
	return db.Table(name)
}

func notifyToMackerel(co2 Co2) error {
	body := []map[string]interface{}{
		{
			"hostId": HostID,
			"name":   "custom.co2.ppm",
			"time":   co2.Time,
			"value":  co2.PPM,
		},
	}
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, mackerelEndpoint, &buf)
	if err != nil {
		return err
	}
	req.Header.Add("content-type", "application/json")
	req.Header.Add("x-api-key", APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if !(200 <= resp.StatusCode && resp.StatusCode < 300) {
		b, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("mackerel return code not ok: %v, %s", resp.StatusCode, b)
	}
	io.Copy(ioutil.Discard, resp.Body)
	return nil
}

func co2AddHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "post")
		return
	}
	t := time.Now().Unix()
	var co2 Co2
	err := json.NewDecoder(r.Body).Decode(&co2)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "decode fail: %v", err)
		return
	}
	if co2.PPM == 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "need `co2` field: %v", err)
		return
	}
	co2.Time = t
	co2.Type = TypeCo2

	err = table(dataTable).Put(co2).Run()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "value put failed: %v", err)
		return
	}
	m := MgmtLastValue{ID: Co2MgmtID, Time: t}
	err = table(mgmtTable).Put(m).Run()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "mgmt put failed: %v", err)
		return
	}
	err = notifyToMackerel(co2)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "mackerel put failed: %v", err)
		return
	}
	fmt.Fprint(w, "ok\n")
}
func co2LastHandler(w http.ResponseWriter, r *http.Request) {
	m := MgmtLastValue{}
	err := table(mgmtTable).Get("id", Co2MgmtID).One(&m)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "mgmt get failed: %v", err)
		return
	}
	var co2 Co2
	err = table(dataTable).Get("time", m.Time).One(&co2)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "value get failed: %v", err)
		return
	}
	json.NewEncoder(w).Encode(co2)
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
