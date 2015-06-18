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
	versionString = "Algernon 0.75"
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
	// Not compressing files in cache.
	// TODO: Implement and set to true
	cacheCompressed = false

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

	// If Bolt is to be used and no filename is given
	if useBolt && (boltFilename == "") {
		boltFilename = defaultBoltFilename
	}

	// Use one of the databases for the permission middleware,
	// then assign a name to dbName (used for the status output)
	// TODO Refactor into functions
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
		perm, err = mariadb.NewWithDSN(mariadbDSN, mariaDatabase)
		if err != nil {
			log.Errorf("Could not use MariaDB/MySQL as database backend: %s", err)
		} else {
			// The connection string may contain a password, so don't include it in the dbName
			dbName = "MariaDB/MySQL"
		}
	}
	if dbName == "" && mariaDatabase != "" {
		// Given a database, but not a host, connect to localhost
		// New permissions middleware, using a MariaDB/MySQL database
		perm, err = mariadb.NewWithConf("test:@127.0.0.1/" + mariaDatabase)
		if err != nil {
			if mariaDatabase != "" {
				log.Errorf("Could not use MariaDB/MySQL as database backend: %s", err)
			} else {
				log.Warnf("Could not use MariaDB/MySQL as database backend: %s", err)
			}
		} else {
			// The connection string may contain a password, so don't include it in the dbName
			dbName = "MariaDB/MySQL"
		}
	}
	if dbName == "" {
		// New permissions middleware, using a Redis database
		if err := simpleredis.TestConnectionHost(redisAddr); err != nil {
			// Only output an error when a Redis host other than the default host+port was specified
			if redisAddrSpecified {
				if singleFileMode {
					log.Warnf("Could not use Redis as database backend: %s", err)
				} else {
					log.Errorf("Could not use Redis as database backend: %s", err)
				}
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

	// Create a cache struct for reading files (contains functions that can
	// be used for reading files, also when caching is disabled).
	cache := newFileCache(cacheSize, cacheCompressed, cacheMaxEntitySize)

	// Register HTTP handler functions
	registerHandlers(mux, serverDir, perm, luapool, cache)

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

	// Read server configuration script, if present.
	// The scripts may change global variables.
	var ranConfigurationFilenames []string
	for _, filename := range serverConfigurationFilenames {
		if exists(filename) {
			if verboseMode {
				fmt.Println("Running configuration file: " + filename)
			}
			if err := runConfiguration(filename, perm, luapool); err != nil {
				log.Error("Could not use configuration script: " + filename)
				fatalExit(err)
			}
			ranConfigurationFilenames = append(ranConfigurationFilenames, filename)
		}
	}
	// Only keep the active ones. Used when outputting server information.
	serverConfigurationFilenames = ranConfigurationFilenames

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
