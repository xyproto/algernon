package engine

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xyproto/pinterface/v2"
)

// prefixPerm rejects requests to paths that start with the given prefixes,
// the same way the permission packages do.
type prefixPerm struct {
	rejectPrefixes []string
}

func (p *prefixPerm) Rejected(_ http.ResponseWriter, req *http.Request) bool {
	lowerPath := strings.ToLower(req.URL.Path)
	for _, prefix := range p.rejectPrefixes {
		if strings.HasPrefix(lowerPath, prefix) {
			return true
		}
	}
	return false
}

func (p *prefixPerm) DenyFunction() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Permission denied.", http.StatusForbidden)
	}
}

func (p *prefixPerm) AddAdminPath(string)                                            {}
func (p *prefixPerm) AddPublicPath(string)                                           {}
func (p *prefixPerm) AddUserPath(string)                                             {}
func (p *prefixPerm) Clear()                                                         {}
func (p *prefixPerm) ServeHTTP(http.ResponseWriter, *http.Request, http.HandlerFunc) {}
func (p *prefixPerm) SetAdminPath([]string)                                          {}
func (p *prefixPerm) SetDenyFunction(http.HandlerFunc)                               {}
func (p *prefixPerm) SetPublicPath([]string)                                         {}
func (p *prefixPerm) SetUserPath([]string)                                           {}
func (p *prefixPerm) UserState() pinterface.IUserState                               { return nil }

// serveWithGuards builds the same middleware chain as NewGracefulServer, on
// top of a mux that records the path that the handlers saw.
func serveWithGuards(ac *Config, seen *string) http.Handler {
	mux := http.NewServeMux()
	record := func(w http.ResponseWriter, req *http.Request) {
		*seen = req.URL.Path
		w.WriteHeader(http.StatusOK)
	}
	mux.HandleFunc("/", record)
	mux.HandleFunc(hmrUpdatePrefix, record)
	return canonicalPathMiddleware(ac.permissionMiddleware(mux))
}

// A percent-encoded slash must not be able to sneak past the permission check.
func TestCanonicalPathMiddlewareEncodedSlash(t *testing.T) {
	ac := &Config{perm: &prefixPerm{rejectPrefixes: []string{"/admin"}}}

	tests := []struct {
		target string
		status int
		seen   string
	}{
		{"/admin/secret.lua", http.StatusForbidden, ""},
		{"/%2fadmin/secret.lua", http.StatusForbidden, ""},
		{"/%2e%2e/admin/secret.lua", http.StatusForbidden, ""},
		{"/%2f%2fadmin/secret.lua", http.StatusForbidden, ""},
		{"/ADMIN/secret.lua", http.StatusForbidden, ""},
		{"/index.html", http.StatusOK, "/index.html"},
	}
	for _, tt := range tests {
		var seen string
		w := httptest.NewRecorder()
		serveWithGuards(ac, &seen).ServeHTTP(w, httptest.NewRequest("GET", tt.target, nil))
		if w.Code != tt.status {
			t.Errorf("GET %s gave status %d, want %d", tt.target, w.Code, tt.status)
		}
		if seen != tt.seen {
			t.Errorf("GET %s reached the handler as %q, want %q", tt.target, seen, tt.seen)
		}
	}
}

// Routes registered outside of RegisterHandlers must be guarded as well.
func TestPermissionMiddlewareGuardsHMR(t *testing.T) {
	ac := &Config{perm: &prefixPerm{rejectPrefixes: []string{"/"}}}

	var seen string
	w := httptest.NewRecorder()
	serveWithGuards(ac, &seen).ServeHTTP(w, httptest.NewRequest("GET", hmrUpdatePrefix+"app.js", nil))

	if w.Code != http.StatusForbidden {
		t.Errorf("GET %sapp.js gave status %d, want %d", hmrUpdatePrefix, w.Code, http.StatusForbidden)
	}
	if seen != "" {
		t.Errorf("GET %sapp.js reached the handler as %q", hmrUpdatePrefix, seen)
	}
}

// Requests with control characters in the path are refused
func TestCanonicalPathMiddlewareControlCharacters(t *testing.T) {
	var seen string
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.URL.Path = "/a\x00b"
	serveWithGuards(&Config{}, &seen).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Path with a NUL byte gave status %d, want %d", w.Code, http.StatusBadRequest)
	}
}
