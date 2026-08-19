package engine

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// newTestProxyConfig points pathPrefix at the given backend URL
func newTestProxyConfig(t *testing.T, pathPrefix, endpoint string) *ReverseProxyConfig {
	t.Helper()
	u, err := url.Parse(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	rc := NewReverseProxyConfig()
	rc.Add(&ReverseProxy{PathPrefix: pathPrefix, Endpoint: *u})
	return rc
}

// A redirect from the backend must reach the client instead of being followed
// by the proxy itself.
func TestReverseProxyDoesNotFollowRedirects(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/redir" {
			http.Redirect(w, req, "/landed", http.StatusFound)
			return
		}
		io.WriteString(w, "landed")
	}))
	defer backend.Close()

	rp := newTestProxyConfig(t, "/api", backend.URL).FindMatchingReverseProxy("/api/redir")
	if rp == nil {
		t.Fatal("expected a matching reverse proxy")
	}

	rec := httptest.NewRecorder()
	rp.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/redir", nil))

	if rec.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if loc := rec.Header().Get("Location"); loc != "/landed" {
		t.Errorf("Location = %q, want %q", loc, "/landed")
	}
}

// The backend must be able to see the original client and scheme.
func TestReverseProxySetsForwardedHeaders(t *testing.T) {
	var gotFor, gotProto, gotHost, gotPath, gotHostHeader string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		gotFor = req.Header.Get("X-Forwarded-For")
		gotProto = req.Header.Get("X-Forwarded-Proto")
		gotHost = req.Header.Get("X-Forwarded-Host")
		gotHostHeader = req.Host
		gotPath = req.URL.Path
	}))
	defer backend.Close()

	rp := newTestProxyConfig(t, "/api", backend.URL).FindMatchingReverseProxy("/api/whoami")
	if rp == nil {
		t.Fatal("expected a matching reverse proxy")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/whoami", nil)
	req.Host = "example.com"
	req.RemoteAddr = "203.0.113.7:1234"
	rp.ServeHTTP(httptest.NewRecorder(), req)

	if gotFor != "203.0.113.7" {
		t.Errorf("X-Forwarded-For = %q, want %q", gotFor, "203.0.113.7")
	}
	if gotProto != "http" {
		t.Errorf("X-Forwarded-Proto = %q, want %q", gotProto, "http")
	}
	if gotHost != "example.com" {
		t.Errorf("X-Forwarded-Host = %q, want %q", gotHost, "example.com")
	}
	// The client's Host is forwarded, so the backend builds correct absolute URLs
	if gotHostHeader != "example.com" {
		t.Errorf("Host = %q, want %q", gotHostHeader, "example.com")
	}
	// The path prefix is stripped
	if gotPath != "/whoami" {
		t.Errorf("path = %q, want %q", gotPath, "/whoami")
	}
}

// Requesting exactly the prefix must produce "/" upstream, not an empty path.
func TestReverseProxyPrefixOnlyPath(t *testing.T) {
	var gotPath string
	backend := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		gotPath = req.URL.Path
	}))
	defer backend.Close()

	rp := newTestProxyConfig(t, "/api", backend.URL).FindMatchingReverseProxy("/api")
	if rp == nil {
		t.Fatal("expected a matching reverse proxy")
	}
	rp.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api", nil))

	if gotPath != "/" {
		t.Errorf("path = %q, want %q", gotPath, "/")
	}
}

// A base path on the endpoint is kept in front of the stripped path.
func TestReverseProxyEndpointBasePath(t *testing.T) {
	var gotPath string
	backend := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		gotPath = req.URL.Path
	}))
	defer backend.Close()

	rp := newTestProxyConfig(t, "/api", backend.URL+"/v2").FindMatchingReverseProxy("/api/things")
	if rp == nil {
		t.Fatal("expected a matching reverse proxy")
	}
	rp.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/things", nil))

	if gotPath != "/v2/things" {
		t.Errorf("path = %q, want %q", gotPath, "/v2/things")
	}
}

// An unreachable backend gives 502, not a partial 200.
func TestReverseProxyBadGateway(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	backendURL := backend.URL
	backend.Close() // nothing is listening any more

	rp := newTestProxyConfig(t, "/api", backendURL).FindMatchingReverseProxy("/api/gone")
	if rp == nil {
		t.Fatal("expected a matching reverse proxy")
	}

	rec := httptest.NewRecorder()
	rp.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/gone", nil))

	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
	if !strings.Contains(rec.Body.String(), "reverse proxy error") {
		t.Errorf("body = %q, want it to mention a reverse proxy error", rec.Body.String())
	}
}

// Multi-valued headers must keep all of their values.
func TestReverseProxyMultiValueHeaders(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Set-Cookie", "a=1")
		w.Header().Add("Set-Cookie", "b=2")
		w.Header().Add("Vary", "Accept")
		w.Header().Add("Vary", "Origin")
		io.WriteString(w, "ok")
	}))
	defer backend.Close()

	rp := newTestProxyConfig(t, "/api", backend.URL).FindMatchingReverseProxy("/api/x")
	if rp == nil {
		t.Fatal("expected a matching reverse proxy")
	}
	rec := httptest.NewRecorder()
	rp.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/x", nil))

	if got := rec.Result().Header.Values("Set-Cookie"); len(got) != 2 {
		t.Errorf("Set-Cookie = %v, want 2 values", got)
	}
	if got := rec.Result().Header.Values("Vary"); len(got) != 2 {
		t.Errorf("Vary = %v, want 2 values", got)
	}
}

