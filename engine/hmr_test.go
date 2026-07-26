package engine

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Path components of ".." must not escape the server root (CWE-22).
func TestHMRUpdateHandlerRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	ac := &Config{serverDirOrFilename: root}

	req := httptest.NewRequest("GET", hmrUpdatePrefix+"../../etc/passwd", nil)
	w := httptest.NewRecorder()
	ac.HMRUpdateHandler(w, req)

	if w.Code != 403 {
		t.Errorf("Expected 403 for traversal attempt, got %d", w.Code)
	}
}

// A symlink in the server root pointing outside it must not be served.
func TestHMRUpdateHandlerRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Symlink creation typically requires elevated privileges on Windows")
	}

	outside := t.TempDir()
	secretPath := filepath.Join(outside, "secret.js")
	if err := os.WriteFile(secretPath, []byte("export const secret = 42;\n"), 0o600); err != nil {
		t.Fatalf("Failed writing secret file: %s", err)
	}

	root := t.TempDir()
	linkPath := filepath.Join(root, "leak.js")
	if err := os.Symlink(secretPath, linkPath); err != nil {
		t.Fatalf("Failed creating symlink: %s", err)
	}

	ac := &Config{serverDirOrFilename: root}

	req := httptest.NewRequest("GET", hmrUpdatePrefix+"leak.js", nil)
	w := httptest.NewRecorder()
	ac.HMRUpdateHandler(w, req)

	if w.Code != 403 {
		t.Errorf("Expected 403 for symlink escape, got %d (body: %s)", w.Code, w.Body.String())
	}
}

// A request with no path after the HMR prefix must return 400.
func TestHMRUpdateHandlerMissingPath(t *testing.T) {
	ac := &Config{serverDirOrFilename: t.TempDir()}

	req := httptest.NewRequest("GET", hmrUpdatePrefix, nil)
	w := httptest.NewRecorder()
	ac.HMRUpdateHandler(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400 for missing path, got %d", w.Code)
	}
}

// An absolute path supplied by the client must be refused.
func TestHMRUpdateHandlerRejectsAbsolutePath(t *testing.T) {
	ac := &Config{serverDirOrFilename: t.TempDir()}

	req := httptest.NewRequest("GET", hmrUpdatePrefix+"/etc/passwd", nil)
	w := httptest.NewRecorder()
	ac.HMRUpdateHandler(w, req)

	if w.Code != 403 {
		t.Errorf("Expected 403 for absolute path, got %d", w.Code)
	}
}

// Files that can not be hot-swapped must not be readable through this endpoint.
func TestHMRUpdateHandlerRejectsNonScriptFiles(t *testing.T) {
	root := t.TempDir()
	ac := &Config{serverDirOrFilename: root}

	for _, filename := range []string{".env", "secret.lua", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(root, filename), []byte("SECRET=1"), 0o644); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("GET", hmrUpdatePrefix+filename, nil)
		w := httptest.NewRecorder()
		ac.HMRUpdateHandler(w, req)

		if w.Code != 403 {
			t.Errorf("Expected 403 for %s, got %d", filename, w.Code)
		}
		if strings.Contains(w.Body.String(), "SECRET") {
			t.Errorf("The contents of %s were served", filename)
		}
	}
}
