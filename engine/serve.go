package engine

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/sirupsen/logrus"
	"github.com/tylerb/graceful"
	"github.com/xyproto/env/v2"
	"golang.org/x/net/http2"
)

// List of functions to run at shutdown
var (
	shutdownFunctions  []func()
	serverServingMutex sync.Mutex
	completed          bool
)

// AtShutdown adds a function to the list of functions that will be ran at shutdown
func AtShutdown(shutdownFunction func()) {
	serverServingMutex.Lock()
	defer serverServingMutex.Unlock()
	shutdownFunctions = append(shutdownFunctions, shutdownFunction)
}

// NewGracefulServer creates a new graceful server configuration
func (ac *Config) NewGracefulServer(mux *http.ServeMux, http2support bool, addr string) *graceful.Server {
	// Server configuration
	s := &http.Server{
		Addr:    addr,
		Handler: mux,

		// The timeout values is also the maximum time it can take
		// for a complete page of Server-Sent Events (SSE).
		ReadTimeout:  10 * time.Second,
		WriteTimeout: time.Duration(ac.writeTimeout) * time.Second,

		MaxHeaderBytes: 1 << 20,
	}
	if http2support {
		// Enable HTTP/2 support
		http2.ConfigureServer(s, nil)
	}
	gracefulServer := &graceful.Server{
		Server:  s,
		Timeout: ac.shutdownTimeout,
	}
	// Handle ctrl-c
	gracefulServer.ShutdownInitiated = ac.GenerateShutdownFunction(gracefulServer) // for investigating gracefulServer.Interrupted
	return gracefulServer
}

// GenerateShutdownFunction generates a function that will run the postponed
// shutdown functions.  Note that gracefulServer can be nil. It's only used for
// finding out if the server was interrupted (ctrl-c or killed, SIGINT/SIGTERM)
func (ac *Config) GenerateShutdownFunction(gracefulServer *graceful.Server) func() {
	return func() {
		if completed {
			// The shutdown functions have already been called
			return
		}

		if ac.verboseMode {
			logrus.Info("Initiating shutdown")
		}

		// Call the shutdown functions in chronological order (FIFO)
		for _, shutdownFunction := range shutdownFunctions {
			serverServingMutex.Lock()
			shutdownFunction()
			serverServingMutex.Unlock()
		}

		completed = true

		if ac.verboseMode {
			logrus.Info("Shutdown complete")
		}

		serverServingMutex.Lock()
		defer serverServingMutex.Unlock()

		// Forced shutdown
		if gracefulServer != nil {
			if gracefulServer.Interrupted {
				// gracefulServer.Stop(forcedShutdownTimeout)
				ac.fatalExit(errors.New("Interrupted"))
			}
		}

		// One final flush
		os.Stdout.Sync()
	}
}

