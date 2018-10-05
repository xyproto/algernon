package sheepcounter

import (
	"net/http"
)

// SheepCounter is a struct that both wraps and implements a http.ResponseWriter
type SheepCounter struct {
	wrappedResponseWriter http.ResponseWriter
	bytesWritten          int64
}

// NewSheepCounter creates a struct that wraps an existing http.ResponseWriter
func NewSheepCounter(w http.ResponseWriter) *SheepCounter {
	return &SheepCounter{w, 0}
}

// Header helps fulfill the http.ResponseWriter interface
func (sc *SheepCounter) Header() http.Header {
	return sc.wrappedResponseWriter.Header()
}

// Write helps fulfill the http.ResponseWriter interface, while also recording the written bytes
func (sc *SheepCounter) Write(data []byte) (int, error) {
	bytesWritten, err := sc.wrappedResponseWriter.Write(data)
	sc.bytesWritten += int64(bytesWritten)
	return bytesWritten, err
}

// WriteHeader helps fulfill the http.ResponseWriter interface
func (sc *SheepCounter) WriteHeader(statusCode int) {
	sc.wrappedResponseWriter.WriteHeader(statusCode)
}

// Counter returns the bytes written so far
func (sc *SheepCounter) Counter() int64 {
	return sc.bytesWritten
}

// Reset resets the written bytes counter
func (sc *SheepCounter) Reset() {
	sc.bytesWritten = 0
}
