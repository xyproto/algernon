// Package sheepcounter is a byte counter that wraps a http.ResponseWriter
package sheepcounter

import (
	"bufio"
	"errors"
	"math"
	"net"
	"net/http"
)

var (
	errNoHijack        = errors.New("the wrapped http.ResponseWriter does not implement http.Hijacker")
	errOverflow        = errors.New("counter overflow")
	errConvertOverflow = errors.New("overflow when converting from uint64 to int64")
)

// SheepCounter is a struct that both wraps and implements a http.ResponseWriter
type SheepCounter struct {
	wrappedResponseWriter http.ResponseWriter
	bytesWritten          uint64
	overflow              bool
}

// New is an alias for the NewSheepCounter function
var New = NewSheepCounter

// NewSheepCounter creates a struct that wraps an existing http.ResponseWriter
func NewSheepCounter(w http.ResponseWriter) *SheepCounter {
	return &SheepCounter{w, 0, false}
}

// Header helps fulfill the http.ResponseWriter interface
func (sc *SheepCounter) Header() http.Header {
	return sc.wrappedResponseWriter.Header()
}

// Write helps fulfill the http.ResponseWriter interface, while also recording the written bytes
func (sc *SheepCounter) Write(data []byte) (int, error) {
	bytesWritten, err := sc.wrappedResponseWriter.Write(data)
	previousValue := sc.bytesWritten
	sc.bytesWritten += uint64(bytesWritten)
	// Detect uint64 overflow, but don't fail here
	sc.overflow = sc.bytesWritten < previousValue
	return bytesWritten, err
}

// WriteHeader helps fulfill the http.ResponseWriter interface
func (sc *SheepCounter) WriteHeader(statusCode int) {
	sc.wrappedResponseWriter.WriteHeader(statusCode)
}

// Counter returns the bytes written so far, as an int64. May return a negative
// number if the counter has overflown math.MaxInt64. Use Counter2() or
// UCounter2() instead if you wish to catch any overflow that may happen.
func (sc *SheepCounter) Counter() int64 {
	return int64(sc.bytesWritten)
}

// Counter2 returns the bytes written so far, as an int64.
// An error is returned if the counter has overflown.
func (sc *SheepCounter) Counter2() (int64, error) {
	if sc.overflow {
		return 0, errOverflow
	}
	if sc.bytesWritten >= math.MaxInt64 {
		return 0, errConvertOverflow
	}
	return int64(sc.bytesWritten), nil
}

// UCounter returns the bytes written so far, as an uint64
func (sc *SheepCounter) UCounter() uint64 {
	return sc.bytesWritten
}

// UCounter2 returns the bytes written so far, as an int64.
// An error is returned if the counter has overflown.
func (sc *SheepCounter) UCounter2() (uint64, error) {
	if sc.overflow {
		return 0, errOverflow
	}
	return sc.bytesWritten, nil
}

// Reset resets the written bytes counter
func (sc *SheepCounter) Reset() {
	sc.bytesWritten = 0
}

// Hijack helps fulfill the http.Hijacker interface
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

// Flush implements the http.Flusher interface and tries to flush the data.
func (sc *SheepCounter) Flush() {
	if flusher, ok := sc.wrappedResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
