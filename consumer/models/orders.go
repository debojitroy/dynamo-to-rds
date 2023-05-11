package models

import (
	"database/sql"
	"errors"
	"github.com/aws/aws-lambda-go/events"
	mysqlErrors "github.com/go-mysql/errors"
	"log"
	"time"
)

const maxRetries = 3

const insertNewRecordStatement string = `
	INSERT INTO tbl_orders (order_id, merchant_id, amount, currency, status, created_at, updated_at) 
	VALUES (?, ?, ?, ?, ?, ?, ?)
`

const updateRecordStatement string = `
	UPDATE tbl_orders SET amount = ?, currency = ?, status = ?, updated_at = ? where order_id = ?
`

type insertRecord struct {
	orderId    string
	merchantId string
	amount     float64
	currency   string
	status     string
	createdAt  int64
	updatedAt  int64
}

type updateRecord struct {
	orderId   string
	amount    float64
	currency  string
	status    string
	updatedAt int64
}

func convertEventToInsertRecord(change *events.DynamoDBStreamRecord) (*insertRecord, error) {
	orderId, ok := change.Keys["p_key"]

	if !ok {
		return nil, errors.New("order Id is missing")
	}

	merchantId, merchantOk := change.NewImage["merchant_id"]

	if !merchantOk {
		return nil, errors.New("merchant Id is missing")
	}

	amount, amountOk := change.NewImage["amount"]

	if !amountOk {
		return nil, errors.New("amount is missing")
	}

	amountVal, amountFloatErr := amount.Float()

	if amountFloatErr != nil {
		return nil, errors.New("amount is not floating value")
	}

	status, statusOk := change.NewImage["status"]

	if !statusOk {
		return nil, errors.New("status is missing")
	}

	currency, currencyOk := change.NewImage["currency"]

	if !currencyOk {
		return nil, errors.New("currency is missing")
	}

	createdAt, createdAtOk := change.NewImage["created_at"]

	if !createdAtOk {
		return nil, errors.New("created_at is missing")
	}

	createdAtVal, createdAtErr := createdAt.Integer()

	if createdAtErr != nil {
		return nil, errors.New("created_at is not integer value")
	}

	updatedAt, updatedOk := change.NewImage["updated_at"]

	if !updatedOk {
		return nil, errors.New("updated_at is missing")
	}

	updatedAtVal, updatedAtErr := updatedAt.Integer()

	if updatedAtErr != nil {
		return nil, errors.New("updated_at is not integer value")
	}

	insertRecordItem := insertRecord{
		orderId:    orderId.String(),
		merchantId: merchantId.String(),
		amount:     amountVal,
		status:     status.String(),
		currency:   currency.String(),
		createdAt:  createdAtVal,
		updatedAt:  updatedAtVal,
	}

	return &insertRecordItem, nil
}

func convertEventToUpdateRecord(change *events.DynamoDBStreamRecord) (*updateRecord, error) {
	orderId, ok := change.Keys["p_key"]

	if !ok {
		return nil, errors.New("order Id is missing")
	}

	amount, amountOk := change.NewImage["amount"]

	if !amountOk {
		return nil, errors.New("amount is missing")
	}

	amountVal, amountFloatErr := amount.Float()

	if amountFloatErr != nil {
		return nil, errors.New("amount is not floating value")
	}

	status, statusOk := change.NewImage["status"]

	if !statusOk {
		return nil, errors.New("status is missing")
	}

	currency, currencyOk := change.NewImage["currency"]

	if !currencyOk {
		return nil, errors.New("currency is missing")
	}

	updatedAt, updatedOk := change.NewImage["updated_at"]

	if !updatedOk {
		return nil, errors.New("updated_at is missing")
	}

	updatedAtVal, updatedAtErr := updatedAt.Integer()

	if updatedAtErr != nil {
		return nil, errors.New("updated_at is not integer value")
	}

	updateRecordItem := updateRecord{
		orderId:   orderId.String(),
		amount:    amountVal,
		status:    status.String(),
		currency:  currency.String(),
		updatedAt: updatedAtVal,
	}

	return &updateRecordItem, nil
}

