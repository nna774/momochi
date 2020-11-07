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

	"github.com/nna774/momochi/types"
)

var (
	endpoint  = os.Getenv("DYNAMODB_ENDPOINT")
	co2Table  = os.Getenv("DYNAMODB_TABLE")
	tempTable = os.Getenv("DYNAMODB_TEMP_TABLE")
	mgmtTable = os.Getenv("DYNAMODB_MGMT_TABLE")

	// HostID is mackerel Host ID
	HostID = os.Getenv("HOST_ID")
	// APIKey is mackerel API Key
	APIKey = os.Getenv("API_KEY")

	mackerelEndpoint = "https://mackerel.io/api/v0/tsdb"
)

func table(name string) dynamo.Table {
	cfg := aws.NewConfig()
	if endpoint != "" {
		cfg = cfg.WithEndpoint(endpoint)
	}
	db := dynamo.New(session.New(), cfg)
	return db.Table(name)
}

type mackerel struct {
	HostID string      `json:"hostId"`
	Name   string      `json:"name"`
	Time   int64       `json:"time"`
	Value  interface{} `json:"value"`
}

func co2NotifyToMackerel(co2 types.Co2) error {
	return notifyToMackerel([]mackerel{
		{
			HostID: HostID,
			Name:   "custom.co2.ppm",
			Time:   co2.Time,
			Value:  co2.PPM,
		},
	})
}

func tempNotifyToMackerel(temp types.Temp) error {
	return notifyToMackerel([]mackerel{
		{
			HostID: HostID,
			Name:   "custom.temp.temp",
			Time:   temp.Time,
			Value:  temp.Temp,
		},
		{
			HostID: HostID,
			Name:   "custom.temp.humid",
			Time:   temp.Time,
			Value:  temp.Humid,
		},
	})
}

func notifyToMackerel(vals []mackerel) error {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(vals)
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
	var co2 types.Co2
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
	co2.Type = types.TypeCo2

	err = table(co2Table).Put(co2).Run()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "value put failed: %v", err)
		return
	}
	err = putMgmtValue(types.Co2MgmtID, t)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "mgmt put failed: %v", err)
		return
	}
	err = co2NotifyToMackerel(co2)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "mackerel put failed: %v", err)
		return
	}
	fmt.Fprint(w, "ok\n")
}

func tempAddHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "post")
		return
	}
	t := time.Now().Unix()
	var temp types.Temp
	err := json.NewDecoder(r.Body).Decode(&temp)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "decode fail: %v", err)
		return
	}
	temp.Time = t
	temp.Type = types.TypeTemp

	err = table(tempTable).Put(temp).Run()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "value put failed: %v", err)
		return
	}
	err = putMgmtValue(types.TempMgmtID, t)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "mgmt put failed: %v", err)
		return
	}
	err = tempNotifyToMackerel(temp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "mackerel put failed: %v", err)
		return
	}
	fmt.Fprint(w, "ok\n")
}

func putMgmtValue(id string, time int64) error {
	m := types.MgmtLastValue{ID: id, Time: time}
	return table(mgmtTable).Put(m).Run()
}

func lastHandler(w http.ResponseWriter, r *http.Request, id string, from string) {
	m := types.MgmtLastValue{}
	err := table(mgmtTable).Get("id", id).One(&m)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "mgmt get failed: %v", err)
		return
	}

	value := map[string]interface{}{}
	err = table(from).Get("time", m.Time).One(&value)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "value get failed: %v", err)
		return
	}
	json.NewEncoder(w).Encode(value)
}

func co2LastHandler(w http.ResponseWriter, r *http.Request) {
	lastHandler(w, r, types.Co2MgmtID, co2Table)
}

func tempLastHandler(w http.ResponseWriter, r *http.Request) {
	lastHandler(w, r, types.TempMgmtID, tempTable)
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "Hello!(momochi)") })
	http.HandleFunc("/co2/add", co2AddHandler)
	http.HandleFunc("/co2/last", co2LastHandler)
	http.HandleFunc("/temp/add", tempAddHandler)
	http.HandleFunc("/temp/last", tempLastHandler)
	if os.Getenv("MOMOCHI_ENV") == "development" {
		panic(http.ListenAndServe(":8000", nil))
	} else {
		algnhsa.ListenAndServe(nil, &algnhsa.Options{RequestType: algnhsa.RequestTypeAPIGateway})
	}
}
