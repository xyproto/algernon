package engine

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
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

// Containment helper: only descendants of root should pass.
func TestPathContains(t *testing.T) {
	root := t.TempDir()
	cases := []struct {
		name string
		path string
		want bool
	}{
		{"root itself", root, true},
		{"direct child", filepath.Join(root, "a.js"), true},
		{"nested child", filepath.Join(root, "sub", "a.js"), true},
		{"sibling", filepath.Join(root, "..", "other"), false},
		{"ancestor", filepath.Dir(root), false},
	}
	for _, c := range cases {
		if got := pathContains(root, c.path); got != c.want {
			t.Errorf("%s: pathContains(%q,%q) = %v, want %v", c.name, root, c.path, got, c.want)
		}
	}
}
