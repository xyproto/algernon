package engine

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// The Host header ends up in a JavaScript string in the auto-refresh script
// and must not be able to break out of it.
func TestInsertAutoRefreshHostEscaping(t *testing.T) {
	ac := &Config{
		serverAddr:       ":3000",
		defaultEventPath: "/sse",
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "x'+alert(1)+'"

	page := string(ac.InsertAutoRefresh(req, []byte("<html><head></head><body></body></html>")))

	if strings.Contains(page, "alert(1)") {
		t.Errorf("The Host header was interpolated unescaped:\n%s", page)
	}
	if !strings.Contains(page, "localhost:3000") {
		t.Errorf("Expected a fallback to the server address, got:\n%s", page)
	}
}

func TestValidHostPort(t *testing.T) {
	tests := []struct {
		hostPort string
		want     bool
	}{
		{"localhost:3000", true},
		{"[::1]:3000", true},
		{"example.com", true},
		{"", false},
		{"x'+alert(1)+'", false},
		{"host:3000';x='", false},
		{"host\n:3000", false},
	}
	for _, tt := range tests {
		if got := validHostPort(tt.hostPort); got != tt.want {
			t.Errorf("validHostPort(%q) = %v, want %v", tt.hostPort, got, tt.want)
		}
	}
}
