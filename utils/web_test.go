package utils

import (
	"net/http"
	"testing"
)

func TestGetDomain(t *testing.T) {
	tests := []struct {
		host string
		want string
	}{
		{"example.com:8080", "example.com"},
		{"example.com", "example.com"},
		{"[::1]:443", "localhost"},
		{"::1", "localhost"},
		{"127.0.0.1:3000", "localhost"},
		{"127.0.0.1", "localhost"},
		{"localhost:3000", "localhost"},
	}
	for _, tt := range tests {
		req := &http.Request{Host: tt.host}
		got := GetDomain(req)
		if got != tt.want {
			t.Errorf("GetDomain(Host=%q) = %q, want %q", tt.host, got, tt.want)
		}
	}
}

func TestJoinHostPort(t *testing.T) {
	tests := []struct {
		host      string
		colonPort string
		want      string
	}{
		{"", ":8080", ":8080"},
		{"localhost", ":8080", "localhost:8080"},
		{"::1", ":443", "[::1]:443"},
		{"example.com", ":3000", "example.com:3000"},
	}
	for _, tt := range tests {
		got := JoinHostPort(tt.host, tt.colonPort)
		if got != tt.want {
			t.Errorf("JoinHostPort(%q, %q) = %q, want %q", tt.host, tt.colonPort, got, tt.want)
		}
	}
}

func TestIsIPv6(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{"::1", true},
		{"fe80::1", true},
		{"127.0.0.1", false},
		{"localhost", false},
		{"", false},
	}
	for _, tt := range tests {
		got := IsIPv6(tt.host)
		if got != tt.want {
			t.Errorf("IsIPv6(%q) = %v, want %v", tt.host, got, tt.want)
		}
	}
}

func TestHostPortToURL(t *testing.T) {
	tests := []struct {
		addr string
		want string
	}{
		{":8080", "localhost:8080"},
		{"localhost:3000", "localhost:3000"},
		{"::1", "[::1]"},
		{"[::1]:443", "[::1]:443"},
		{"example.com:80", "example.com:80"},
	}
	for _, tt := range tests {
		got := HostPortToURL(tt.addr)
		if got != tt.want {
			t.Errorf("HostPortToURL(%q) = %q, want %q", tt.addr, got, tt.want)
		}
	}
}
