// HTTP/2 web server with built-in support for Lua, Markdown, GCSS, Amber and JSX.
package main

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	internallog "log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/bradfitz/http2"
	"github.com/tylerb/graceful"
	"github.com/xyproto/unzip"
	"github.com/yuin/gopher-lua"
)

const (
	versionString = "Algernon 0.81"
	description   = "HTTP/2 Web Server"
)

var (
	// For convenience. Set in the main function.
	serverHost      string
	dbName          string
	refreshDuration time.Duration
)

func newServerConfiguration(mux *http.ServeMux, http2support bool, addr string) *http.Server {
	// Server configuration
	s := &http.Server{
		Addr:    addr,
		Handler: mux,

		// The timeout values is also the maximum time it can take
		// for a complete page of Server-Sent Events (SSE).
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,

		MaxHeaderBytes: 1 << 20,
	}
	if http2support {
		// Enable HTTP/2 support
		http2.ConfigureServer(s, nil)
	}
	return s
}

func main() {
	var err error

	// TODO: Benchmark to see if runtime.NumCPU() * 4 scales better.
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Temporary directory that might be used for logging, databases or file extraction
	serverTempDir, err := ioutil.TempDir("", "algernon")
	if err != nil {
		log.Fatalln(err)
	}
	defer os.RemoveAll(serverTempDir)

	// Set several configuration variables, based on the given flags and arguments
	serverHost = handleFlags(serverTempDir)

	// Version
	if showVersion {
		if !quietMode {
			fmt.Println(versionString)
		}
		os.Exit(0)
	}

	// Console output
	if !quietMode {
		fmt.Println(banner())
	}

	// Dividing line between the banner and output from any of the configuration scripts
	if len(serverConfigurationFilenames) > 0 && !quietMode {
		fmt.Println("--------------------------------------- - - · ·")
	}

	// Request handlers
	mux := http.NewServeMux()

	// Read mime data from the system, if available
	initializeMime()

	// Check if the given directory really is a directory
	if !isDir(serverDir) {
		// Possibly a file
		filename := serverDir
		// Check if the file exists
		if exists(filename) {
			// Switch based on the lowercase filename extension
			switch strings.ToLower(filepath.Ext(filename)) {
			case ".md":
				// Serve the given Markdown file as a static HTTP server
				serveStaticFile(filename, defaultWebColonPort)
				return
			case ".zip", ".alg":
				// Assume this to be a compressed Algernon application
				err := unzip.Extract(filename, serverTempDir)
				if err != nil {
					log.Fatalln(err)
				}
				// Use the directory where the file was extracted as the server directory
				serverDir = serverTempDir
				// If there is only one directory there, assume it's the
				// directory of the newly extracted ZIP file.
				if filenames := getFilenames(serverDir); len(filenames) == 1 {
					fullPath := filepath.Join(serverDir, filenames[0])
					if isDir(fullPath) {
						// Use this as the server directory instead
						serverDir = fullPath
					}
				}
				// If there are server configuration files in the extracted
				// directory, register them.
				for _, filename := range serverConfigurationFilenames {
					configFilename := filepath.Join(serverDir, filename)
					if exists(configFilename) {
						serverConfigurationFilenames = append(serverConfigurationFilenames, configFilename)
					}
				}
				// Disregard all configuration files from the current directory
				// (filenames without a path separator), since we are serving a
				// ZIP file.
				for i, filename := range serverConfigurationFilenames {
					if strings.Count(filepath.ToSlash(filename), "/") == 0 {
						// Remove the filename from the slice
						serverConfigurationFilenames = append(serverConfigurationFilenames[:i], serverConfigurationFilenames[i+1:]...)
					}
				}
			default:
				singleFileMode = true
			}
		} else {
			fatalExit(errors.New("File does not exist: " + filename))
		}
	}

	// Make a few changes to the defaults if we are serving a single file
	if singleFileMode {
		debugMode = true
		serveJustHTTP = true
	}

	// Connect to a database and retrieve a Permissions struct
	perm := mustAquirePermissions()

	// Lua LState pool
	luapool := &lStatePool{saved: make([]*lua.LState, 0, 4)}
	defer luapool.Shutdown()

	// Create a cache struct for reading files (contains functions that can
	// be used for reading files, also when caching is disabled).
	cache := newFileCache(cacheSize, cacheCompression, cacheMaxEntitySize)

	if singleFileMode && filepath.Base(serverDir) == "server.lua" {
		luaServerFilename = serverDir
		serverDir = filepath.Dir(serverDir)
		singleFileMode = false
	}

	// Log to a file as JSON, if a log file has been specified
	if serverLogFile != "" {
		f, err := os.OpenFile(serverLogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, defaultPermissions)
		if err != nil {
			log.Error("Could not log to", serverLogFile)
			fatalExit(err)
		}
		log.SetFormatter(&log.JSONFormatter{})
		log.SetOutput(f)
	} else if quietMode {
		// If quiet mode is enabled and no log file has been specified, disable logging
		log.SetOutput(ioutil.Discard)
	}

	if quietMode {
		os.Stdout.Close()
		os.Stderr.Close()
	}

	// Read server configuration script, if present.
	// The scripts may change global variables.
	var ranConfigurationFilenames []string
	for _, filename := range serverConfigurationFilenames {
		if exists(filename) {
			if verboseMode {
				fmt.Println("Running configuration file: " + filename)
			}
			if err := runConfiguration(filename, perm, luapool, cache, mux, false); err != nil {
				log.Error("Could not use configuration script: " + filename)
				fatalExit(err)
			}
			ranConfigurationFilenames = append(ranConfigurationFilenames, filename)
		}
	}
	// Only keep the active ones. Used when outputting server information.
	serverConfigurationFilenames = ranConfigurationFilenames

	// Run the standalone Lua server, if specified
	if luaServerFilename != "" {
		// Run the Lua server file and set up handlers
		if verboseMode {
			fmt.Println("Running Lua Server File")
		}
		if err := runConfiguration(luaServerFilename, perm, luapool, cache, mux, true); err != nil {
			log.Error("Error in Lua server script: " + luaServerFilename)
			fatalExit(err)
		}
	} else {
		// Register HTTP handler functions
		registerHandlers(mux, "/", serverDir, perm, luapool, cache)
	}

	// Set the values that has not been set by flags nor scripts
	// (and can be set by both)
	ranServerReadyFunction := finalConfiguration(serverHost)

	// If no configuration files were being ran succesfully,
	// output basic server information.
	if len(serverConfigurationFilenames) == 0 {
		if !quietMode {
			fmt.Println(serverInfo())
		}
		ranServerReadyFunction = true
	}

	// Dividing line between the banner and output from any of the
	// configuration scripts. Marks the end of the configuration output.
	if ranServerReadyFunction && !quietMode {
		fmt.Println("--------------------------------------- - - · ·")
	}

	// If we are not writing internal logs to a file, reduce the verboseness
	http2.VerboseLogs = (internalLogFilename != os.DevNull)

	// Direct internal logging elsewhere
	internalLogFile, err := os.Open(internalLogFilename)
	defer internalLogFile.Close()
	if err != nil {
		// Could not open the internalLogFilename filename, try using another filename
		internalLogFile, err = os.OpenFile("internal.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, defaultPermissions)
		defer internalLogFile.Close()
		if err != nil {
			fatalExit(fmt.Errorf("Could not write to %s nor %s.", internalLogFilename, "internal.log"))
		}
	}
	internallog.SetOutput(internalLogFile)

	// Serve filesystem events in the background.
	// Used for reloading pages when the sources change.
	// Can also be used when serving a single file.
	if autoRefreshMode {
		refreshDuration, err = time.ParseDuration(eventRefresh)
		if err != nil {
			log.Warn(fmt.Sprintf("%s is an invalid duration. Using %s instead.", eventRefresh, defaultEventRefresh))
			// Ignore the error, since defaultEventRefresh is a constant and must be parseable
			refreshDuration, _ = time.ParseDuration(defaultEventRefresh)
		}
		if autoRefreshDir != "" {
			// Only watch the autoRefreshDir, recursively
			EventServer(eventAddr, defaultEventPath, autoRefreshDir, refreshDuration, "*")
		} else {
			// Watch everything in the server directory, recursively
			EventServer(eventAddr, defaultEventPath, serverDir, refreshDuration, "*")
		}
	}

	// The Lua REPL
	if !serverMode {
		go REPL(perm, luapool, cache)
	}

	// Timeout when closing down the server
	shutdownTimeout := 30 * time.Second

	// Decide which protocol to listen to
	switch {
	case productionMode:
		// Listen for both HTTPS+HTTP/2 and HTTP requests, on different ports
		go func() {
			log.Info("Serving HTTPS + HTTP/2 on " + serverHost + ":443")
			// Start serving. Shut down gracefully at exit.
			// Listen for HTTPS + HTTP/2 requests
			HTTPS2server := newServerConfiguration(mux, true, serverHost+":443")
			// Start serving. Shut down gracefully at exit.
			if err := graceful.ListenAndServeTLS(HTTPS2server, serverCert, serverKey, shutdownTimeout); err != nil {
				log.Error(err)
			}
		}()
		log.Info("Serving HTTP on " + serverHost + ":80")
		HTTPserver := newServerConfiguration(mux, false, serverHost+":80")
		if err := graceful.ListenAndServe(HTTPserver, shutdownTimeout); err != nil {
			// If we can't serve regular HTTP on port 80, give up
			fatalExit(err)
		}
	case serveJustHTTP2: // It's unusual to serve HTTP/2 withoutHTTPS
		log.Info("Serving HTTP/2 on " + serverAddr)
		// Listen for HTTP/2 requests
		HTTP2server := newServerConfiguration(mux, true, serverAddr)
		// Start serving. Shut down gracefully at exit.
		if err := graceful.ListenAndServe(HTTP2server, shutdownTimeout); err != nil {
			fatalExit(err)
		}
	case !(serveJustHTTP2 || serveJustHTTP):
		log.Info("Serving HTTPS + HTTP/2 on " + serverAddr)
		// Listen for HTTPS + HTTP/2 requests
		HTTPS2server := newServerConfiguration(mux, true, serverAddr)
		// Start serving. Shut down gracefully at exit.
		if err := graceful.ListenAndServeTLS(HTTPS2server, serverCert, serverKey, shutdownTimeout); err != nil {
			log.Warn("Could not serve HTTPS + HTTP/2 (" + err.Error() + ")")
			log.Info("Use the -h flag to serve HTTP only")
			// If HTTPS failed (perhaps the key + cert are missing), serve
			// plain HTTP instead, by falling through to the next case, below.
		} else {
			// Don't fall through
			break
		}
		// Serve regular HTTP instead
		fallthrough
	default:
		log.Info("Serving HTTP on " + serverAddr)
		HTTPserver := newServerConfiguration(mux, false, serverAddr)

		//if err := HTTPserver.ListenAndServe(); err != nil {

		// Start serving. Shut down gracefully at exit.
		if err := graceful.ListenAndServe(HTTPserver, shutdownTimeout); err != nil {
			// If we can't serve regular HTTP, give up
			fatalExit(err)
		}
	}
}
