package momochi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Type is type of value
type Type string

const (
	// TypeCo2 is type of co2
	TypeCo2 Type = "co2"
	// Co2MgmtID is manegement id of co2
	Co2MgmtID = "co2-latest"

	// TypeTemp is type of temp
	TypeTemp Type = "temperature"
	// TempMgmtID is manegement id of temp
	TempMgmtID = "temperature-latest"
)

// MgmtLastValue is last value of id
type MgmtLastValue struct {
	ID   string `json:"id" dynamo:"id"`
	Time int64  `json:"time" dynamo:"time"`
}

// Keys is base keys of values
type Keys struct {
	Time int64 `json:"time" dynamo:"time"`
	Type Type  `json:"type" dynamo:"type"`
}

// Co2 is co2 value
type Co2 struct {
	Keys
	PPM int `json:"co2" dynamo:"co2"`
}

// Temp is temperature value
type Temp struct {
	Keys
	Temp  float32 `json:"temperature" dynamo:"temperature"`
	Humid float32 `json:"humidity" dynamo:"humidity"`
}

func (t *Temp) toJSON() io.Reader {
	buf := &bytes.Buffer{}
	json.NewEncoder(buf).Encode(t)
	return buf
}

func NewTemp(temp float32, humid float32) Temp {
	return Temp{
		Keys: Keys{
			Type: TypeTemp,
			Time: time.Now().Unix(),
		},
		Temp:  temp,
		Humid: humid,
	}
}

type MomochiClient interface {
	PostTemp(Temp) (*http.Response, error)
	PostCo2(Co2) (*http.Response, error)
}

type momochiClient struct {
	Endpoint string
}

func NewClient(e string) MomochiClient {
	return &momochiClient{Endpoint: e}
}

func (m *momochiClient) PostTemp(t Temp) (*http.Response, error) {
	uri := m.Endpoint + "/temp/add"
	req, err := http.NewRequest(http.MethodPost, uri, t.toJSON())
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}
func (m *momochiClient) PostCo2(t Co2) (*http.Response, error) {
	return nil, fmt.Errorf("unimpled")
}

var _ MomochiClient = (*momochiClient)(nil)