// An upstream that closes right after the headers must give 502, not an empty 200.
func TestReverseProxyUpstreamDiesAfterHeaders(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "512")
		w.WriteHeader(http.StatusOK)
		// Drop the connection without sending the promised body
		conn, _, err := http.NewResponseController(w).Hijack()
		if err == nil {
			conn.Close()
		}
	}))
	defer backend.Close()

	rp := newTestProxyConfig(t, "/api", backend.URL).FindMatchingReverseProxy("/api/x")
	if rp == nil {
		t.Fatal("expected a matching reverse proxy")
	}
	rec := httptest.NewRecorder()
	rp.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/x", nil))

	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
}

// An event stream must not be peeked at, so that headers reach the client
// before the first event is produced.
func TestReverseProxyEventStreamNotPeeked(t *testing.T) {
	release := make(chan struct{})
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		http.NewResponseController(w).Flush()
		<-release // no body until the test says so
		io.WriteString(w, "data: hello\n\n")
	}))
	defer backend.Close()

	rp := newTestProxyConfig(t, "/api", backend.URL).FindMatchingReverseProxy("/api/stream")
	if rp == nil {
		t.Fatal("expected a matching reverse proxy")
	}
	front := httptest.NewServer(http.HandlerFunc(rp.ServeHTTP))
	defer front.Close()

	// http.Get returns as soon as the headers arrive, so this blocks if the
	// proxy waits for the first body byte
	type result struct {
		resp *http.Response
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		resp, err := http.Get(front.URL + "/api/stream")
		ch <- result{resp, err}
	}()

	select {
	case <-time.After(3 * time.Second):
		close(release)
		t.Fatal("proxying an event stream blocked waiting for the first byte")
	case got := <-ch:
		close(release)
		if got.err != nil {
			t.Fatal(got.err)
		}
		defer got.resp.Body.Close()
		if got.resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", got.resp.StatusCode, http.StatusOK)
		}
		body, _ := io.ReadAll(got.resp.Body)
		if !strings.Contains(string(body), "data: hello") {
			t.Errorf("body = %q, want it to contain the event", string(body))
		}
	}
}

// A response of unknown length may be a stream or a long poll, so it must not
// be peeked at either, even when it is not an event stream.
func TestReverseProxyChunkedResponseNotPeeked(t *testing.T) {
	release := make(chan struct{})
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// No Content-Length, so the response is chunked
		w.Header().Set(contentType, "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		http.NewResponseController(w).Flush()
		<-release // no body until the test says so
		io.WriteString(w, "{\"hello\":1}\n")
	}))
	defer backend.Close()

	rp := newTestProxyConfig(t, "/api", backend.URL).FindMatchingReverseProxy("/api/poll")
	if rp == nil {
		t.Fatal("expected a matching reverse proxy")
	}
	front := httptest.NewServer(http.HandlerFunc(rp.ServeHTTP))
	defer front.Close()

	type result struct {
		resp *http.Response
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		resp, err := http.Get(front.URL + "/api/poll")
		ch <- result{resp, err}
	}()

	select {
	case <-time.After(3 * time.Second):
		close(release)
		t.Fatal("proxying a long poll blocked waiting for the first byte")
	case got := <-ch:
		close(release)
		if got.err != nil {
			t.Fatal(got.err)
		}
		defer got.resp.Body.Close()
		if got.resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", got.resp.StatusCode, http.StatusOK)
		}
		body, _ := io.ReadAll(got.resp.Body)
		if !strings.Contains(string(body), "hello") {
			t.Errorf("body = %q, want it to contain the response", string(body))
		}
	}
}

// The query string survives proxying.
func TestReverseProxyKeepsQueryString(t *testing.T) {
	var gotQuery string
	backend := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		gotQuery = req.URL.RawQuery
	}))
	defer backend.Close()

	rp := newTestProxyConfig(t, "/api", backend.URL).FindMatchingReverseProxy("/api/search")
	if rp == nil {
		t.Fatal("expected a matching reverse proxy")
	}
	rp.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/search?q=hi&n=2", nil))

	if gotQuery != "q=hi&n=2" {
		t.Errorf("query = %q, want %q", gotQuery, "q=hi&n=2")
	}
}

// The longest matching prefix wins.
func TestReverseProxyLongestPrefixWins(t *testing.T) {
	rc := NewReverseProxyConfig()
	for _, pair := range [][2]string{{"/api", "http://one.invalid"}, {"/api/inner", "http://two.invalid"}} {
		u, err := url.Parse(pair[1])
		if err != nil {
			t.Fatal(err)
		}
		rc.Add(&ReverseProxy{PathPrefix: pair[0], Endpoint: *u})
	}
	rp := rc.FindMatchingReverseProxy("/api/inner/x")
	if rp == nil {
		t.Fatal("expected a matching reverse proxy")
	}
	if rp.PathPrefix != "/api/inner" {
		t.Errorf("PathPrefix = %q, want %q", rp.PathPrefix, "/api/inner")
	}
}
