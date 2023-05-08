package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	runtime "github.com/aws/aws-lambda-go/lambda"
	"github.com/debojitroy/dynamo-to-rds/consumer/services"
	_ "github.com/go-sql-driver/mysql"
	"log"
)

func handleRequest(_ context.Context, event events.DynamoDBEvent) (string, error) {
	// event
	eventJson, _ := json.MarshalIndent(event, "", "  ")
	log.Printf("EVENT: %s", eventJson)

	db, err := services.GetDbConnection()

	if err != nil {
		log.Fatal(err)
	}

	defer func(db *sql.DB) {
		err := services.CloseDBConnection(db)
		if err != nil {
			log.Println("Failed to close DB Connection...")
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
