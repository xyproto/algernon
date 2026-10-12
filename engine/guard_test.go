package engine

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeTestCert writes a self-signed certificate and key for 127.0.0.1
func writeTestCert(t *testing.T) (certFile, keyFile string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	certFile = filepath.Join(dir, "cert.pem")
	keyFile = filepath.Join(dir, "key.pem")
	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		t.Fatal(err)
	}
	return certFile, keyFile
}

// newGuardTestConfig returns a Config where /admin is protected, together
// with a mux that answers every request with "served"
func newGuardTestConfig() (*Config, *http.ServeMux) {
	ac := &Config{perm: &prefixPerm{rejectPrefixes: []string{"/admin", hmrUpdatePrefix + "private/"}}}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("served"))
	})
	mux.HandleFunc(hmrUpdatePrefix, func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("served"))
	})
	return ac, mux
}

// checkGuarded checks that protected paths are denied and that a public path
// is served, by the server at baseURL. If wantProto is not 0, the responses
// must use that major HTTP version.
func checkGuarded(t *testing.T, client *http.Client, baseURL string, wantProto int) {
	t.Helper()
	for path, want := range map[string]int{
		"/public":                     http.StatusOK,
		"/admin/x":                    http.StatusForbidden,
		"/%2fadmin/x":                 http.StatusForbidden,
		"//admin/x":                   http.StatusForbidden,
		"/ADMIN/x":                    http.StatusForbidden,
		"/public/../admin/x":          http.StatusForbidden,
		hmrUpdatePrefix + "private/x": http.StatusForbidden,
	} {
		req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Errorf("GET %s: %v", path, err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if wantProto != 0 && resp.ProtoMajor != wantProto {
			t.Errorf("GET %s: served over HTTP/%d, want HTTP/%d", path, resp.ProtoMajor, wantProto)
		}
		if want == http.StatusOK {
			if resp.StatusCode != http.StatusOK || string(body) != "served" {
				t.Errorf("GET %s: got %d %q, want 200 \"served\"", path, resp.StatusCode, body)
			}
		} else if string(body) == "served" {
			t.Errorf("GET %s: the protected handler was reached (status %d)", path, resp.StatusCode)
		}
	}
}

// TestServersAreGuarded starts the servers that NewGracefulServer creates and
// checks that they all go through guardHandler.
func TestServersAreGuarded(t *testing.T) {
	certFile, keyFile := writeTestCert(t)
	insecureTLS := &tls.Config{InsecureSkipVerify: true}

	for _, tc := range []struct {
		name      string
		https     bool
		http2     bool
		wantProto int
	}{
		{"HTTP", false, false, 1},
		{"HTTPS", true, false, 1},
		{"HTTP/2", true, true, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ac, mux := newGuardTestConfig()
			addr := findFreePort(t)
			gs := ac.NewGracefulServer(mux, tc.http2, addr)
			t.Cleanup(func() { gs.Server.Close() })
			go func() {
				if tc.https {
					gs.ListenAndServeTLS(certFile, keyFile)
				} else {
					gs.ListenAndServe()
				}
			}()
			waitForPort(t, addr)

			transport := &http.Transport{TLSClientConfig: insecureTLS, ForceAttemptHTTP2: tc.http2}
			if !tc.http2 {
				transport.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
			}
			defer transport.CloseIdleConnections()
			scheme := "http"
			if tc.https {
				scheme = "https"
			}
			checkGuarded(t, &http.Client{Transport: transport, Timeout: 5 * time.Second}, scheme+"://"+addr, tc.wantProto)
		})
	}
}
