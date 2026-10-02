package engine

import (
	"bytes"
	"net/http"
	"net/http/httptest"
)

// WriteRecorder writes to a ResponseWriter from a ResponseRecorder.
// Also flushes the recorder and returns how many bytes were written.
func WriteRecorder(w http.ResponseWriter, recorder *httptest.ResponseRecorder) (int64, error) {
	for key, values := range recorder.Result().Header {
		for _, value := range values {
			w.Header().Set(key, value)
		}
	}
	if statusCode := recorder.Result().StatusCode; statusCode == 0 {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(statusCode)
	}
	bytesWritten, err := recorder.Body.WriteTo(w)
	if bytesWritten > 0 && err == nil {
		recorder.Flush()
	}
	return bytesWritten, err
}

// RecorderToString discards the HTTP headers and return the recorder body as
// a string. Also flushes the recorder.
func RecorderToString(recorder *httptest.ResponseRecorder) (string, error) {
	var buf bytes.Buffer
	n, err := recorder.Body.WriteTo(&buf)
	if n > 0 && err == nil {
		recorder.Flush()
	}
	return buf.String(), err
}

// forbiddenWriter sends 403 Forbidden instead of 200 OK, unless a status code
// has been set explicitly before the body is written
type forbiddenWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (fw *forbiddenWriter) WriteHeader(status int) {
	fw.wroteHeader = true
	fw.ResponseWriter.WriteHeader(status)
}

func (fw *forbiddenWriter) Write(p []byte) (int, error) {
	if !fw.wroteHeader {
		fw.WriteHeader(http.StatusForbidden)
	}
	return fw.ResponseWriter.Write(p)
}

func (fw *forbiddenWriter) Unwrap() http.ResponseWriter { return fw.ResponseWriter }

// finish sends 403 Forbidden if nothing has been written
func (fw *forbiddenWriter) finish() {
	if !fw.wroteHeader {
		fw.WriteHeader(http.StatusForbidden)
	}
}
