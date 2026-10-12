package engine

import (
	"crypto/tls"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/caddyserver/certmagic"
)

// TestCertMagicServesWhileCertificatesFail checks that the HTTP and HTTPS
// listeners are started even if no certificate can be obtained, so that one
// failing domain does not stop the others from being served.
func TestCertMagicServesWhileCertificatesFail(t *testing.T) {
	portOf := func(addr string) int {
		_, p, _ := net.SplitHostPort(addr)
		n, _ := strconv.Atoi(p)
		return n
	}
	httpAddr, httpsAddr := findFreePort(t), findFreePort(t)

	oldHTTP, oldHTTPS := certmagic.HTTPPort, certmagic.HTTPSPort
	oldCA, oldStorage := certmagic.DefaultACME.CA, certmagic.Default.Storage
	t.Cleanup(func() {
		certmagic.HTTPPort, certmagic.HTTPSPort = oldHTTP, oldHTTPS
		certmagic.DefaultACME.CA, certmagic.Default.Storage = oldCA, oldStorage
	})
	certmagic.HTTPPort, certmagic.HTTPSPort = portOf(httpAddr), portOf(httpsAddr)
	// An ACME server that can not be reached, so that every certificate fails
	certmagic.DefaultACME.CA = "https://127.0.0.1:1/directory"
	certmagic.Default.Storage = &certmagic.FileStorage{Path: t.TempDir()}

	ac := &Config{}
	ac.serve.certMagicDomains = []string{"broken.example", "other.example"}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("served")) })
	go func() {
		if err := ac.serveCertMagic(mux); err != nil {
			t.Error(err)
		}
	}()
	waitForPort(t, httpAddr)
	waitForPort(t, httpsAddr)

	client := &http.Client{
		Timeout:       5 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	req, _ := http.NewRequest(http.MethodGet, "http://"+httpAddr+"/a?b=c", nil)
	req.Host = "other.example"
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMovedPermanently || resp.Header.Get("Location") != "https://other.example/a?b=c" {
		t.Errorf("got %d %q, want a redirect to https://other.example/a?b=c", resp.StatusCode, resp.Header.Get("Location"))
	}

	// There is no certificate yet, but the HTTPS listener must be up
	conn, err := net.DialTimeout("tcp", httpsAddr, 2*time.Second)
	if err != nil {
		t.Fatal("HTTPS is not being served:", err)
	}
	tlsConn := tls.Client(conn, &tls.Config{ServerName: "broken.example", InsecureSkipVerify: true})
	if err := tlsConn.Handshake(); err == nil {
		t.Error("expected the TLS handshake to fail, since there is no certificate")
	}
	tlsConn.Close()
}
