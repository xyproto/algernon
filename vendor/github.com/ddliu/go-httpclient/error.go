// Copyright 2014-2019 Liu Dong <ddliuhb@gmail.com>.
// Licensed under the MIT license.

package httpclient

import (
	"fmt"
	"net"
	"strings"
)

// Package errors
const (
	_ = iota
	ERR_DEFAULT
	ERR_TIMEOUT
	ERR_REDIRECT_POLICY
)

// Custom error
type Error struct {
	Code    int
	Message string
}

// Implement the error interface
func (this Error) Error() string {
	return fmt.Sprintf("httpclient #%d: %s", this.Code, this.Message)
}

func getErrorCode(err error) int {
	if err == nil {
		return 0
	}

	if e, ok := err.(*Error); ok {
		return e.Code
	}

	return ERR_DEFAULT
}

// Check a timeout error.
func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	// TODO: does not work?
	if e, ok := err.(net.Error); ok && e.Timeout() {
		return true
	}

	// TODO: make it reliable
	if strings.Contains(err.Error(), "timeout") {
		return true
	}

	return false
}

// Check a redirect error
func IsRedirectError(err error) bool {
	if err == nil {
		return false
	}

	// TODO: does not work?
	if getErrorCode(err) == ERR_REDIRECT_POLICY {
		return true
	}

	// TODO: make it reliable
	if strings.Contains(err.Error(), "redirect") {
		return true
	}

	return false
}
