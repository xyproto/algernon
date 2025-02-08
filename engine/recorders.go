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
