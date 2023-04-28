package utils

import "github.com/google/uuid"

func GenerateId() string {
	return uuid.NewString()
}
