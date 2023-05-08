package services

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"os"
)

const rdsConnParamName = "rds_conn_param_name"
const rdsSecretsArn = "rds_cred_secrets_arn"

type DBConnection struct {
	Hostname string `json:"hostname"`
	Port     int    `json:"port"`
	Database string `json:"db"`
	Driver   string `json:"driver"`
}

type RDSCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func GetDbConnection() (*sql.DB, error) {
	dbConnectionDetails, dbConnErr := getDBConnectionDetails()

	if dbConnErr != nil {
		return nil, dbConnErr
	}

	rdsCredentials, rdsCredErr := getRDSCredentials()

	if rdsCredErr != nil {
		return nil, rdsCredErr
	}

	connectionString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", rdsCredentials.Username, rdsCredentials.Password, dbConnectionDetails.Hostname, dbConnectionDetails.Port, dbConnectionDetails.Database)

	return sql.Open(dbConnectionDetails.Driver,
		connectionString)
}

func CloseDBConnection(db *sql.DB) error {
	return db.Close()
}

func getDBConnectionDetails() (*DBConnection, error) {
	// Read param value from Env
	connParamName, ok := os.LookupEnv(rdsConnParamName)

	if !ok {
		return nil, errors.New(fmt.Sprintf("'%s' env variable is not defined", rdsConnParamName))
	}

	// Try to retrieve value from cache
	rdsConnectionObjString, found := GetValue(connParamName)

	if found {
		log.Print("Found DB Connection Object in Cache...")
	} else {
		log.Print("NOT Found DB Connection Object in Cache...")

		// If not found in cache, lookup SSM
		results, err := GetSsmParameter(connParamName)

		if err != nil {
			return nil, errors.New(fmt.Sprintf("Failed to get param value: %s, Error: %v", connParamName, err))
		}

		rdsConnectionObjString = results.Parameter.Value

		log.Print("Saving DB Connection Object in Cache...")

		// If found in SSM, save in cache
		SetValue(connParamName, *rdsConnectionObjString)
	}

	var dbConnectionObject DBConnection

	err := json.Unmarshal([]byte(*rdsConnectionObjString), &dbConnectionObject)

	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to parse Json for DB Connection Object, Error: %v", err))
	}

	return &dbConnectionObject, nil
}

func getRDSCredentials() (*RDSCredentials, error) {
	// Read param value from Env
	rdsSecretsArnName, ok := os.LookupEnv(rdsSecretsArn)

	if !ok {
		return nil, errors.New(fmt.Sprintf("'%s' env variable is not defined", rdsSecretsArn))
	}

	// Try to retrieve value from cache
	rdsCredentialsString, found := GetValue(rdsSecretsArnName)

	if found {
		log.Print("Found RDS Credentials in Cache...")
	} else {
		log.Print("NOT Found RDS Credentials in Cache...")

		// If not found in cache, lookup SSM
		result, err := GetSecretValue(rdsSecretsArnName)

		if err != nil {
			return nil, errors.New(fmt.Sprintf("Failed to get param value: %s, Error: %v", rdsSecretsArnName, err))
		}

		rdsCredentialsString = result.SecretString

		log.Print("Saving RDS Credentials in Cache...")

		// If found in SSM, save in cache
		SetValue(rdsSecretsArnName, *rdsCredentialsString)
	}

	var rdsCredentialsObject RDSCredentials

	err := json.Unmarshal([]byte(*rdsCredentialsString), &rdsCredentialsObject)

	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to parse Json for RDS Credentials, Error: %v", err))
	}

	return &rdsCredentialsObject, nil
}
