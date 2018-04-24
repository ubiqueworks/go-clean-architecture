package util

import (
	"github.com/google/uuid"
)

func NewUUID() string {
	requestId, _ := uuid.NewRandom()
	return requestId.String()
}
