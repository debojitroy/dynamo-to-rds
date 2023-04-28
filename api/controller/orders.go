package controller

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/debojitroy/dynamo-to-rds/api/models"
)

type OrderCreateRequest struct {
	MerchantId string  `json:"merchant_id" binding:"required"`
	Amount     float64 `json:"amount" binding:"required"`
	Currency   string  `json:"currency" binding:"required"`
}

type OrderCreateResponse struct {
	OrderId string `json:"order_id" binding:"required"`
}

func CreateOrder(config *aws.Config, order *OrderCreateRequest) (OrderCreateResponse, error) {
	orderInput := models.NewOrderInput(order.MerchantId, order.Amount, order.Currency, "NEW")

	orderId, err := models.CreateOrder(config, orderInput)

	if err != nil {
		fmt.Printf("Failed to create order %v", err)
		return OrderCreateResponse{OrderId: ""}, err
	}

	return OrderCreateResponse{OrderId: orderId}, nil
}
