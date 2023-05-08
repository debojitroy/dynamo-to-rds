package services

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"log"
	"sync"
)

var ssmLock = &sync.Mutex{}
var ssmClient *ssm.Client

func initialiseSsmClient() *ssm.Client {
	ssmLock.Lock()
	defer ssmLock.Unlock()

	if ssmClient == nil {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			log.Fatalf("Failed to load config: %v ", err)
		}

		ssmClient = ssm.NewFromConfig(cfg)
	}

	return ssmClient
}

func GetSsmParameter(paramName string) (*ssm.GetParameterOutput, error) {
	input := &ssm.GetParameterInput{
		Name: aws.String(paramName),
	}

	initialiseSsmClient()

	return ssmClient.GetParameter(context.TODO(), input)
}
