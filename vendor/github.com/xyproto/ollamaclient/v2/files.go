package ollamaclient

import (
	"encoding/base64"
	"errors"
	"os"
)

// Base64EncodeFile reads in a file and returns a base64-encoded string
func Base64EncodeFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", errors.New(filePath + " is empty")
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	return encoded, nil
}
