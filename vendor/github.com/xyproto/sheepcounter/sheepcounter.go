package sheepcounter

import (
	"bufio"
	"errors"
	"net"
	"net/http"
)

var (
	errNoHijack = errors.New("the wrapped http.ResponseWriter does not implement http.Hijacker")
)

// SheepCounter is a struct that both wraps and implements a http.ResponseWriter
type SheepCounter struct {
	wrappedResponseWriter http.ResponseWriter
	bytesWritten          int64
}

// NewSheepCounter is a deprecated alias for the New function
var NewSheepCounter = New

// New creates a struct that wraps an existing http.ResponseWriter
func New(w http.ResponseWriter) *SheepCounter {
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

// Hijack helps fulfill the http.Hijack interface
func (sc *SheepCounter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w, implementsHijacker := sc.wrappedResponseWriter.(http.Hijacker)
	if !implementsHijacker {
		return nil, nil, errNoHijack
	}
	return w.Hijack()
}

// ReponseWriter returns a pointer to the wrapped http.ResponseWriter
func (sc *SheepCounter) ResponseWriter() http.ResponseWriter {
	return sc.wrappedResponseWriter
}
