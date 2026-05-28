package engine

import (
	"context"
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
	"github.com/xyproto/algernon/utils"
	"github.com/xyproto/env/v2"
	"golang.org/x/net/http2"
)

// List of functions to run at shutdown
var (
	shutdownFunctions  []func()
	serverServingMutex sync.Mutex
	completed          atomic.Bool
)

// AtShutdown adds a function to the list of functions that will be ran at shutdown
func AtShutdown(shutdownFunction func()) {
	serverServingMutex.Lock()
	defer serverServingMutex.Unlock()
	shutdownFunctions = append(shutdownFunctions, shutdownFunction)
}

// NewGracefulServer creates a new graceful server configuration
func (ac *Config) NewGracefulServer(handler http.Handler, http2support bool, addr string) *graceful.Server {
	// Server configuration
	s := &http.Server{
		Addr:    addr,
		Handler: handler, // Use the provided http.Handler (e.g. httprouter)
		// The timeout values are also the maximum time it can take
		// for a complete page of Server-Sent Events (SSE).
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      time.Duration(ac.writeTimeout) * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	if http2support {
		// Enable HTTP/2 support
		http2.ConfigureServer(s, nil)
	}
	gracefulServer := &graceful.Server{
		Server:  s,
		Timeout: ac.shutdownTimeout,
	}
	// Handle ctrl-c: run the shutdown functions
	gracefulServer.ShutdownInitiated = ac.GenerateShutdownFunction(gracefulServer)
	return gracefulServer
}

// GenerateShutdownFunction generates a function that will run the postponed
// shutdown functions.  Note that gracefulServer can be nil. It's only used for
// finding out if the server was interrupted (ctrl-c or killed, SIGINT/SIGTERM)
func (ac *Config) GenerateShutdownFunction(gracefulServer *graceful.Server) func() {
	return func() {
		if completed.Load() {
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

		completed.Store(true)

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

// configureCertMagic sets CertMagic package-level defaults from ac.
// If $XDG_CONFIG_DIR is not set, uses $HOME, then $TMPDIR, then /tmp for cert storage.
func (ac *Config) configureCertMagic() {
	certStorageDir := env.StrAlt("XDG_CONFIG_DIR", "HOME", env.Str("TMPDIR", "/tmp"))
	defaultEmail := env.StrAlt("LOGNAME", "USER", "root") + "@localhost"
	if len(ac.serve.certMagicDomains) > 0 {
		defaultEmail = "webmaster@" + ac.serve.certMagicDomains[0]
	}
	certmagic.DefaultACME.Email = env.Str("EMAIL", defaultEmail)
	// TODO: Find a way for Algernon users to agree on this manually
	certmagic.DefaultACME.Agreed = true
	certmagic.Default.Storage = &certmagic.FileStorage{Path: certStorageDir}
	if ac.serve.useCertMagicStaging {
		certmagic.DefaultACME.CA = certmagic.LetsEncryptStagingCA
	}
}

// Serve HTTP, HTTP/2 and/or HTTPS. Returns an error if unable to serve, or nil when done serving.
func (ac *Config) Serve(handler http.Handler, done, ready chan bool) error {
	// If we are not writing internal logs to a file, reduce the verbosity
	http2.VerboseLogs = (ac.internalLogFilename != os.DevNull)

	if ac.onlyLuaMode {
		ready <- true // Send a "ready" message to the REPL
		<-done        // Wait for a "done" message from the REPL (or just keep waiting)
		// Serve nothing
		return nil // Done
	}

	// When --domain is specified, refuse to start if TLS is required but no certificates
	// are available and Let's Encrypt is not enabled.
	if ac.serverAddDomain && !ac.serve.useCertMagic {
		needsTLS := ac.serve.httpsAddr != ""
		for _, ps := range ac.serve.portSettings {
			if ps.TLS {
				needsTLS = true
				break
			}
		}
		if needsTLS {
			if _, err := os.Stat(ac.serve.serverCert); err != nil {
				ac.fatalExit(errors.New("TLS certificate not found: " + ac.serve.serverCert + " (use --cert, --letsencrypt, or remove --https-addr)"))
			}
			if _, err := os.Stat(ac.serve.serverKey); err != nil {
				ac.fatalExit(errors.New("TLS key not found: " + ac.serve.serverKey + " (use --key, --letsencrypt, or remove --https-addr)"))
			}
		}
	}

	// If explicit port settings are configured (from SetPorts in Lua), use them
	if len(ac.serve.portSettings) > 0 {
		return ac.servePortSettings(handler, done, ready)
	}

	// If --http-addr and/or --https-addr are set, convert to portSettings and use them
	if ac.serve.httpAddr != "" || ac.serve.httpsAddr != "" {
		if ac.serve.httpAddr != "" {
			ac.serve.portSettings = append(ac.serve.portSettings, PortSetting{Addr: ac.serve.httpAddr, Protocol: "http", TLS: false})
		}
		if ac.serve.httpsAddr != "" {
			ac.serve.portSettings = append(ac.serve.portSettings, PortSetting{Addr: ac.serve.httpsAddr, Protocol: "http2", TLS: true})
		}
		return ac.servePortSettings(handler, done, ready)
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
		logrus.Info("Serving HTTP on http://" + utils.HostPortToURL(ac.serverAddr) + "/")

		servingHTTP.Store(true)
		HTTPserver := ac.NewGracefulServer(handler, false, ac.serverAddr)
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
	case ac.serve.useCertMagic:
		if len(ac.serve.certMagicDomains) == 0 {
			logrus.Warnln("Found no directories looking like domains in the given directory.")
		} else if len(ac.serve.certMagicDomains) == 1 {
			logrus.Infof("Serving one domain with CertMagic: %s", ac.serve.certMagicDomains[0])
		} else {
			logrus.Infof("Serving %d domains with CertMagic: %s", len(ac.serve.certMagicDomains), strings.Join(ac.serve.certMagicDomains, ", "))
		}
		servingHTTPS.Store(true)
		// TODO: Look at "Advanced use" at https://github.com/caddyserver/certmagic#examples
		// Listen for HTTP and HTTPS requests, for specific domain(s)
		go func() {
			ac.configureCertMagic()
			if err := certmagic.HTTPS(ac.serve.certMagicDomains, handler); err != nil {
				servingHTTPS.Store(false)
				logrus.Error(err)
				// Don't serve HTTP if CertMagic fails, just quit
				// justServeRegularHTTP <- true
			}
		}()
	case ac.serveJustQUIC: // Just serve QUIC, but fallback to HTTP
		logrus.Info("Serving QUIC on https://" + utils.HostPortToURL(ac.serverAddr) + "/")
		servingHTTPS.Store(true)
		// Start serving over QUIC
		go ac.ListenAndServeQUIC(handler, justServeRegularHTTP, &servingHTTPS)
	case ac.productionMode:
		// Listen for both HTTPS+HTTP/2 and HTTP requests, on different ports
		logrus.Info("Serving HTTP/2 on https://" + utils.HostPortToURL(utils.JoinHostPort(ac.serverHost, ":443")) + "/")
		servingHTTPS.Store(true)
		go func() {
			// Start serving. Shut down gracefully at exit.
			// Listen for HTTPS + HTTP/2 requests
			HTTPS2server := ac.NewGracefulServer(handler, true, utils.JoinHostPort(ac.serverHost, ":443"))
			// Start serving. Shut down gracefully at exit.
			if err := HTTPS2server.ListenAndServeTLS(ac.serve.serverCert, ac.serve.serverKey); err != nil {
				servingHTTPS.Store(false)
				logrus.Error(err)
			}
		}()
		logrus.Info("Serving HTTP on http://" + utils.HostPortToURL(utils.JoinHostPort(ac.serverHost, ":80")) + "/")
		servingHTTP.Store(true)
		go func() {
			if ac.serve.redirectHTTP {
				// Redirect HTTP to HTTPS
				redirectFunc := func(w http.ResponseWriter, req *http.Request) {
					http.Redirect(w, req, "https://"+req.Host+req.URL.String(), http.StatusMovedPermanently)
				}
				if err := http.ListenAndServe(utils.JoinHostPort(ac.serverHost, ":80"), http.HandlerFunc(redirectFunc)); err != nil {
					servingHTTP.Store(false)
					// If we can't serve regular HTTP on port 80, give up
					ac.fatalExit(err)
				}
			} else {
				// Don't redirect, but serve the same contents as the HTTPS server as HTTP on port 80
				HTTPserver := ac.NewGracefulServer(handler, false, utils.JoinHostPort(ac.serverHost, ":80"))
				if err := HTTPserver.ListenAndServe(); err != nil {
					servingHTTP.Store(false)
					// If we can't serve regular HTTP on port 80, give up
					ac.fatalExit(err)
				}
			}
		}()
	case ac.serveJustHTTP2: // It's unusual to serve HTTP/2 without HTTPS
		logrus.Warn("Serving HTTP/2 without HTTPS (not recommended!) on http://" + utils.HostPortToURL(ac.serverAddr) + "/")
		servingHTTPS.Store(true)
		go func() {
			// Listen for HTTP/2 requests
			HTTP2server := ac.NewGracefulServer(handler, true, ac.serverAddr)
			// Start serving. Shut down gracefully at exit.
			if err := HTTP2server.ListenAndServe(); err != nil {
				servingHTTPS.Store(false)
				justServeRegularHTTP <- true
				logrus.Error(err)
			}
		}()
	case !ac.serveJustHTTP2 && !ac.serveJustHTTP:
		logrus.Info("Serving HTTP/2 on https://" + utils.HostPortToURL(ac.serverAddr) + "/")
		servingHTTPS.Store(true)
		// Listen for HTTPS + HTTP/2 requests
		HTTPS2server := ac.NewGracefulServer(handler, true, ac.serverAddr)
		// Start serving. Shut down gracefully at exit.
		go func() {
			if err := HTTPS2server.ListenAndServeTLS(ac.serve.serverCert, ac.serve.serverKey); err != nil {
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

// servePortSettings starts listeners based on the explicit portSettings configuration.
func (ac *Config) servePortSettings(handler http.Handler, done, ready chan bool) error {
	// When using CertMagic, set up managed TLS and the HTTP challenge handler.
	var (
		cmCfg      *certmagic.Config
		acmeIssuer *certmagic.ACMEIssuer
	)
	if ac.serve.useCertMagic {
		ac.configureCertMagic()
		cmCfg = certmagic.NewDefault()
		if err := cmCfg.ManageAsync(context.Background(), ac.serve.certMagicDomains); err != nil {
			logrus.Errorf("CertMagic setup failed, falling back to cert/key files: %v", err)
			cmCfg = nil
		} else if len(cmCfg.Issuers) > 0 {
			if am, ok := cmCfg.Issuers[0].(*certmagic.ACMEIssuer); ok {
				acmeIssuer = am
			}
		}
	}

	for _, ps := range ac.serve.portSettings {
		switch ps.Protocol {
		case "http":
			if ps.TLS {
				logrus.Infof("Serving HTTPS on https://%s/", utils.HostPortToURL(ps.Addr))
				if cmCfg != nil {
					go func() {
						tlsCfg := cmCfg.TLSConfig()
						tlsCfg.NextProtos = append([]string{"h2", "http/1.1"}, tlsCfg.NextProtos...)
						srv := ac.NewGracefulServer(handler, false, ps.Addr)
						if err := srv.ListenAndServeTLSConfig(tlsCfg); err != nil {
							logrus.Error(err)
						}
					}()
				} else {
					go func() {
						srv := ac.NewGracefulServer(handler, false, ps.Addr)
						if err := srv.ListenAndServeTLS(ac.serve.serverCert, ac.serve.serverKey); err != nil {
							logrus.Error(err)
						}
					}()
				}
			} else {
				logrus.Infof("Serving HTTP on http://%s/", utils.HostPortToURL(ps.Addr))
				go func() {
					if ac.serve.redirectHTTP && ac.serve.httpsAddr != "" {
						redirectFunc := func(w http.ResponseWriter, req *http.Request) {
							http.Redirect(w, req, "https://"+req.Host+req.URL.String(), http.StatusMovedPermanently)
						}
						var h http.Handler = http.HandlerFunc(redirectFunc)
						if acmeIssuer != nil {
							h = acmeIssuer.HTTPChallengeHandler(h)
						}
						if err := http.ListenAndServe(ps.Addr, h); err != nil {
							ac.fatalExit(err)
						}
					} else {
						h := handler
						if acmeIssuer != nil {
							h = acmeIssuer.HTTPChallengeHandler(h)
						}
						srv := ac.NewGracefulServer(h, false, ps.Addr)
						if err := srv.ListenAndServe(); err != nil {
							ac.fatalExit(err)
						}
					}
				}()
			}
		case "http2":
			if ps.TLS {
				logrus.Infof("Serving HTTP/2 on https://%s/", utils.HostPortToURL(ps.Addr))
				if cmCfg != nil {
					go func() {
						tlsCfg := cmCfg.TLSConfig()
						tlsCfg.NextProtos = append([]string{"h2", "http/1.1"}, tlsCfg.NextProtos...)
						srv := ac.NewGracefulServer(handler, true, ps.Addr)
						if err := srv.ListenAndServeTLSConfig(tlsCfg); err != nil {
							logrus.Error(err)
						}
					}()
				} else {
					go func() {
						srv := ac.NewGracefulServer(handler, true, ps.Addr)
						if err := srv.ListenAndServeTLS(ac.serve.serverCert, ac.serve.serverKey); err != nil {
							logrus.Error(err)
						}
					}()
				}
			} else {
				logrus.Infof("Serving HTTP/2 (h2c) on http://%s/", utils.HostPortToURL(ps.Addr))
				go func() {
					srv := ac.NewGracefulServer(handler, true, ps.Addr)
					if err := srv.ListenAndServe(); err != nil {
						ac.fatalExit(err)
					}
				}()
			}
		case "http3":
			if ps.TLS {
				logrus.Infof("Serving HTTP/3 (QUIC) on https://%s/", utils.HostPortToURL(ps.Addr))
			} else {
				logrus.Infof("Serving HTTP/3 (QUIC, no TLS) on %s/", utils.HostPortToURL(ps.Addr))
			}
			go ac.serveQUICPortSetting(handler, ps)
		case "event":
			logrus.Infof("Event server (SSE) on %s", utils.HostPortToURL(ps.Addr))
			ac.eventAddr = ps.Addr
			ac.separateEventServer = true
		default:
			logrus.Errorf("SetPorts: unknown protocol %q for %s", ps.Protocol, ps.Addr)
		}
	}

	// Wait just a tiny bit for listeners to start
	time.Sleep(20 * time.Millisecond)

	ready <- true // Send a "ready" message to the REPL

	// Open the URL, if specified
	if ac.openURLAfterServing {
		hasTLS := false
		firstAddr := ""
		for _, ps := range ac.serve.portSettings {
			if ps.Protocol == "event" {
				continue
			}
			if firstAddr == "" {
				firstAddr = ps.Addr
			}
			if ps.TLS {
				hasTLS = true
				break
			}
		}
		if firstAddr != "" {
			ac.OpenURL(ac.serverHost, firstAddr, hasTLS)
		}
	}

	<-done // Wait for a "done" message from the REPL (or just keep waiting)

	return nil // Done serving
}
