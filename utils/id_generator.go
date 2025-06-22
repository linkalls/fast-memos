package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateID generates a random string ID
func GenerateID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}
