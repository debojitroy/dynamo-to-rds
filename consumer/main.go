package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	runtime "github.com/aws/aws-lambda-go/lambda"
	"github.com/debojitroy/dynamo-to-rds/consumer/models"
	"github.com/debojitroy/dynamo-to-rds/consumer/services"
	_ "github.com/go-sql-driver/mysql"
	"log"
)

func handleRequest(_ context.Context, event events.DynamoDBEvent) (string, error) {
	// event
	eventJson, _ := json.Marshal(event)
	log.Println("EVENT")
	log.Println(string(eventJson))

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

	for _, record := range event.Records {
		_, err := models.ProcessDynamoDbEvent(&record, db)

		if err != nil {
			log.Fatalf("Failed to process record: %v", err)
		}
	}

	return "Successfully processed records", nil
}

func main() {
	runtime.Start(handleRequest)
}
