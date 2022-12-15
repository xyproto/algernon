package jpath

import (
	"bytes"
	"strings"
)

// lastpart returns the last part of a given JSON path
func lastpart(JSONpath string) string {
	if !strings.Contains(JSONpath, ".") {
		return JSONpath
	}
	parts := strings.Split(JSONpath, ".")
	return parts[len(parts)-1]
}

// badd can concatenate several byte slices
func badd(args ...[]byte) []byte {
	var buf bytes.Buffer
	for _, byteslice := range args {
		buf.Write(byteslice)
	}
	return buf.Bytes()
}
