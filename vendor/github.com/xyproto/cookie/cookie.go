package cookie

// Thanks to @hoisie / [web.go](https://github.com/hoisie/web) for several of these functions

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// Version number. Stable API within major version numbers.
	Version = 2.0

	// DefaultCookieTime represent how long login cookies should last, by deafault
	DefaultCookieTime = 24 * 3600 // 24 hours
)

// SecureCookie retrieves a secure cookie from a HTTP request
func SecureCookie(req *http.Request, name string, cookieSecret string) (string, bool) {
	for _, cookie := range req.Cookies() {
		if cookie.Name != name {
			continue
		}

		parts := strings.SplitN(cookie.Value, "|", 3)

		// fix potential out of range error
		if len(parts) != 3 {
			return "", false
		}

		val := parts[0]
		timestamp := parts[1]
		signature := parts[2]

		if Signature(cookieSecret, []byte(val), timestamp) != signature {
			return "", false
		}

		ts, _ := strconv.ParseInt(timestamp, 0, 64)
		if time.Now().Unix()-31*86400 > ts {
			return "", false
		}

		buf := bytes.NewBufferString(val)
		encoder := base64.NewDecoder(base64.StdEncoding, buf)

		res, _ := ioutil.ReadAll(encoder)
		return string(res), true
	}
	return "", false
}

/*SetCookiePathWithFlags sets a cookie with an explicit path.
 * age is the time-to-live, in seconds (0 means forever).
 *
 * The secure and httponly flags are documented here:
 * https://golang.org/pkg/net/http/#Cookie
 */
func SetCookiePathWithFlags(w http.ResponseWriter, name, value string, age int64, path string, secure, httponly bool) {
	var utctime time.Time
	if age == 0 {
		// 2^31 - 1 seconds (roughly 2038)
		utctime = time.Unix(2147483647, 0)
	} else {
		utctime = time.Unix(time.Now().Unix()+age, 0)
	}
	cookie := http.Cookie{Name: name, Value: value, Expires: utctime, Path: path, Secure: secure, HttpOnly: httponly}
	SetHeader(w, "Set-Cookie", cookie.String(), false)
}

// SetCookiePath sets a cookie with an explicit path.
// age is the time-to-live, in seconds (0 means forever).
func SetCookiePath(w http.ResponseWriter, name, value string, age int64, path string) {
	SetCookiePathWithFlags(w, name, value, age, path, false, false)
}

// ClearCookie clears the cookie with the given cookie name and a corresponding path.
// The cookie is cleared by setting the expiration date to 1970-01-01.
// Note that browsers *may* be configured to not delete the cookie.
func ClearCookie(w http.ResponseWriter, cookieName, cookiePath string) {
	ignoredContent := "SNUSNU" // random string
	cookie := fmt.Sprintf("%s=%s; path=%s; expires=Thu, 01 Jan 1970 00:00:00 GMT", cookieName, ignoredContent, cookiePath)
	SetHeader(w, "Set-Cookie", cookie, true)
}

/*SetSecureCookiePathWithFlags creates and sets a secure cookie with an explicit path.
 * age is the time-to-live, in seconds (0 means forever).
 *
 * The secure and httponly flags are documented here:
 * https://golang.org/pkg/net/http/#Cookie
 */
func SetSecureCookiePathWithFlags(w http.ResponseWriter, name, val string, age int64, path string, cookieSecret string, secure, httponly bool) {
	// base64 encode the value
	if len(cookieSecret) == 0 {
		log.Fatalln("Secret Key for secure cookies has not been set. Please use a non-empty secret.")
	}
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	encoder.Write([]byte(val))
	encoder.Close()
	vs := buf.String()
	vb := buf.Bytes()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	sig := Signature(cookieSecret, vb, timestamp)
	cookie := strings.Join([]string{vs, timestamp, sig}, "|")
	SetCookiePathWithFlags(w, name, cookie, age, path, secure, httponly)
}

// SetSecureCookiePath creates and sets a secure cookie with an explicit path.
// age is the time-to-live, in seconds (0 means forever).
func SetSecureCookiePath(w http.ResponseWriter, name, val string, age int64, path string, cookieSecret string) {
	SetSecureCookiePathWithFlags(w, name, val, age, path, cookieSecret, false, false)
}

// Signature retrieves the cookie signature
func Signature(key string, val []byte, timestamp string) string {
	hm := hmac.New(sha1.New, []byte(key))

	hm.Write(val)
	hm.Write([]byte(timestamp))

	hex := fmt.Sprintf("%02x", hm.Sum(nil))
	return hex
}

// SetHeader sets cookies in the HTTP header
func SetHeader(w http.ResponseWriter, hdr, val string, unique bool) {
	if unique {
		w.Header().Set(hdr, val)
	} else {
		w.Header().Add(hdr, val)
	}
}
