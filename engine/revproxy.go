package engine

import (
	"net/http"
	"net/url"

	"github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/utils"
)

// ReverseProxy holds which path prefix (like "/api") should be sent where (like "http://localhost:8080")
type ReverseProxy struct {
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

// DoProxyPass tries to proxy the given http.Request to where the ReverseProxy points
func (rp *ReverseProxy) DoProxyPass(req http.Request) (*http.Response, error) {
	client := &http.Client{}
	endpoint := rp.Endpoint
	req.RequestURI = ""
	req.URL.Path = req.URL.Path[len(rp.PathPrefix):]
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
	for i, rp := range rc.ReverseProxies {
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
