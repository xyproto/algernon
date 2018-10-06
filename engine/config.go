// Package engine provides the server configuration struct and several functions for serving files over HTTP
package engine

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	internallog "log"
	"net/http"
	"os"
	"path/filepath"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jvatic/goja-babel"
	log "github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/cachemode"
	"github.com/xyproto/algernon/lua/pool"
	"github.com/xyproto/algernon/platformdep"
	"github.com/xyproto/algernon/utils"
	"github.com/xyproto/datablock"
	"github.com/xyproto/mime"
	"github.com/xyproto/pinterface"
	"github.com/xyproto/recwatch"
	"github.com/xyproto/unzip"
)

const (
	// Version number. Stable API within major version numbers.
	Version = 2.0

	dividerLine = "················································································"
)

// Config is the main structure for the Algernon server.
// It contains all the state and settings.
type Config struct {

	// For convenience. Set in the main function.
	serverHost      string
	dbName          string
	refreshDuration time.Duration // for the auto-refresh feature
	shutdownTimeout time.Duration

	defaultWebColonPort       string
	defaultRedisColonPort     string
	defaultEventColonPort     string
	defaultEventRefresh       string
	defaultEventPath          string
	defaultLimit              int64
	defaultPermissions        os.FileMode
	defaultCacheSize          uint64        // 1 MiB
	defaultCacheMaxEntitySize uint64        // 64 KB
	defaultStatCacheRefresh   time.Duration // Refresh the stat cache, if the stat cache feature is enabled

	// Default rate limit, as a string
	defaultLimitString string

	// Store the request limit as a string for faster HTTP header creation later on
	limitRequestsString string

	// Default Bolt database file, for some operating systems
	defaultBoltFilename string

	// Default log file, for some operating systems
	defaultLogFile string

	// Default filename for a Lua script that provides data to a template
	defaultLuaDataFilename string

	// List of configuration filenames to check
	serverConfigurationFilenames []string

	// Configuration that is exposed to the server configuration script(s)
	serverDirOrFilename, serverAddr, serverCert, serverKey, serverConfScript, internalLogFilename, serverLogFile string

	// If only HTTP/2 or HTTP
	serveJustHTTP2, serveJustHTTP bool

	// If only QUIC
	serveJustQUIC bool

	// If only using the Lua REPL
	serveNothing bool

	// Configuration that may only be set in the server configuration script(s)
	serverAddrLua          string
	serverReadyFunctionLua func()

	// Server modes
	debugMode, verboseMode, productionMode, serverMode bool

	// For the Server-Sent Event (SSE) server
	eventAddr    string // Host and port to serve Server-Sent Events on
	eventRefresh string // The duration of an event cycle

	// Enable the event server and inject JavaScript to reload pages when sources change
	autoRefresh bool

	// If only watching a single directory recursively
	autoRefreshDir string

	// If serving a single file, like a lua script
	singleFileMode bool

	// Development mode aims to make it easy to get started
	devMode bool

	// Databases
	boltFilename       string
	useBolt            bool
	mariadbDSN         string // connection string
	mariaDatabase      string // database name
	postgresDSN        string // connection string
	postgresDatabase   string // database name
	redisAddr          string
	redisDBindex       int
	redisAddrSpecified bool

	limitRequests       int64 // rate limit to this many requests per client per second
	disableRateLimiting bool

	// Access logs
	commonAccessLogFilename   string // NCSA access log
	combinedAccessLogFilename string // CLF access log

	// For the version flag
	showVersion bool

	// Caching
	cacheSize             uint64
	cacheMode             cachemode.Setting
	cacheCompression      bool
	cacheMaxEntitySize    uint64
	cacheCompressionSpeed bool // Compression speed over compactness
	cacheMaxGivenDataSize uint64
	noCache               bool

	// HTTP headers
	noHeaders       bool
	stricterHeaders bool

	// Output
	quietMode bool
	noBanner  bool

	// If a single Lua file is provided, or Server() is used.
	luaServerFilename string

	// Used in the HTTP headers as "Server"
	serverHeaderName string

	// CPU profile filename
	profileCPU string

	// Memory profile filename
	profileMem string

	// Trace filename
	traceFilename string

	// Assume files will not be removed from the server directory while
	// Algernon is running. This allows caching of costly os.Stat calls.
	cacheFileStat bool

	// Look for files in the directory with the same name as the requested hostname
	serverAddDomain bool

	// Don't use a database backend. There will be loss of functionality.
	// TODO: Add a flag for this.
	useNoDatabase bool

	// For serving a directory with files over regular HTTP
	simpleMode bool

	// Open the URL after serving
	openURLAfterServing bool
	// Open the URL after serving, with a specific application
	openExecutable string

	// Quit after the first request?
	quitAfterFirstRequest bool

	// Markdown mode
	markdownMode bool

	// Theme for Markdown and error pages
	defaultTheme string

	// Workaround for rendering Pongo2 pages without concurrency issues
	pongomutex *sync.RWMutex

	// Temporary directory
	serverTempDir string

	// REPL
	ctrldTwice bool

	// State and caching
	perm    pinterface.IPermissions
	luapool *pool.LStatePool
	cache   *datablock.FileCache

	// Default program for opening files and URLs in the current OS
	defaultOpenExecutable string

	// Version and description
	versionString string
	description   string

	// Mime info
	mimereader *mime.Reader

	// For checking if files exists. FileStat cache.
	fs *datablock.FileStat

	// JSX rendering options
	jsxOptions map[string]interface{}

	// Convert JSX to HyperApp JS or React JS?
	hyperApp bool

	// Support clients like "curl" that downloads uncompressed by default
	curlSupport bool

	// Indicate if path prefixes like "/admin" should be cleared,
	// or if the default settings should be kept.
	clearDefaultPathPrefixes bool
}

