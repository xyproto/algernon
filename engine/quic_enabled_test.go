//go:build linux || freebsd || windows || netbsd || darwin

package engine

import (
	"crypto/tls"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/quic-go/quic-go/http3"
)

// TestQUICServerIsGuarded starts the HTTP/3 server, which also serves HTTPS
// over TCP on the same port, and checks that both go through guardHandler.
func TestQUICServerIsGuarded(t *testing.T) {
	certFile, keyFile := writeTestCert(t)
	ac, mux := newGuardTestConfig()
	ac.serverAddr = findFreePort(t)
	ac.serve.serverCert = certFile
	ac.serve.serverKey = keyFile

	fallback := make(chan bool, 1)
	var servingHTTPS atomic.Bool
	servingHTTPS.Store(true)
	go ac.ListenAndServeQUIC(mux, fallback, &servingHTTPS)
	waitForPort(t, ac.serverAddr)

	insecureTLS := &tls.Config{InsecureSkipVerify: true}

	h3 := &http3.Transport{TLSClientConfig: insecureTLS}
	defer h3.Close()
	checkGuarded(t, &http.Client{Transport: h3, Timeout: 5 * time.Second}, "https://"+ac.serverAddr, 3)

	tcp := &http.Transport{TLSClientConfig: insecureTLS, ForceAttemptHTTP2: true}
	defer tcp.CloseIdleConnections()
	checkGuarded(t, &http.Client{Transport: tcp, Timeout: 5 * time.Second}, "https://"+ac.serverAddr, 0)

	select {
	case <-fallback:
		t.Error("the QUIC server gave up and asked for plain HTTP instead")
	default:
	}
}
