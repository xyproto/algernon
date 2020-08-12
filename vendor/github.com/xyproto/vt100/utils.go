package vt100

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// For each element in a slice, apply the function f
func mapSB(sl []string, f func(string) byte) []byte {
	result := make([]byte, len(sl))
	for i, s := range sl {
		result[i] = f(s)
	}
	return result
}

// For each element in a slice, apply the function f
func mapBS(bl []byte, f func(byte) string) []string {
	result := make([]string, len(bl))
	for i, b := range bl {
		result[i] = f(b)
	}
	return result
}

// umin finds the smallest uint
func umin(a, b uint) uint {
	if a < b {
		return a
	}
	return b
}

// logf, for quick "printf-style" debugging
func logf(head string, tail ...interface{}) {
	tmpdir := os.Getenv("TMPDIR")
	if tmpdir == "" {
		tmpdir = "/tmp"
	}
	logfilename := filepath.Join(tmpdir, "o.log")
	f, err := os.OpenFile(logfilename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		f, err = os.Create(logfilename)
		if err != nil {
			log.Fatalln(err)
		}
	}
	f.WriteString(fmt.Sprintf(head, tail...))
	f.Sync()
	f.Close()
}

// Silence the "logf is unused" message by staticcheck
var _ = logf