// ErrVersion is returned when the initialization quits because all that is done
// is showing version information
var (
	ErrVersion  = errors.New("only showing version information")
	ErrDatabase = errors.New("could not find a usable database backend")
)

// New creates a new server configuration based using the default values
func New(versionString, description string) (*Config, error) {
	ac := &Config{
		curlSupport: true,

		shutdownTimeout: 10 * time.Second,

		defaultWebColonPort:       ":3000",
		defaultRedisColonPort:     ":6379",
		defaultEventColonPort:     ":5553",
		defaultEventRefresh:       "350ms",
		defaultEventPath:          "/fs",
		defaultLimit:              10,
		defaultPermissions:        0660,
		defaultCacheSize:          1 * utils.MiB,   // 1 MiB
		defaultCacheMaxEntitySize: 64 * utils.KiB,  // 64 KB
		defaultStatCacheRefresh:   time.Minute * 1, // Refresh the stat cache, if the stat cache feature is enabled

		// Default rate limit, as a string
		defaultLimitString: strconv.Itoa(10),

		// Default Bolt database file, for some operating systems
		defaultBoltFilename: "/tmp/algernon.db",

		// Default log file, for some operating systems
		defaultLogFile: "/tmp/algernon.log",

		// Default filename for a Lua script that provides data to a template
		defaultLuaDataFilename: "data.lua",

		// List of configuration filenames to check
		serverConfigurationFilenames: []string{"/etc/algernon/serverconf.lua", "/etc/algernon/server.lua"},

		// Compression speed over compactness
		cacheCompressionSpeed: true,

		// TODO: Make configurable
		// Maximum given file size for caching, 7 MiB
		cacheMaxGivenDataSize: 7 * utils.MiB,

		// Mutex for rendering Pongo2 pages
		pongomutex: &sync.RWMutex{},

		// Program for opening URLs
		defaultOpenExecutable: platformdep.DefaultOpenExecutable,

		// General information about Algernon
		versionString: versionString,
		description:   description,

		// JSX rendering options
		jsxOptions: map[string]interface{}{
			"plugins": []string{
				"transform-react-jsx",
				"transform-es2015-block-scoping",
			},
		},
	}
	if err := ac.initFilesAndCache(); err != nil {
		return nil, err
	}
	ac.initializeMime()
	ac.setupLogging()

	// File stat cache
	ac.fs = datablock.NewFileStat(ac.cacheFileStat, ac.defaultStatCacheRefresh)

	// JSX rendering pool
	babel.Init(8)

	return ac, nil
}

