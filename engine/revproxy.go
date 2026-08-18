package engine

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/utils"
)

// proxyTransport is shared by all reverse proxies, so that connections to the
// backends are pooled and kept alive across requests.
var proxyTransport http.RoundTripper = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	ForceAttemptHTTP2: true,
	MaxIdleConns:      256,
	// The default is 2, which re-dials the backend for almost every request
	MaxIdleConnsPerHost:   128,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: time.Second,
	ResponseHeaderTimeout: 60 * time.Second,
}

// peekedBody is a response body that is read through a bufio.Reader,
// while still closing the original body
type peekedBody struct {
	io.Reader
	io.Closer
}

// proxyRecorder records the status code and size of a proxied response.
// Unwrap lets http.ResponseController reach the underlying Flusher and Hijacker.
type proxyRecorder struct {
	http.ResponseWriter
	written int64
	status  int
}

func (pr *proxyRecorder) WriteHeader(status int) {
	if pr.status == 0 {
		pr.status = status
	}
	pr.ResponseWriter.WriteHeader(status)
}

func (pr *proxyRecorder) Write(p []byte) (int, error) {
	if pr.status == 0 {
		pr.status = http.StatusOK
	}
	n, err := pr.ResponseWriter.Write(p)
	pr.written += int64(n)
	return n, err
}

func (pr *proxyRecorder) Unwrap() http.ResponseWriter { return pr.ResponseWriter }

// ReverseProxy holds which path prefix (like "/api") should be sent where (like "http://localhost:8080")
type ReverseProxy struct {
	proxy      *httputil.ReverseProxy
	PathPrefix string
	Endpoint   url.URL
}

// ReverseProxyConfig holds several "path prefix --> URL" ReverseProxy structs,
// together with structures that speeds up the prefix matching.
type ReverseProxyConfig struct {
	proxyMatcher   utils.PrefixMatch
	prefix2rproxy  map[string]int
	ReverseProxies []ReverseProxy
}

// NewReverseProxyConfig creates a new and empty ReverseProxyConfig struct
func NewReverseProxyConfig() *ReverseProxyConfig {
	return &ReverseProxyConfig{}
}

// Add can add a ReverseProxy and will also (re-)initialize the internal proxy matcher
func (rc *ReverseProxyConfig) Add(rp *ReverseProxy) {
	rc.ReverseProxies = append(rc.ReverseProxies, *rp)
	rc.Init()
}

// newProxyHandler builds the httputil.ReverseProxy that does the actual proxying.
// The prefix is stripped from the request path and the endpoint path is prepended.
func newProxyHandler(pathPrefix string, endpoint url.URL) *httputil.ReverseProxy {
	basePath := strings.TrimSuffix(endpoint.Path, "/")
	endpointString := endpoint.String()
	return &httputil.ReverseProxy{
		Transport: proxyTransport,
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetXForwarded()
			// Forward the Host the client asked for, so that the backend
			// generates absolute URLs that point back at Algernon
			r.Out.Host = r.In.Host
			r.Out.URL.Scheme = endpoint.Scheme
			r.Out.URL.Host = endpoint.Host
			p := strings.TrimPrefix(r.In.URL.Path, pathPrefix)
			if !strings.HasPrefix(p, "/") {
				p = "/" + p
			}
			r.Out.URL.Path = basePath + p
			// Let EscapedPath re-derive the escaped form from Path
			r.Out.URL.RawPath = ""
		},
		ModifyResponse: func(res *http.Response) error {
			// Upgrades and event streams must not be read ahead of time:
			// the next bytes may only arrive once the client has spoken
			if res.StatusCode == http.StatusSwitchingProtocols ||
				strings.Contains(res.Header.Get(contentType), "text/event-stream") {
				return nil
			}
			// Peek one byte before sending the status: if upstream fails
			// before any body arrives, send 502 instead of a partial 200.
			// io.EOF is the legitimate empty-body case.
			br := bufio.NewReader(res.Body)
			if _, err := br.Peek(1); err != nil && err != io.EOF {
				return err
			}
			res.Body = &peekedBody{Reader: br, Closer: res.Body}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			logrus.Errorf("reverse proxy %s -> %s: %v\nPlease check your server config for AddReverseProxy calls.", pathPrefix, endpointString, err)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte("reverse proxy error, please check your server config for AddReverseProxy calls\n"))
		},
	}
}

// ServeHTTP proxies the given request to where the ReverseProxy points.
// Redirects, streaming responses and WebSocket upgrades are passed through.
func (rp *ReverseProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if rp.proxy == nil {
		rp.proxy = newProxyHandler(rp.PathPrefix, rp.Endpoint)
	}
	rp.proxy.ServeHTTP(w, req)
}

// DoProxyPass tries to proxy the given http.Request to where the ReverseProxy points
//
// Deprecated: use ServeHTTP instead. This follows redirects instead of passing them
// on, sets no X-Forwarded-* headers and can not handle WebSocket upgrades.
func (rp *ReverseProxy) DoProxyPass(req http.Request) (*http.Response, error) {
	client := &http.Client{Transport: proxyTransport}
	endpoint := rp.Endpoint
	req.RequestURI = ""
	req.URL.Path = strings.TrimPrefix(req.URL.Path, rp.PathPrefix)
	req.URL.Scheme = endpoint.Scheme
	req.URL.Host = endpoint.Host
	res, err := client.Do(&req)
	if err != nil {
		logrus.Errorf("reverse proxy error: %v\nPlease check your server config for AddReverseProxy calls.\n", err)
		return nil, err
	}
	return res, nil
}

// Init prepares the proxyMatcher and prefix2rproxy fields according to the ReverseProxy structs
func (rc *ReverseProxyConfig) Init() {
	keys := make([]string, 0, len(rc.ReverseProxies))
	rc.prefix2rproxy = make(map[string]int)
	for i := range rc.ReverseProxies {
		rp := &rc.ReverseProxies[i]
		if rp.proxy == nil {
			rp.proxy = newProxyHandler(rp.PathPrefix, rp.Endpoint)
		}
		keys = append(keys, rp.PathPrefix)
		rc.prefix2rproxy[rp.PathPrefix] = i
	}
	rc.proxyMatcher.Build(keys)
}

// FindMatchingReverseProxy checks if the given URL path should be proxied
func (rc *ReverseProxyConfig) FindMatchingReverseProxy(path string) *ReverseProxy {
	matches := rc.proxyMatcher.Match(path)
	if len(matches) == 0 {
		return nil
	}
	if len(matches) > 1 {
		logrus.Warnf("found more than one reverse proxy for `%s`: %+v. returning the longest", matches, path)
	}
	var match *ReverseProxy
	maxlen := 0
	for _, prefix := range matches {
		if len(prefix) > maxlen {
			maxlen = len(prefix)
			match = &rc.ReverseProxies[rc.prefix2rproxy[prefix]]
		}
	}
	return match
}
