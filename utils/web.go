package utils

import (
	"net/http"
)

// NoPage provides the same functionality as NoPage, but returns []byte
func NoPage(filename, theme string) []byte {
	return MessagePageBytes("Not found", []byte("File not found: "+filename), theme)
}

// GetDomain returns the domain of a request (up to ":", if any)
func GetDomain(req *http.Request) string {
	for i, r := range req.Host {
		if r == ':' {
			return req.Host[:i]
		}
	}
	return req.Host
}