// Serve HTTP, HTTP/2 and/or HTTPS. Returns an error if unable to serve, or nil when done serving.
func (ac *Config) Serve(mux *http.ServeMux, done, ready chan bool) error {
	// If we are not writing internal logs to a file, reduce the verbosity
	http2.VerboseLogs = (ac.internalLogFilename != os.DevNull)

	if ac.onlyLuaMode {
		ready <- true // Send a "ready" message to the REPL
		<-done        // Wait for a "done" message from the REPL (or just keep waiting)
		// Serve nothing
		return nil // Done
	}

	// Channel to wait and see if we should just serve regular HTTP instead
	justServeRegularHTTP := make(chan bool)

	var (
		servingHTTPS atomic.Bool
		servingHTTP  atomic.Bool
	)

	// Goroutine that wait for a message to just serve regular HTTP, if needed
	go func() {
		<-justServeRegularHTTP // Wait for a message to just serve regular HTTP
		if strings.HasPrefix(ac.serverAddr, ":") {
			logrus.Info("Serving HTTP on http://localhost" + ac.serverAddr + "/")
		} else {
			logrus.Info("Serving HTTP on http://" + ac.serverAddr + "/")
		}

		servingHTTP.Store(true)
		HTTPserver := ac.NewGracefulServer(mux, false, ac.serverAddr)
		// Open the URL before the serving has started, in a short delay
		if ac.openURLAfterServing && ac.luaServerFilename != "" {
			go func() {
				time.Sleep(delayBeforeLaunchingBrowser)
				ac.OpenURL(ac.serverHost, ac.serverAddr, false)
			}()
		}
		// Start serving. Shut down gracefully at exit.
		if err := HTTPserver.ListenAndServe(); err != nil {
			servingHTTP.Store(false)

			// If we can't serve regular HTTP on port 80, give up
			ac.fatalExit(err)
		}
	}()

	// Decide which protocol to listen to
	switch {
	case ac.useCertMagic:
		if len(ac.certMagicDomains) == 0 {
			logrus.Warnln("Found no directories looking like domains in the given directory.")
		} else if len(ac.certMagicDomains) == 1 {
			logrus.Infof("Serving one domain with CertMagic: %s", ac.certMagicDomains[0])
		} else {
			logrus.Infof("Serving %d domains with CertMagic: %s", len(ac.certMagicDomains), strings.Join(ac.certMagicDomains, ", "))
		}
		servingHTTPS.Store(true)
		// TODO: Look at "Advanced use" at https://github.com/caddyserver/certmagic#examples
		// Listen for HTTP and HTTPS requests, for specific domain(s)
		go func() {
			// If $XDG_CONFIG_DIR is not set, use $HOME.
			// If $HOME is not set, use $TMPDIR.
			// If $TMPDIR is not set, use /tmp.
			certStorageDir := env.StrAlt("XDG_CONFIG_DIR", "HOME", env.Str("TMPDIR", "/tmp"))

			defaultEmail := env.StrAlt("LOGNAME", "USER", "root") + "@localhost"
			if len(ac.certMagicDomains) > 0 {
				defaultEmail = "webmaster@" + ac.certMagicDomains[0]
			}

			certmagic.DefaultACME.Email = env.Str("EMAIL", defaultEmail)
			// TODO: Find a way for Algernon users to agree on this manually
			certmagic.DefaultACME.Agreed = true
			certmagic.Default.Storage = &certmagic.FileStorage{Path: certStorageDir}
			if err := certmagic.HTTPS(ac.certMagicDomains, mux); err != nil {
				servingHTTPS.Store(false)
				logrus.Error(err)
				// Don't serve HTTP if CertMagic fails, just quit
				// justServeRegularHTTP <- true
			}
		}()
	case ac.serveJustQUIC: // Just serve QUIC, but fallback to HTTP
		if strings.HasPrefix(ac.serverAddr, ":") {
			logrus.Info("Serving QUIC on https://localhost" + ac.serverAddr + "/")
		} else {
			logrus.Info("Serving QUIC on https://" + ac.serverAddr + "/")
		}
		servingHTTPS.Store(true)
		// Start serving over QUIC
		go ac.ListenAndServeQUIC(mux, justServeRegularHTTP, &servingHTTPS)
	case ac.productionMode:
		// Listen for both HTTPS+HTTP/2 and HTTP requests, on different ports
		if len(ac.serverHost) == 0 {
			logrus.Info("Serving HTTP/2 on https://localhost/")
		} else {
			logrus.Info("Serving HTTP/2 on https://" + ac.serverHost + "/")
		}
		servingHTTPS.Store(true)
		go func() {
			// Start serving. Shut down gracefully at exit.
			// Listen for HTTPS + HTTP/2 requests
			HTTPS2server := ac.NewGracefulServer(mux, true, ac.serverHost+":443")
			// Start serving. Shut down gracefully at exit.
			if err := HTTPS2server.ListenAndServeTLS(ac.serverCert, ac.serverKey); err != nil {
				servingHTTPS.Store(false)
				logrus.Error(err)
			}
		}()
		if len(ac.serverHost) == 0 {
			logrus.Info("Serving HTTP on http://localhost/")
		} else {
			logrus.Info("Serving HTTP on http://" + ac.serverHost + "/")
		}
		servingHTTP.Store(true)
		go func() {
			if ac.redirectHTTP {
				// Redirect HTTPS to HTTP
				redirectFunc := func(w http.ResponseWriter, req *http.Request) {
					http.Redirect(w, req, "https://"+req.Host+req.URL.String(), http.StatusMovedPermanently)
				}
				if err := http.ListenAndServe(ac.serverHost+":80", http.HandlerFunc(redirectFunc)); err != nil {
					servingHTTP.Store(false)
					// If we can't serve regular HTTP on port 80, give up
					ac.fatalExit(err)
				}
			} else {
				// Don't redirect, but serve the same contents as the HTTPS server as HTTP on port 80
				HTTPserver := ac.NewGracefulServer(mux, false, ac.serverHost+":80")
				if err := HTTPserver.ListenAndServe(); err != nil {
					servingHTTP.Store(false)
					// If we can't serve regular HTTP on port 80, give up
					ac.fatalExit(err)
				}
			}
		}()
	case ac.serveJustHTTP2: // It's unusual to serve HTTP/2 without HTTPS
		if strings.HasPrefix(ac.serverAddr, ":") {
			logrus.Warn("Serving HTTP/2 without HTTPS (not recommended!) on http://localhost" + ac.serverAddr + "/")
		} else {
			logrus.Warn("Serving HTTP/2 without HTTPS (not recommended!) on http://" + ac.serverAddr + "/")
		}
		servingHTTPS.Store(true)
		go func() {
			// Listen for HTTP/2 requests
			HTTP2server := ac.NewGracefulServer(mux, true, ac.serverAddr)
			// Start serving. Shut down gracefully at exit.
			if err := HTTP2server.ListenAndServe(); err != nil {
				servingHTTPS.Store(false)
				justServeRegularHTTP <- true
				logrus.Error(err)
			}
		}()
	case !ac.serveJustHTTP2 && !ac.serveJustHTTP:
		if strings.HasPrefix(ac.serverAddr, ":") {
			logrus.Info("Serving HTTP/2 on https://localhost" + ac.serverAddr + "/")
		} else {
			logrus.Info("Serving HTTP/2 on https://" + ac.serverAddr + "/")
		}
		servingHTTPS.Store(true)
		// Listen for HTTPS + HTTP/2 requests
		HTTPS2server := ac.NewGracefulServer(mux, true, ac.serverAddr)
		// Start serving. Shut down gracefully at exit.
		go func() {
			if err := HTTPS2server.ListenAndServeTLS(ac.serverCert, ac.serverKey); err != nil {
				logrus.Errorf("%s. Not serving HTTP/2.", err)
				logrus.Info("Use the -t flag for serving regular HTTP.")
				servingHTTPS.Store(false)
				// If HTTPS failed (perhaps the key + cert are missing),
				// serve plain HTTP instead
				justServeRegularHTTP <- true
			}
		}()
	default:
		servingHTTP.Store(true)
		justServeRegularHTTP <- true
	}

	// Wait just a tiny bit
	time.Sleep(20 * time.Millisecond)

	ready <- true // Send a "ready" message to the REPL

	// Open the URL, if specified
	if ac.openURLAfterServing {
		if !servingHTTP.Load() && !servingHTTPS.Load() {
			ac.fatalExit(errors.New("serving neither over http:// nor over https://"))
		}
		// Open the https:// URL if both http:// and https:// are being served
		ac.OpenURL(ac.serverHost, ac.serverAddr, servingHTTPS.Load())
	}

	<-done // Wait for a "done" message from the REPL (or just keep waiting)

	return nil // Done serving
}
