package ollamaclient

import (
	"encoding/base64"
	"fmt"
	"os"
)

// Base64EncodeFile reads in a file and returns a base64-encoded string
func Base64EncodeFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", fmt.Errorf("%s contains 0 bytes", filePath)
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	return encoded, nil
}