func ProcessDynamoDbEvent(record *events.DynamoDBEventRecord, db *sql.DB) (bool, error) {
	log.Printf("Processing Event Id::%s, Event Name::%s, Event Version::%s, Event Source::%s, Event Source Arn::%s", record.EventID, record.EventName, record.EventVersion, record.EventSource, record.EventSourceArn)

	// Ignore expire and delete events
	// These will be TTL cleanups
	if record.EventName == "REMOVE" {
		log.Printf("Ignoring REMOVE events...")
		return true, nil
	}

	var tryInsert = false
	var tryUpdate = false

	// Prepare statement for inserting data
	stmtIns, insertStmtErr := db.Prepare(insertNewRecordStatement)
	if insertStmtErr != nil {
		log.Printf("Failed to create Insert Statement: %v\n", insertStmtErr)
		return false, insertStmtErr
	}
	defer func(stmtIns *sql.Stmt) {
		err := stmtIns.Close()
		if err != nil {
			log.Printf("Failed to close Open Statements")
		}
	}(stmtIns)

	// Prepare statement for updating data
	stmtUpdates, updateStmtErr := db.Prepare(updateRecordStatement)
	if updateStmtErr != nil {
		log.Printf("Failed to create Update Statement: %v\n", updateStmtErr)
		return false, updateStmtErr
	}

	defer func(stmtUpdates *sql.Stmt) {
		err := stmtUpdates.Close()
		if err != nil {
			log.Printf("Failed to close Open Statements")
		}
	}(stmtUpdates)

	for tryNo := 1; tryNo <= maxRetries; tryNo++ {
		if !tryUpdate && (record.EventName == "INSERT" || tryInsert) {
			log.Println("Trying to insert record")

			// Convert the record
			recordForInsert, conversionErr := convertEventToInsertRecord(&record.Change)

			if conversionErr != nil {
				log.Printf("Failed to parse changed record for insert: %v", conversionErr)
				return false, conversionErr
			}

			// Try Insert
			_, insertErr := stmtIns.Exec(recordForInsert.orderId,
				recordForInsert.merchantId,
				recordForInsert.amount,
				recordForInsert.currency,
				recordForInsert.status,
				time.Unix(recordForInsert.createdAt, 0).UTC().Format("2006-01-02 15:04:05"),
				time.Unix(recordForInsert.updatedAt, 0).UTC().Format("2006-01-02 15:04:05"))

			if insertErr != nil {
				if ok, sqlError := mysqlErrors.Error(insertErr); ok { // MySql error
					// Check if Primary Key violation
					if sqlError == mysqlErrors.ErrDupeKey {
						log.Printf("Primary Key Violation, trying update")
						tryUpdate = true
						continue
					}

					// Check if retry is possible
					if mysqlErrors.CanRetry(sqlError) {
						// Exponential Backup
						time.Sleep(time.Duration(tryNo*tryNo) * time.Second)
						continue
					}

					// Cannot proceed anymore
					return false, insertErr
				}
			}

			// Insert successful
			log.Println("Successfully inserted record")

			return true, nil
		}

		// Try to Update record
		log.Println("Trying to update record")

		// Convert the record
		recordForUpdate, conversionErr := convertEventToUpdateRecord(&record.Change)

		if conversionErr != nil {
			log.Printf("Failed to parse changed record for update: %v", conversionErr)
			return false, conversionErr
		}

		// Try Update
		updateResult, updateError := stmtUpdates.Exec(recordForUpdate.amount,
			recordForUpdate.currency,
			recordForUpdate.status,
			time.Unix(recordForUpdate.updatedAt, 0).UTC().Format("2006-01-02 15:04:05"),
			recordForUpdate.orderId)

		if updateError != nil {
			if ok, sqlError := mysqlErrors.Error(updateError); ok { // MySql error
				// Check if retry is possible
				if mysqlErrors.CanRetry(sqlError) {
					// Exponential Backup
					time.Sleep(time.Duration(tryNo*tryNo) * time.Second)
					continue
				}

				// Cannot proceed anymore
				return false, updateError
			}
		}

		// Check the number of updated rows
		rowsUpdated, fetchError := updateResult.RowsAffected()

		if fetchError != nil {
			log.Printf("Failed to fetch update results: %v", fetchError)
			return false, fetchError
		}

		// Check how many rows were updated
		// But don't check in case of insert failures
		if !tryUpdate && rowsUpdated == 0 {
			// No Rows were updated, try inserting the data
			log.Println("No Rows were updated, trying to insert")
			tryInsert = true
			continue
		}

		// Update Successful
		log.Println("Successfully updated record")
		return true, nil
	}

	return false, errors.New("failed to process record as max number of retries exceeded")
}