// SetFileStatCache can be used to set a different FileStat cache than the default one
func (ac *Config) SetFileStatCache(fs *datablock.FileStat) {
	ac.fs = fs
}

// Initialize a temporary directory, handle flags, output version and handle profiling
func (ac *Config) initFilesAndCache() error {
	// Temporary directory that might be used for logging, databases or file extraction
	serverTempDir, err := ioutil.TempDir("", "algernon")
	if err != nil {
		return err
	}
	ac.serverTempDir = serverTempDir

	// Set several configuration variables, based on the given flags and arguments
	ac.handleFlags(ac.serverTempDir)

	// Version
	if ac.showVersion {
		if !ac.quietMode {
			fmt.Println(ac.versionString)
		}
		return ErrVersion
	}

	// CPU profiling
	if ac.profileCPU != "" {
		f, errProfile := os.Create(ac.profileCPU)
		if errProfile != nil {
			return errProfile
		}
		go func() {
			log.Info("Profiling CPU usage")
			pprof.StartCPUProfile(f)
		}()
		AtShutdown(func() {
			pprof.StopCPUProfile()
			log.Info("Done profiling CPU usage")
			f.Close()
		})
	}

	// Memory profiling at server shutdown
	if ac.profileMem != "" {
		AtShutdown(func() {
			f, errProfile := os.Create(ac.profileMem)
			if errProfile != nil {
				// Fatal is okay here, since it's inside the anonymous shutdown function
				log.Fatalln(errProfile)
			}
			defer f.Close()
			log.Info("Saving heap profile to ", ac.profileMem)
			pprof.WriteHeapProfile(f)
		})
	}

	// Tracing
	if ac.traceFilename != "" {

		f, errTrace := os.Create(ac.traceFilename)
		if errTrace != nil {
			return errTrace
		}
		go func() {
			log.Info("Tracing")
			if err = trace.Start(f); err != nil {
				panic(err)
			}
		}()
		AtShutdown(func() {
			pprof.StopCPUProfile()
			trace.Stop()
			log.Info("Done tracing")
			f.Close()
		})
	}

	// Touch the common access log, if specified
	if ac.commonAccessLogFilename != "" {
		// Create if missing
		f, err := os.OpenFile(ac.commonAccessLogFilename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		f.Close()
	}
	// Touch the combined access log, if specified
	if ac.combinedAccessLogFilename != "" {
		// Create if missing
		f, err := os.OpenFile(ac.combinedAccessLogFilename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		f.Close()
	}

	// Create a cache struct for reading files (contains functions that can
	// be used for reading files, also when caching is disabled).
	// The final argument is for compressing with "fast" instead of "best".
	ac.cache = datablock.NewFileCache(ac.cacheSize, ac.cacheCompression, ac.cacheMaxEntitySize, ac.cacheCompressionSpeed, ac.cacheMaxGivenDataSize)
	return nil
}

func (ac *Config) setupLogging() {
	// Log to a file as JSON, if a log file has been specified
	if ac.serverLogFile != "" {
		f, errJSONLog := os.OpenFile(ac.serverLogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, ac.defaultPermissions)
		if errJSONLog != nil {
			log.Warn("Could not log to", ac.serverLogFile, ":", errJSONLog.Error())
		} else {
			// Log to the given log filename
			log.SetFormatter(&log.JSONFormatter{})
			log.SetOutput(f)
		}
	} else if ac.quietMode {
		// If quiet mode is enabled and no log file has been specified, disable logging
		log.SetOutput(ioutil.Discard)
	}
	// Close stdout and stderr if quite mode has been enabled
	if ac.quietMode {
		os.Stdout.Close()
		os.Stderr.Close()
	}
}

// Close removes the temporary directory
func (ac *Config) Close() {
	os.RemoveAll(ac.serverTempDir)
}

// Fatal exit
func (ac *Config) fatalExit(err error) {
	// Log to file, if a log file is used
	if ac.serverLogFile != "" {
		log.Error(err)
	}
	// Then switch to stderr and log the message there as well
	log.SetOutput(os.Stderr)
	// Use the standard formatter
	log.SetFormatter(&log.TextFormatter{})
	// Log and exit
	log.Fatalln(err)
}

// Abrupt exit
func (ac *Config) abruptExit(msg string) {
	// Log to file, if a log file is used
	if ac.serverLogFile != "" {
		log.Info(msg)
	}
	// Then switch to stderr and log the message there as well
	log.SetOutput(os.Stderr)
	// Use the standard formatter
	log.SetFormatter(&log.TextFormatter{})
	// Log and exit
	log.Info(msg)
	os.Exit(0)
}

// Quit after a short duration
func (ac *Config) quitSoon(msg string, soon time.Duration) {
	time.Sleep(soon)
	ac.abruptExit(msg)
}

// Return true of the given file type (extension) should be cached
func (ac *Config) shouldCache(ext string) bool {
	switch ac.cacheMode {
	case cachemode.On:
		return true
	case cachemode.Production, cachemode.Small:
		switch ext {
		case ".amber", ".lua", ".po2", ".tpl", ".pongo2":
			return false
		default:
			return true
		}
	case cachemode.Images:
		switch ext {
		case ".png", ".jpg", ".gif", ".svg", ".jpeg", ".ico", ".bmp", ".apng":
			return true
		default:
			return false
		}
	case cachemode.Off:
		return false
	case cachemode.Development, cachemode.Unset:
		fallthrough
	default:
		switch ext {
		case ".amber", ".lua", ".md", ".gcss", ".jsx", ".po2", ".tpl", ".pongo2", ".happ", ".js", ".scss":
			return false
		default:
			return true
		}
	}
}

// hasHandlers checks if the given filename contains "handle(" or "handle ("
func hasHandlers(fn string) bool {
	data, err := ioutil.ReadFile(fn)
	return err == nil && (bytes.Contains(data, []byte("handle(")) || bytes.Contains(data, []byte("handle (")))
}

// MustServe sets up a server with handlers
func (ac *Config) MustServe(mux *http.ServeMux) error {
	var err error

	defer ac.Close()

	// Output what we are attempting to access and serve
	if ac.verboseMode {
		log.Info("Accessing " + ac.serverDirOrFilename)
	}

	// Check if the given directory really is a directory
	if !ac.fs.IsDir(ac.serverDirOrFilename) {
		// It is not a directory
		serverFile := ac.serverDirOrFilename
		// Check if the file exists
		if ac.fs.Exists(serverFile) {
			if ac.markdownMode {
				// Serve the given Markdown file as a static HTTP server
				if serveErr := ac.ServeStaticFile(serverFile, ac.defaultWebColonPort); serveErr != nil {
					// Must serve
					ac.fatalExit(serveErr)
				}
				return nil
			}
			// Switch based on the lowercase filename extension
			switch strings.ToLower(filepath.Ext(serverFile)) {
			case ".md", ".markdown":
				// Serve the given Markdown file as a static HTTP server
				if serveErr := ac.ServeStaticFile(serverFile, ac.defaultWebColonPort); serveErr != nil {
					// Must serve
					ac.fatalExit(serveErr)
				}
				return nil
			case ".zip", ".alg":
				// Assume this to be a compressed Algernon application
				if extractErr := unzip.Extract(serverFile, ac.serverTempDir); extractErr != nil {
					return extractErr
				}
				// Use the directory where the file was extracted as the server directory
				ac.serverDirOrFilename = ac.serverTempDir
				// If there is only one directory there, assume it's the
				// directory of the newly extracted ZIP file.
				if filenames := utils.GetFilenames(ac.serverDirOrFilename); len(filenames) == 1 {
					fullPath := filepath.Join(ac.serverDirOrFilename, filenames[0])
					if ac.fs.IsDir(fullPath) {
						// Use this as the server directory instead
						ac.serverDirOrFilename = fullPath
					}
				}
				// If there are server configuration files in the extracted
				// directory, register them.
				for _, filename := range ac.serverConfigurationFilenames {
					configFilename := filepath.Join(ac.serverDirOrFilename, filename)
					if ac.fs.Exists(configFilename) {
						ac.serverConfigurationFilenames = append(ac.serverConfigurationFilenames, configFilename)
					}
				}
				// Disregard all configuration files from the current directory
				// (filenames without a path separator), since we are serving a
				// ZIP file.
				for i, filename := range ac.serverConfigurationFilenames {
					if strings.Count(filepath.ToSlash(filename), "/") == 0 {
						// Remove the filename from the slice
						ac.serverConfigurationFilenames = append(ac.serverConfigurationFilenames[:i], ac.serverConfigurationFilenames[i+1:]...)
					}
				}

			default:
				ac.singleFileMode = true
			}
		} else {
			return errors.New("File does not exist: " + serverFile)
		}
	}

	// Make a few changes to the defaults if we are serving a single file
	if ac.singleFileMode {
		ac.debugMode = true
		ac.serveJustHTTP = true
	}

	// Console output
	if !ac.quietMode && !ac.singleFileMode && !ac.simpleMode && !ac.noBanner {
		// Output a colorful ansi logo if a proper terminal is available
		fmt.Println(platformdep.Banner(ac.versionString, ac.description))
	}

	// Dividing line between the banner and output from any of the configuration scripts
	if len(ac.serverConfigurationFilenames) > 0 && !ac.quietMode && !ac.serveNothing {
		fmt.Println(dividerLine)
	}

	// Disable the database backend if the BoltDB filename is the /dev/null file (or OS equivalent)
	if ac.boltFilename == os.DevNull {
		ac.useNoDatabase = true
	}

	if !ac.useNoDatabase {
		// Connect to a database and retrieve a Permissions struct
		ac.perm, err = ac.DatabaseBackend()
		if err != nil {
			return ErrDatabase
		}
	}

	// Lua LState pool
	ac.luapool = pool.New()
	AtShutdown(func() {
		// TODO: Why not defer?
		ac.luapool.Shutdown()
	})

	// TODO: save repl history + close luapool + close logs ++ at shutdown

	if ac.singleFileMode && filepath.Ext(ac.serverDirOrFilename) == ".lua" {
		ac.luaServerFilename = ac.serverDirOrFilename
		if ac.luaServerFilename == "index.lua" || ac.luaServerFilename == "data.lua" {
			// Friendly message to new users
			if !hasHandlers(ac.luaServerFilename) {
				log.Warn("Found no handlers in " + ac.luaServerFilename)
				log.Info("How to implement \"Hello, World!\" in " + ac.luaServerFilename + " file:\n\nhandle(\"/\", function()\n  print(\"Hello, World!\")\nend)\n")
			}
		}
		ac.serverDirOrFilename = filepath.Dir(ac.serverDirOrFilename)
		// Make it possible to read other files from the Lua script
		ac.singleFileMode = false
	}

	// Read server configuration script, if present.
	// The scripts may change global variables.
	var ranConfigurationFilenames []string
	for _, filename := range ac.serverConfigurationFilenames {
		if ac.fs.Exists(filename) {
			if ac.verboseMode {
				log.Info("Running configuration file: " + filename)
			}
			withHandlerFunctions := true
			if errConf := ac.RunConfiguration(filename, mux, withHandlerFunctions); errConf != nil {
				if ac.perm != nil {
					log.Error("Could not use configuration script: " + filename)
					return errConf
				}
				if ac.verboseMode {
					log.Info("Skipping " + filename + " because the database backend is not in use.")
				}
			}
			ranConfigurationFilenames = append(ranConfigurationFilenames, filename)
		} else {
			if ac.verboseMode {
				log.Info("Looking for: " + filename)
			}
		}
	}
	// Only keep the active ones. Used when outputting server information.
	ac.serverConfigurationFilenames = ranConfigurationFilenames

	// Run the standalone Lua server, if specified
	if ac.luaServerFilename != "" {
		// Run the Lua server file and set up handlers
		if ac.verboseMode {
			fmt.Println("Running Lua Server File")
		}
		withHandlerFunctions := true
		if errLua := ac.RunConfiguration(ac.luaServerFilename, mux, withHandlerFunctions); errLua != nil {
			log.Errorf("Error in %s (interpreted as a server script):\n%s\n", ac.luaServerFilename, errLua)
			return errLua
		}
	} else {
		// Register HTTP handler functions
		ac.RegisterHandlers(mux, "/", ac.serverDirOrFilename, ac.serverAddDomain)
	}

	// Set the values that has not been set by flags nor scripts
	// (and can be set by both)
	ranServerReadyFunction := ac.finalConfiguration(ac.serverHost)

	// If no configuration files were being ran successfully,
	// output basic server information.
	if len(ac.serverConfigurationFilenames) == 0 {
		if !ac.quietMode && !ac.serveNothing {
			fmt.Println(ac.Info())
		}
		ranServerReadyFunction = true
	}

	// Dividing line between the banner and output from any of the
	// configuration scripts. Marks the end of the configuration output.
	if ranServerReadyFunction && !ac.quietMode && !ac.serveNothing {
		fmt.Println(dividerLine)
	}

	// Direct internal logging elsewhere
	internalLogFile, err := os.Open(ac.internalLogFilename)
	if err != nil {
		// Could not open the internalLogFilename filename, try using another filename
		internalLogFile, err = os.OpenFile("internal.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, ac.defaultPermissions)
		AtShutdown(func() {
			// TODO This one is is special and should be closed after the other shutdown functions.
			//      Set up a "done" channel instead of sleeping.
			time.Sleep(100 * time.Millisecond)
			internalLogFile.Close()
		})
		if err != nil {
			ac.fatalExit(fmt.Errorf("Error: could not write to %s nor %s", ac.internalLogFilename, "internal.log"))
		}
	}
	defer internalLogFile.Close()
	internallog.SetOutput(internalLogFile)

	// Serve filesystem events in the background.
	// Used for reloading pages when the sources change.
	// Can also be used when serving a single file.
	if ac.autoRefresh {
		ac.refreshDuration, err = time.ParseDuration(ac.eventRefresh)
		if err != nil {
			log.Warn(fmt.Sprintf("%s is an invalid duration. Using %s instead.", ac.eventRefresh, ac.defaultEventRefresh))
			// Ignore the error, since defaultEventRefresh is a constant and must be parseable
			ac.refreshDuration, _ = time.ParseDuration(ac.defaultEventRefresh)
		}
		recwatch.SetVerbose(ac.verboseMode)
		recwatch.LogError = func(err error) {
			log.Error(err)
		}
		recwatch.FatalExit = ac.fatalExit
		recwatch.Exists = ac.fs.Exists
		if ac.autoRefreshDir != "" {
			// Only watch the autoRefreshDir, recursively
			recwatch.EventServer(ac.autoRefreshDir, "*", ac.eventAddr, ac.defaultEventPath, ac.refreshDuration)
		} else {
			// Watch everything in the server directory, recursively
			recwatch.EventServer(ac.serverDirOrFilename, "*", ac.eventAddr, ac.defaultEventPath, ac.refreshDuration)
		}
	}

	// For communicating to and from the REPL
	ready := make(chan bool) // for when the server is up and running
	done := make(chan bool)  // for when the user wish to quit the server

	// The Lua REPL
	if !ac.serverMode {
		// If the REPL uses readline, the SIGWINCH signal is handled there
		go ac.REPL(ready, done)
	} else {
		// Ignore SIGWINCH if we are not going to use a REPL
		platformdep.IgnoreTerminalResizeSignal()
	}

	// Run the shutdown functions if graceful does not
	defer ac.GenerateShutdownFunction(nil, nil)()

	// Serve HTTP, HTTP/2 and/or HTTPS
	return ac.Serve(mux, done, ready)
}
