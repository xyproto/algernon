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
	bolt "github.com/xyproto/permissionbolt"
	redis "github.com/xyproto/permissions2"
	mariadb "github.com/xyproto/permissionsql"
	"github.com/xyproto/pinterface"
	"github.com/xyproto/simpleredis"
	"github.com/xyproto/unzip"
	"github.com/yuin/gopher-lua"
)

const (
	versionString = "Algernon 0.71"
	description   = "HTTP/2 Web Server"
)

var (
	// List of filenames that should be displayed instead of a directory listing
	indexFilenames = []string{"index.lua", "index.html", "index.md", "index.txt", "index.amber"}

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
	var (
		err  error
		perm pinterface.IPermissions
	)

	// Use all CPUs. Soon to be the default for Go.
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
		fmt.Println(versionString)
		os.Exit(0)
	}

	// Console output
	fmt.Println(banner())

	// Dividing line between the banner and output from any of the configuration scripts
	if len(serverConfigurationFilenames) > 0 {
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
			case ".zip":
				// It might be a compressed Algernon application
				err := unzip.Extract(filename, serverTempDir)
				if err != nil {
					log.Fatalln(err)
				}
				// Use the directory where the file was extracted as the server directory
				serverDir = serverTempDir
				// If there is only one directory there, assume it's the directory of
				// the newly extracted ZIP file.
				if filenames := getFilenames(serverDir); len(filenames) == 1 {
					fullPath := filepath.Join(serverDir, filenames[0])
					if isDir(fullPath) {
						// Use this as the server directory instead
						serverDir = fullPath
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

	// If Bolt is to be used and no filename is given
	if useBolt && (boltFilename == "") {
		boltFilename = defaultBoltFilename
	}

	// Use one of the databases for the permission middleware,
	// then assign a name to dbName (used for the status output)
	dbName = ""
	if boltFilename != "" {
		// New permissions middleware, using a Bolt database
		perm, err = bolt.NewWithConf(boltFilename)
		if err != nil {
			log.Errorf("Could not use Bolt as database backend: %s", err)
		} else {
			dbName = "Bolt (" + boltFilename + ")"
		}
	}
	if dbName == "" && mariadbDSN != "" {
		// New permissions middleware, using a MariaDB/MySQL database
		perm, err = mariadb.NewWithDSN(mariadbDSN, mariadbDatabase)
		if err != nil {
			log.Errorf("Could not use MariaDB/MySQL as database backend: %s", err)
		} else {
			// The connection string may contain a password, so don't include it in the dbName
			dbName = "MariaDB/MySQL"
		}
	}
	if dbName == "" && mariadbDatabase != "" {
		// Given a database, but not a host, connect to localhost
		// New permissions middleware, using a MariaDB/MySQL database
		perm, err = mariadb.NewWithConf("test:@127.0.0.1/" + mariadbDatabase)
		if err != nil {
			log.Errorf("Could not use MariaDB/MySQL as database backend: %s", err)
		} else {
			// The connection string may contain a password, so don't include it in the dbName
			dbName = "MariaDB/MySQL"
		}
	}
	if dbName == "" {
		// New permissions middleware, using a Redis database
		if err := simpleredis.TestConnectionHost(redisAddr); err != nil {
			// Only warn when not in single file mode (too verbose)
			if !singleFileMode {
				log.Errorf("Could not use Redis as database backend: %s", err)
			}
		} else {
			perm = redis.NewWithRedisConf(redisDBindex, redisAddr)
			dbName = "Redis"
		}
	}
	if dbName == "" && boltFilename == "" {
		perm, err = bolt.NewWithConf(defaultBoltFilename)
		if err != nil {
			log.Errorf("Could not use Bolt as database backend: %s", err)
		} else {
			dbName = "Bolt (" + defaultBoltFilename + ")"
		}
	}
	if dbName == "" {
		// This may typically happen if Algernon is already running
		log.Fatalln("Could not find a usable database backend.")
	}

	// Lua LState pool
	luapool := &lStatePool{saved: make([]*lua.LState, 0, 4)}
	defer luapool.Shutdown()

	// Register HTTP handler functions
	registerHandlers(mux, serverDir, perm, luapool)

	if serverLogFile != "" {
		f, err := os.OpenFile(serverLogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Error("Could not log to", serverLogFile)
			fatalExit(err)
		}
		log.SetFormatter(&log.JSONFormatter{})
		log.SetOutput(f)
	}

	// Read server configuration script, if present.
	// The scripts may change global variables.
	for _, filename := range serverConfigurationFilenames {
		if exists(filename) {
			if err := runConfiguration(filename, perm, luapool); err != nil {
				log.Error("Could not use configuration script: " + filename)
				fatalExit(err)
			}
		}
	}

	// Set the values that has not been set by flags nor scripts (and can be set by both)
	ranServerReadyFunction := finalConfiguration(serverHost)

	// Dividing line between the banner and output from any of the configuration scripts
	if ranServerReadyFunction {
		fmt.Println("--------------------------------------- - - · ·")
	}

	// If we are not keeping the logs, reduce the verboseness
	http2.VerboseLogs = (serverHTTP2log != "/dev/null")

	// Direct the logging from the http2 package elsewhere
	f, err := os.Open(serverHTTP2log)
	defer f.Close()
	if err != nil {
		// Could not open the serverHTTP2log filename, try using another filename
		f, err = os.OpenFile("http2.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		defer f.Close()
		if err != nil {
			fatalExit(fmt.Errorf("Could not write to %s nor %s.", serverHTTP2log, "http2.log"))
		}
	}
	internallog.SetOutput(f)

	// Serve filesystem events in the background.
	// Used for reloading pages when the sources change.
	// Can also be used when serving a single file.
	if autoRefresh {
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

	if interactiveMode {
		go REPL(perm, luapool)
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
			log.Error(err)
			// If HTTPS failed (perhaps the key + cert are missing), serve
			// plain HTTP instead, by falling through to the next case.
		} else {
			// Don't fall through
			break
		}
		// Serve regular HTTP instead
		fallthrough
	default:
		log.Info("Serving HTTP on " + serverAddr)
		HTTPserver := newServerConfiguration(mux, false, serverAddr)
		// Start serving. Shut down gracefully at exit.
		if err := graceful.ListenAndServe(HTTPserver, shutdownTimeout); err != nil {
			// If we can't serve regular HTTP, give up
			fatalExit(err)
		}
	}
}
