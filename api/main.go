package main

import (
	"github.com/debojitroy/dynamo-to-rds/api/services"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/debojitroy/dynamo-to-rds/api/controller"
)

func setupRouter() *gin.Engine {
	r := gin.Default()

	r.POST("/v1/orders", func(c *gin.Context) {
		config := services.ConfigureAws()

		var orderRequest controller.OrderCreateRequest

		if c.Bind(&orderRequest) == nil {
			orderResponse, err := controller.CreateOrder(&config, &orderRequest)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			} else {
				c.JSON(http.StatusOK, orderResponse)
			}
		}
	})

	return r
}

func main() {
	r := setupRouter()
	// Listen and Server in 0.0.0.0:8080
	err := r.Run(":8080")
	if err != nil {
		panic("Failed to start server")
	}
}
