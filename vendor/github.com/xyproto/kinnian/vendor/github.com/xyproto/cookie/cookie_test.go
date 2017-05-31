package cookie

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Secure cookie expects the value to be pipe deliminated
func TestSecureCookieFormat(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com/", nil)
	req.Header.Set("Cookie", "user=malformeduservalue")
	if _, ok := SecureCookie(req, "user", "foobar"); ok {
		t.Fatalf("TestSecureCookieFormat expected false instead %v", ok)
	}
}

func cookiePartInSlice(cookieSlice []string, part string) bool {
	for i := range cookieSlice {
		entry := strings.TrimSpace(cookieSlice[i])
		if entry == part {
			return true
		}
	}
	return false
}

type SetCookieTest struct {
	name     string
	value    string
	age      int64
	path     string
	secret   string
	secure   bool
	httponly bool
}

// SetSecureCookiePath can set cookies with different flags
func TestSetSecureCookiePath(t *testing.T) {
	tests := []SetCookieTest{
		{
			name:     "username",
			value:    "foo",
			age:      int64(3600),
			path:     "/",
			secret:   "12345secret",
			secure:   false,
			httponly: false,
		},
		{
			name:     "username",
			value:    "foo",
			age:      int64(3600),
			path:     "/",
			secret:   "12345secret",
			secure:   true,
			httponly: false,
		},
		{
			name:     "username",
			value:    "foo",
			age:      int64(3600),
			path:     "/",
			secret:   "12345secret",
			secure:   false,
			httponly: true,
		},
		{
			name:     "username",
			value:    "foo",
			age:      int64(3600),
			path:     "/",
			secret:   "12345secret",
			secure:   true,
			httponly: true,
		},
	}

	for _, test := range tests {
		handler := func(w http.ResponseWriter, r *http.Request) {
			SetSecureCookiePathWithFlags(w, test.name, test.value, test.age, test.path, test.secret, test.secure, test.httponly)
			io.WriteString(w, "testing..")
		}

		req := httptest.NewRequest("GET", "http://example.com/foo", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		resp := w.Result()
		cookie := resp.Header.Get("Set-Cookie")

		cookieSlice := strings.Split(cookie, ";")
		httpOnlyExists := cookiePartInSlice(cookieSlice, "HttpOnly")
		if httpOnlyExists != test.httponly {
			t.Logf("In test case: %+v\n", test)
			t.Fatalf("TestSetSecureCookiePath expected HttpOnly to be set, it isn't, Cookie: %s", cookie)
		}
		secureExists := cookiePartInSlice(cookieSlice, "Secure")
		if secureExists != test.secure {
			t.Logf("In test case: %+v\n", test)
			t.Fatalf("TestSetSecureCookiePath expected Secure to be set, it isn't, Cookie: %s", cookie)
		}
	}
}
