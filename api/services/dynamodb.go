package services

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"os"
)

func PutItemInDynamoDB(config *aws.Config, Item *map[string]types.AttributeValue) (*dynamodb.PutItemOutput, error) {
	tableName, ok := os.LookupEnv("TABLE_NAME")

	if !ok {
		panic("Dynamodb table name is not set")
	}

	fmt.Printf("Table Name: %s", tableName)
	svc := dynamodb.NewFromConfig(*config)

	// PutItem in DynamoDB
	return svc.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      *Item,
	})
}
