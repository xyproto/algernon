package engine

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPrettyErrorEscapesUserInput verifies that filename, error message and
// file contents are HTML-escaped, so that a malicious value cannot inject
// script tags into the rendered error page (CWE-79).
func TestPrettyErrorEscapesUserInput(t *testing.T) {
	ac := &Config{versionString: "Algernon test"}

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()

	filename := "<script>alert(1)</script>.lua"
	filebytes := []byte("print(\"<script>alert(2)</script>\")\n")
	errormessage := "boom: <script>alert(3)</script>"

	ac.PrettyError(w, req, filename, filebytes, errormessage, "lua")

	body := w.Body.String()

	for _, needle := range []string{
		"<script>alert(1)</script>",
		"<script>alert(2)</script>",
		"<script>alert(3)</script>",
	} {
		if strings.Contains(body, needle) {
			t.Errorf("Unescaped %q found in error page body", needle)
		}
	}

	if !strings.Contains(body, "&lt;script&gt;alert(3)&lt;/script&gt;") {
		t.Errorf("Expected escaped error message in body, got:\n%s", body)
	}
}

// TestPrettyErrorHighlightStillRenders verifies that the highlight markup
// around the offending Lua source line is preserved as raw HTML.
func TestPrettyErrorHighlightStillRenders(t *testing.T) {
	ac := &Config{versionString: "Algernon test"}

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()

	filebytes := []byte("line one\nbroken line\nline three\n")
	// Lua errors are formatted as "<file>:<line>: <message>"
	errormessage := "broken.lua:2: syntax error"

	ac.PrettyError(w, req, "broken.lua", filebytes, errormessage, "lua")

	body := w.Body.String()

	if !strings.Contains(body, preHighlight+"broken line"+postHighlight) {
		t.Errorf("Expected highlight markup around broken line, got:\n%s", body)
	}
}

// TestPrettyErrorNonLoopbackHidesDetails verifies that requests from non-loopback
// clients get a generic 500 response rather than the detailed error page.
func TestPrettyErrorNonLoopbackHidesDetails(t *testing.T) {
	ac := &Config{versionString: "Algernon test"}

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "8.8.8.8:1234"
	w := httptest.NewRecorder()

	ac.PrettyError(w, req, "secret.lua", []byte("secret = 42\n"), "boom", "lua")

	if w.Code != 500 {
		t.Errorf("Expected status 500 for non-loopback client, got %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "secret = 42") {
		t.Errorf("File contents leaked to non-loopback client")
	}
}
