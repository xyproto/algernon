package utils

import (
	"net"
	"net/http"
)

// GetDomain returns the host/domain of a request, handling both IPv4 and IPv6
func GetDomain(req *http.Request) string {
	host, _, err := net.SplitHostPort(req.Host)
	if err != nil {
		// No port in host, return as-is
		return req.Host
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
			return net.JoinHostPort(addr, "")
		}
		return addr
	}
	if host == "" {
		host = "localhost"
	}
	return net.JoinHostPort(host, port)
}
