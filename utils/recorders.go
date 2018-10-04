package utils

import (
	"bytes"
	"net/http"
	"net/http/httptest"

	log "github.com/sirupsen/logrus"
)

// Write the contents of a ResponseRecorder to a ResponseWriter.
// Also flushes the recorder and returns the written bytes.
func WriteRecorder(w http.ResponseWriter, recorder *httptest.ResponseRecorder) int64 {
	for key, values := range recorder.HeaderMap {
		for _, value := range values {
			w.Header().Set(key, value)
		}
	}
	bytesWritten, err := recorder.Body.WriteTo(w)
	if err != nil {
		// Writing failed
		log.Error(err)
		return 0
	}
	recorder.Flush()
	return bytesWritten
}

// Discards the HTTP headers and return the recorder body as a string.
// Also flushes the recorder.
func RecorderToString(recorder *httptest.ResponseRecorder) string {
	var buf bytes.Buffer
	recorder.Body.WriteTo(&buf)
	recorder.Flush()
	return buf.String()
}
