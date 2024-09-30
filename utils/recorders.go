package utils

import (
	"bytes"
	"net/http"
	"net/http/httptest"

	log "github.com/sirupsen/logrus"
)

// WriteRecorder writes to a ResponseWriter from a ResponseRecorder.
// Also flushes the recorder and returns how many bytes were written.
func WriteRecorder(w http.ResponseWriter, recorder *httptest.ResponseRecorder) int64 {
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
	if err != nil {
		// Writing failed
		log.Error(err)
		return 0
	}
	recorder.Flush()
	return bytesWritten
}

// RecorderToString discards the HTTP headers and return the recorder body as
// a string. Also flushes the recorder.
func RecorderToString(recorder *httptest.ResponseRecorder) string {
	var buf bytes.Buffer
	recorder.Body.WriteTo(&buf)
	recorder.Flush()
	return buf.String()
}
