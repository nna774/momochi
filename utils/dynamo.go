package utils

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
)

var endpoint = os.Getenv("DYNAMODB_ENDPOINT")

// Table is util for dynamo.Table
func Table(name string) dynamo.Table {
	cfg := aws.NewConfig()
	if endpoint != "" {
		cfg = cfg.WithEndpoint(endpoint)
	}
	db := dynamo.New(session.New(), cfg)
	return db.Table(name)
}
