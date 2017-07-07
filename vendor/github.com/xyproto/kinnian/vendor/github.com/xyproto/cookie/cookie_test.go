package cookie

import (
	"net/http"
	"testing"
)

// Secure cookie expects the value to be pipe deliminated
func TestSecureCookieFormat(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com/", nil)
	req.Header.Set("Cookie", "user=malformeduservalue")
	_, ok := SecureCookie(req, "user", "foobar")

	if ok {
		t.Fatalf("TestSecureCookieFormat expected false instead %v", ok)
	}

}
