package engine

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestHMRUpdateHandlerRejectsTraversal verifies that path components of ".."
// are refused, so an attacker cannot escape the server root (CWE-22).
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

// TestHMRUpdateHandlerRejectsSymlinkEscape verifies that a symlink inside the
// server root pointing at a file outside the root is refused at read time.
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

// TestHMRUpdateHandlerMissingPath verifies that a request with no path after
// the HMR prefix is refused, leaving the existing contract intact.
func TestHMRUpdateHandlerMissingPath(t *testing.T) {
	ac := &Config{serverDirOrFilename: t.TempDir()}

	req := httptest.NewRequest("GET", hmrUpdatePrefix, nil)
	w := httptest.NewRecorder()
	ac.HMRUpdateHandler(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400 for missing path, got %d", w.Code)
	}
}
