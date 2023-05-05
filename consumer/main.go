package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	runtime "github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	_ "github.com/go-sql-driver/mysql"
	"log"
)

func handleRequest(ctx context.Context, event events.DynamoDBEvent) (string, error) {
	// event
	eventJson, _ := json.MarshalIndent(event, "", "  ")
	log.Printf("EVENT: %s", eventJson)

	// request context
	lc, _ := lambdacontext.FromContext(ctx)
	log.Printf("REQUEST ID: %s", lc.AwsRequestID)

	// global variable
	log.Printf("FUNCTION NAME: %s", lambdacontext.FunctionName)

	// context method
	deadline, _ := ctx.Deadline()
	log.Printf("DEADLINE: %s", deadline)

	userName := "admin"
	password := "*********"
	hostname := "******"
	port := "3306"
	dbName := "pgrouter"

	connectionString := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", userName, password, hostname, port, dbName)

	log.Printf("Connection String: %s \n", connectionString)

	db, err := sql.Open("mysql",
		connectionString)
	if err != nil {
		log.Fatal(err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Printf("Failed to close DB connection: %v", err)
		}
	}(db)

	var dateTime string

	err = db.QueryRow("select now()").Scan(&dateTime)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Date: %s \n", dateTime)

	return dateTime + ": hello dynamodb", nil
}

func main() {
	runtime.Start(handleRequest)
}
