package utils

import (
	"github.com/google/uuid"
)

// GenerateID generates a new UUID v4 string.
func GenerateID() string {
	return uuid.NewString()
}
