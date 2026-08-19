package engine

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// GracefulServer is a http.Server that stops accepting new connections when
// SIGINT or SIGTERM is received, then gives the ongoing requests up to Timeout
// to finish before the remaining connections are closed.
type GracefulServer struct {
	*http.Server
	// ShutdownInitiated is called when a shutdown has been initiated, if set
	ShutdownInitiated func()
	// Timeout is how long the ongoing requests are given to finish
	Timeout     time.Duration
	signalsOnce sync.Once
	interrupted atomic.Bool
}

// Interrupted returns true if the server was stopped by a signal
func (gs *GracefulServer) Interrupted() bool {
	return gs.interrupted.Load()
}

// watchSignals stops the server when SIGINT or SIGTERM is received
func (gs *GracefulServer) watchSignals() {
	gs.signalsOnce.Do(func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-signals
			signal.Stop(signals)
			gs.interrupt()
		}()
	})
}

// interrupt marks the server as interrupted, calls ShutdownInitiated and then
// stops the server
func (gs *GracefulServer) interrupt() {
	gs.interrupted.Store(true)
	if gs.ShutdownInitiated != nil {
		gs.ShutdownInitiated()
	}
	gs.stop()
}

// stop lets the ongoing requests finish, for up to Timeout, then closes the
// remaining connections
func (gs *GracefulServer) stop() {
	ctx := context.Background()
	if gs.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, gs.Timeout)
		defer cancel()
	}
	if err := gs.Server.Shutdown(ctx); err != nil {
		// The requests took longer than Timeout
		gs.Server.Close()
	}
}

// ListenAndServe serves HTTP until the server is stopped
func (gs *GracefulServer) ListenAndServe() error {
	gs.watchSignals()
	return ignoreServerClosed(gs.Server.ListenAndServe())
}

// ListenAndServeTLS serves HTTPS, given a certificate and key file
func (gs *GracefulServer) ListenAndServeTLS(certFile, keyFile string) error {
	gs.watchSignals()
	return ignoreServerClosed(gs.Server.ListenAndServeTLS(certFile, keyFile))
}

// ListenAndServeTLSConfig serves HTTPS, given a TLS configuration.
// This is useful for CertMagic, which provides the certificates on demand.
func (gs *GracefulServer) ListenAndServeTLSConfig(tlsConfig *tls.Config) error {
	gs.watchSignals()
	gs.Server.TLSConfig = tlsConfig
	return ignoreServerClosed(gs.Server.ListenAndServeTLS("", ""))
}

// ignoreServerClosed returns nil if the server was stopped on purpose
func ignoreServerClosed(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
