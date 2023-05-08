package services

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"log"
	"sync"
)

var secretsLock = &sync.Mutex{}
var secretsClient *secretsmanager.Client

func initialiseSecretsClient() *secretsmanager.Client {
	secretsLock.Lock()
	defer secretsLock.Unlock()

	if secretsClient == nil {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			log.Fatalf("Failed to load config: %v ", err)
		}

		secretsClient = secretsmanager.NewFromConfig(cfg)
	}

	return secretsClient
}

func GetSecretValue(secretArn string) (*secretsmanager.GetSecretValueOutput, error) {
	initialiseSecretsClient()

	return secretsClient.GetSecretValue(context.TODO(), &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretArn),
	})
}
