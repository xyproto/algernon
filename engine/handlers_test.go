package engine

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xyproto/algernon/utils"
	"github.com/xyproto/datablock"
)

// TestReadAndLogErrorsEscapesFilename verifies that ReadAndLogErrors escapes
// the filename it echoes back when the file cannot be read, so a malicious
// path component cannot inject script tags into the response (CWE-79).
func TestReadAndLogErrorsEscapesFilename(t *testing.T) {
	ac := &Config{debugMode: true}
	ac.cache = datablock.NewFileCache(20000000, true, 64*utils.KiB, true, 0)

	w := httptest.NewRecorder()

	missing := "<script>alert(1)</script>/missing.txt"
	if _, err := ac.ReadAndLogErrors(w, missing, ".txt"); err == nil {
		t.Fatal("Expected error reading nonexistent file")
	}

	body := w.Body.String()
	if strings.Contains(body, "<script>alert(1)</script>") {
		t.Errorf("Unescaped filename leaked into response body:\n%s", body)
	}
	if !strings.Contains(body, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Errorf("Expected escaped filename in response body, got:\n%s", body)
	}
}
