package utils

import (
	"net"
	"net/http"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"
)

// The IP addresses of this machine are cached for this long, since looking
// them up for every request is wasteful, but they may change over time
const localIPCacheDuration = time.Minute

var (
	localIPMut     sync.Mutex
	localIP        map[string]bool
	localIPUpdated time.Time
)

// CanonicalURLPath resolves ".." and "." elements, collapses repeated slashes
// and adds a leading slash. A trailing slash is kept, since it tells
// directories and files apart.
func CanonicalURLPath(p string) string {
	if runtime.GOOS == "windows" {
		// Backslash is a path separator here, and must not survive cleaning
		p = strings.ReplaceAll(p, "\\", "/")
	}
	cleaned := path.Clean("/" + p)
	if cleaned != "/" && strings.HasSuffix(p, "/") {
		cleaned += "/"
	}
	return cleaned
}

// GetHost returns the host of a request, handling both IPv4 and IPv6.
// Unlike GetDomain, it does not collapse loopback addresses, so URLs built
// from it stay same-origin with the request.
func GetHost(req *http.Request) string {
	host, _, err := net.SplitHostPort(req.Host)
	if err != nil {
		// No port in host, use as-is
		return req.Host
	}
	return host
}

// IsLocalIP checks if the given host is a loopback address or one of the IP
// addresses of this machine
func IsLocalIP(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	localIPMut.Lock()
	defer localIPMut.Unlock()
	if localIP == nil || time.Since(localIPUpdated) > localIPCacheDuration {
		// If the addresses can not be listed, cache an empty set, so that
		// a persistent failure does not mean one lookup per request
		addrs, _ := net.InterfaceAddrs()
		localIP = make(map[string]bool, len(addrs))
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				localIP[ipNet.IP.String()] = true
			}
		}
		localIPUpdated = time.Now()
	}
	return localIP[ip.String()]
}

// GetDomain returns the host/domain of a request, handling both IPv4 and IPv6.
// The loopback hosts 127.0.0.1 and ::1, and also the IP addresses of this
// machine, are collapsed to "localhost", so that a single localhost/ document
// root serves them all.
func GetDomain(req *http.Request) string {
	host := GetHost(req)
	if IsLocalIP(host) {
		return "localhost"
	}
	return host
}

// JoinHostPort combines a host and a colonPort (like ":8080") into an address.
// Handles IPv6 by adding brackets as needed. If host is empty, returns colonPort as-is.
func JoinHostPort(host, colonPort string) string {
	if host == "" {
		return colonPort
	}
	_, port, err := net.SplitHostPort(colonPort)
	if err != nil {
		// colonPort might just be a port number without colon
		port = colonPort
	}
	return net.JoinHostPort(host, port)
}

// IsIPv6 checks if the given string is an IPv6 address
func IsIPv6(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.To4() == nil
}

// HostPortToURL converts an address to a URL-friendly host:port string.
// Converts ":8080" to "localhost:8080" and ensures IPv6 addresses have brackets.
func HostPortToURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		// No port, check if it's an IPv6 that needs brackets
		if IsIPv6(addr) {
			return "[" + addr + "]"
		}
		return addr
	}
	if host == "" {
		host = "localhost"
	}
	return net.JoinHostPort(host, port)
}
