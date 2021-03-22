package engine

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tylerb/graceful"
	"golang.org/x/net/http2"
)

// List of functions to run at shutdown
var (
	shutdownFunctions []func()
	mut               sync.Mutex
	completed         bool
)

// AtShutdown adds a function to the list of functions that will be ran at shutdown
func AtShutdown(shutdownFunction func()) {
	mut.Lock()
	defer mut.Unlock()
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
		mut.Lock()
		defer mut.Unlock()

		if completed {
			// The shutdown functions have already been called
			return
		}

		if ac.verboseMode {
			log.Info("Initiating shutdown")
		}

		// Call the shutdown functions in chronological order (FIFO)
		for _, shutdownFunction := range shutdownFunctions {
			shutdownFunction()
		}

		completed = true

		if ac.verboseMode {
			log.Info("Shutdown complete")
		}

		// Forced shutdown
		if gracefulServer != nil {
			if gracefulServer.Interrupted {
				//gracefulServer.Stop(forcedShutdownTimeout)
				ac.fatalExit(errors.New("Interrupted"))
			}
		}
		// TODO: To implement
		//if quicServer != nil {
		//fmt.Println("DEBUG: Has QUIC server at shutdown!")
		//}

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

	servingHTTPS := false
	servingHTTP := false

	// Goroutine that wait for a message to just serve regular HTTP, if needed
	go func() {
		<-justServeRegularHTTP // Wait for a message to just serve regular HTTP
		if strings.HasPrefix(ac.serverAddr, ":") {
			log.Info("Serving HTTP on http://localhost" + ac.serverAddr + "/")
		} else {
			log.Info("Serving HTTP on http://" + ac.serverAddr + "/")
		}
		mut.Lock()
		servingHTTP = true
		mut.Unlock()
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
			mut.Lock()
			servingHTTP = false
			mut.Unlock()
			// If we can't serve regular HTTP on port 80, give up
			ac.fatalExit(err)
		}
	}()

	// Decide which protocol to listen to
	switch {
	case ac.serveJustQUIC: // Just serve QUIC, but fallback to HTTP
		if strings.HasPrefix(ac.serverAddr, ":") {
			log.Info("Serving QUIC on https://localhost" + ac.serverAddr + "/")
		} else {
			log.Info("Serving QUIC on https://" + ac.serverAddr + "/")
		}
		mut.Lock()
		servingHTTPS = true
		mut.Unlock()
		// Start serving over QUIC
		go ac.ListenAndServeQUIC(mux, &mut, justServeRegularHTTP, &servingHTTPS)
	case ac.productionMode:
		// Listen for both HTTPS+HTTP/2 and HTTP requests, on different ports
		if ac.serverHost == "" {
			log.Info("Serving HTTP/2 on https://localhost/")
		} else {
			log.Info("Serving HTTP/2 on https://" + ac.serverHost + "/")
		}
		mut.Lock()
		servingHTTPS = true
		mut.Unlock()
		go func() {
			// Start serving. Shut down gracefully at exit.
			// Listen for HTTPS + HTTP/2 requests
			HTTPS2server := ac.NewGracefulServer(mux, true, ac.serverHost+":443")
			// Start serving. Shut down gracefully at exit.
			if err := HTTPS2server.ListenAndServeTLS(ac.serverCert, ac.serverKey); err != nil {
				mut.Lock()
				servingHTTPS = false
				mut.Unlock()
				log.Error(err)
			}
		}()
		if ac.serverHost == "" {
			log.Info("Serving HTTP on http://localhost/")
		} else {
			log.Info("Serving HTTP on http://" + ac.serverHost + "/")
		}
		mut.Lock()
		servingHTTP = true
		mut.Unlock()
		go func() {
			HTTPserver := ac.NewGracefulServer(mux, false, ac.serverHost+":80")
			if err := HTTPserver.ListenAndServe(); err != nil {
				mut.Lock()
				servingHTTP = false
				mut.Unlock()
				// If we can't serve regular HTTP on port 80, give up
				ac.fatalExit(err)
			}
		}()
	case ac.serveJustHTTP2: // It's unusual to serve HTTP/2 without HTTPS
		if strings.HasPrefix(ac.serverAddr, ":") {
			log.Warn("Serving HTTP/2 without HTTPS (not recommended!) on http://localhost" + ac.serverAddr + "/")
		} else {
			log.Warn("Serving HTTP/2 without HTTPS (not recommended!) on http://" + ac.serverAddr + "/")
		}
		mut.Lock()
		servingHTTPS = true
		mut.Unlock()
		go func() {
			// Listen for HTTP/2 requests
			HTTP2server := ac.NewGracefulServer(mux, true, ac.serverAddr)
			// Start serving. Shut down gracefully at exit.
			if err := HTTP2server.ListenAndServe(); err != nil {
				mut.Lock()
				servingHTTPS = false
				mut.Unlock()
				justServeRegularHTTP <- true
				log.Error(err)
			}
		}()
	case !(ac.serveJustHTTP2 || ac.serveJustHTTP):
		if strings.HasPrefix(ac.serverAddr, ":") {
			log.Info("Serving HTTP/2 on https://localhost" + ac.serverAddr + "/")
		} else {
			log.Info("Serving HTTP/2 on https://" + ac.serverAddr + "/")
		}
		mut.Lock()
		servingHTTPS = true
		mut.Unlock()
		// Listen for HTTPS + HTTP/2 requests
		HTTPS2server := ac.NewGracefulServer(mux, true, ac.serverAddr)
		// Start serving. Shut down gracefully at exit.
		go func() {
			if err := HTTPS2server.ListenAndServeTLS(ac.serverCert, ac.serverKey); err != nil {
				log.Errorf("%s. Not serving HTTP/2.", err)
				log.Info("Use the -t flag for serving regular HTTP.")
				mut.Lock()
				servingHTTPS = false
				mut.Unlock()
				// If HTTPS failed (perhaps the key + cert are missing),
				// serve plain HTTP instead
				justServeRegularHTTP <- true
			}
		}()
	default:
		mut.Lock()
		servingHTTP = true
		mut.Unlock()
		justServeRegularHTTP <- true
	}

	// Wait just a tiny bit
	time.Sleep(20 * time.Millisecond)

	ready <- true // Send a "ready" message to the REPL

	// Open the URL, if specified
	if ac.openURLAfterServing {
		// Open the https:// URL if both http:// and https:// are being served
		mut.Lock()
		if (!servingHTTP) && (!servingHTTPS) {
			ac.fatalExit(errors.New("serving neither over https:// nor over https://"))
		}
		httpsProtocol := servingHTTPS
		mut.Unlock()
		ac.OpenURL(ac.serverHost, ac.serverAddr, httpsProtocol)
	}

	<-done // Wait for a "done" message from the REPL (or just keep waiting)

	return nil // Done serving
}
