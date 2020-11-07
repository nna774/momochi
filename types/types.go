package types

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

// Keys is 
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
	Temp  float32 `json:"temperature"`
	Humid float32 `json:"humidity"`
}
