package utils

import (
	"net/http"
)

// When a file is not found
func NoPage(filename, theme string) string {
	return MessagePage("Not found", "File not found: "+filename, theme)
}

// Return the domain of a request (up to ":", if any)
func GetDomain(req *http.Request) string {
	for i, r := range req.Host {
		if r == ':' {
			return req.Host[:i]
		}
	}
	return req.Host
}
