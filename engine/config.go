// Package engine provides the server configuration struct and several functions for serving files over HTTP
package engine

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/cachemode"
	"github.com/xyproto/algernon/lua/pool"
	"github.com/xyproto/algernon/platformdep"
	"github.com/xyproto/algernon/utils"
	"github.com/xyproto/datablock"
	"github.com/xyproto/env/v2"
	"github.com/xyproto/mime"
	"github.com/xyproto/pinterface"
	"github.com/xyproto/recwatch"
	"github.com/xyproto/unzip"
	"github.com/xyproto/vt"
)

const (
	// Version number. Stable API within major version numbers.
	Version = 2.0

	// The default supporting filename for a Lua script that provides data to a template
	defaultLuaDataFilename = "data.lua"
)

// Config is the main structure for the Algernon server.
// It contains all the state and settings.
// The order of the fields has been decided by the "fieldalignment" utility.
type Config struct {
	perm                         pinterface.IPermissions // the user state, for the permissions system
	mimereader                   *mime.Reader
	serverReadyFunctionLua       func()              // configuration that may only be set in the server configuration script(s)
	pongomutex                   *sync.RWMutex       // workaround for rendering pongo2 pages without concurrency issues
	fs                           *datablock.FileStat // for checking if file exists, possibly in a cached way
	luapool                      *pool.LStatePool    // a pool of Lua interpreters
	cache                        *datablock.FileCache
	reverseProxyConfig           *ReverseProxyConfig
	redisAddr                    string
	defaultEventPath             string
	defaultEventRefresh          string
	description                  string // description of the current program
	versionString                string // program name and version number
	defaultOpenExecutable        string // default program for opening files and URLs in the current operating system
	serverHost                   string
	defaultEventColonPort        string
	defaultLimitString           string // default rate limit, as a string
	limitRequestsString          string // store the request limit as a string for faster HTTP header creation later on
	defaultBoltFilename          string // default bolt database file, for some operating systems
	defaultLogFile               string // default log file, for some operating systems
	defaultRedisColonPort        string
	serverDirOrFilename          string // exposed to the server configuration scripts(s)
	serverAddr                   string // exposed to the server configuration scripts(s)
	serverCert                   string // exposed to the server configuration scripts(s)
	serverKey                    string // exposed to the server configuration scripts(s)
	serverConfScript             string // exposed to the server configuration scripts(s)
	defaultWebColonPort          string
	serverLogFile                string // exposed to the server configuration scripts(s)
	serverTempDir                string // temporary directory
	cookieSecret                 string // secret to be used when setting and getting user login cookies
	defaultTheme                 string // theme for Markdown and error pages
	openExecutable               string // open the URL after serving, with a specific executable
	serverAddrLua                string // configuration that may only be set in the server configuration script(s)
	dbName                       string
	serverHeaderName             string // used in the HTTP headers as the "Server" name
	eventAddr                    string // for the Server-Sent Event (SSE) server (host and port)
	eventRefresh                 string // for the Server-Sent Event (SSE) server (duration of an event cycle)
	luaServerFilename            string // if a single Lua file is provided, or if Server() is used
	autoRefreshDir               string // if only watching a single directory recursively
	combinedAccessLogFilename    string // CLF access log
	commonAccessLogFilename      string // NCSA access log
	boltFilename                 string
	internalLogFilename          string               // exposed to the server configuration scripts(s)
	mariadbDSN                   string               // connection string
	mariaDatabase                string               // database name
	sqliteConnectionString       string               // SQLite connection string
	postgresDSN                  string               // connection string
	postgresDatabase             string               // database name
	dirBaseURL                   string               // optional Base URL, for the directory listings
	jsxOptions                   api.TransformOptions // JSX rendering options
	certMagicDomains             []string
	serverConfigurationFilenames []string // list of configuration filenames to check
	cacheMaxGivenDataSize        uint64
	largeFileSize                uint64        // threshold for not reading large files into memory
	refreshDuration              time.Duration // for the auto-refresh feature
	redisDBindex                 int
	cacheSize                    uint64
	cacheMode                    cachemode.Setting
	shutdownTimeout              time.Duration
	cacheMaxEntitySize           uint64
	defaultLimit                 int64
	defaultCacheMaxEntitySize    uint64        // 64 KiB
	defaultLargeFileSize         uint64        // 42 MiB: the default size for when a static file is large enough to not be read into memory
	limitRequests                int64         // rate limit to this many requests per client per second
	writeTimeout                 uint64        // timeout when writing data to a client, in seconds
	defaultStatCacheRefresh      time.Duration // refresh the stat cache, if the stat cache feature is enabled
	defaultCacheSize             uint64        // 1 MiB
	defaultPermissions           os.FileMode
	quietMode                    bool // no output to the command line
	autoRefresh                  bool // enable the event server and inject JavaScript to reload pages when sources change
	serverMode                   bool // server mode: non-interactive
	productionMode               bool // server mode: non-interactive, assume the server is running as a system service
	verboseMode                  bool // server mode: be more verbose
	debugMode                    bool // server mode: enable debug features and better error messages
	cacheFileStat                bool // assume files will not be removed from the served directories while Algernon is running, which allows caching of costly os.Stat calls
	serverAddDomain              bool // look for files in the directory with the same name as the requested hostname
	stricterHeaders              bool // stricter HTTP headers
	simpleMode                   bool // server mode: for serving a directory with files over regular HTTP, nothing more nothing less
	openURLAfterServing          bool // open the URL after serving
	onlyLuaMode                  bool // if only using the Lua REPL, and not serving anything
	quitAfterFirstRequest        bool // quit when the first request has been responded to?
	markdownMode                 bool
	serveJustQUIC                bool // if only QUIC or HTTP/3
	serveJustHTTP                bool // if only HTTP
	serveJustHTTP2               bool // if only HTTP/2
	ctrldTwice                   bool // require a double press of ctrl-d to exit the REPL
	noHeaders                    bool // HTTP headers
	redisAddrSpecified           bool
	noCache                      bool
	showVersion                  bool
	curlSupport                  bool // support clients like "curl" that downloads uncompressed by default
	noBanner                     bool // don't display the ANSI-graphics banner at start
	cacheCompressionSpeed        bool // compression speed over compactness
	cacheCompression             bool
	singleFileMode               bool // if only serving a single file, like a Lua script
	hyperApp                     bool // convert JSX to HyperApp JS, or React JS?
	devMode                      bool // server mode: aims to make it easy to get started
	clearDefaultPathPrefixes     bool // clear default path prefixes like "/admin" from the permission system?
	disableRateLimiting          bool
	redirectHTTP                 bool // redirect HTTP traffic to HTTPS?
	useCertMagic                 bool // use CertMagic and Let's Encrypt for all directories in the given directory that contains a "."
	useBolt                      bool
	useNoDatabase                bool // don't use a database. There will be a loss of functionality.
}

// ErrVersion is returned when the initialization quits because all that is done
// is showing version information
var (
	ErrVersion  = errors.New("only showing version information")
	ErrDatabase = errors.New("could not find a usable database backend")
)

// New creates a new server configuration based using the default values
func New(versionString, description string) (*Config, error) {
	tmpdir := env.Str("TMPDIR", "/tmp")
	ac := &Config{
		curlSupport: true,

		shutdownTimeout: 10 * time.Second,

		defaultWebColonPort:       ":3000",
		defaultRedisColonPort:     ":6379",
		defaultEventColonPort:     ":5553",
		defaultEventRefresh:       "350ms",
		defaultEventPath:          "/sse",
		defaultLimit:              10,
		defaultPermissions:        0o660,
		defaultCacheSize:          1 * utils.MiB,   // 1 MiB
		defaultCacheMaxEntitySize: 64 * utils.KiB,  // 64 KB
		defaultStatCacheRefresh:   time.Minute * 1, // Refresh the stat cache, if the stat cache feature is enabled

		// When is a static file large enough to not read into memory when serving
		defaultLargeFileSize: 42 * utils.MiB, // 42 MiB

		// Default rate limit, as a string
		defaultLimitString: strconv.Itoa(10),

		// Default Bolt database file, for some operating systems
		defaultBoltFilename: filepath.Join(tmpdir, "algernon.db"),

		// Default log file, for some operating systems
		defaultLogFile: filepath.Join(tmpdir, "algernon.log"),

		// List of configuration filenames to check
		serverConfigurationFilenames: []string{"/etc/algernon/serverconf.lua", "/etc/algernon/server.lua"},

		// Compression speed over compactness
		cacheCompressionSpeed: true,

		// TODO: Make configurable
		// Maximum given file size for caching, 7 MiB
		cacheMaxGivenDataSize: 7 * utils.MiB,

		// Mutex for rendering Pongo2 pages
		pongomutex: &sync.RWMutex{},

		// Program for opening URLs, keep empty for using the default in url.go
		defaultOpenExecutable: "",

		// General information about Algernon
		versionString: versionString,
		description:   description,

		// JSX rendering options
		jsxOptions: api.TransformOptions{
			Loader:            api.LoaderJSX,
			MinifyWhitespace:  true,
			MinifyIdentifiers: true,
			MinifySyntax:      true,
			Charset:           api.CharsetUTF8,
		},
	}
	if err := ac.initFilesAndCache(); err != nil {
		return nil, err
	}
	// Read in the mimetype information from the system. Set UTF-8 when setting Content-Type.
	ac.mimereader = mime.New("/etc/mime.types", true)
	ac.setupLogging()

	// File stat cache
	ac.fs = datablock.NewFileStat(ac.cacheFileStat, ac.defaultStatCacheRefresh)

	return ac, nil
}

// SetFileStatCache can be used to set a different FileStat cache than the default one
func (ac *Config) SetFileStatCache(fs *datablock.FileStat) {
	ac.fs = fs
}

// Initialize a temporary directory, handle flags, output version and handle profiling
func (ac *Config) initFilesAndCache() error {
	// Temporary directory that might be used for logging, databases or file extraction
	serverTempDir, err := os.MkdirTemp("", "algernon")
	if err != nil {
		return err
	}
	ac.serverTempDir = serverTempDir

	// Set several configuration variables, based on the given flags and arguments
	ac.handleFlags(ac.serverTempDir)

	// Version (--version)
	if ac.showVersion {
		if !ac.quietMode {
			fmt.Println(ac.versionString)
		}
		return ErrVersion
	}

	// CPU and memory profiling, if it is enabled at build time, and one of these
	// flags are provided (+ a filename): -cpuprofile, -memprofile, -fgtrace or -trace
	traceStart()

	// Touch the common access log, if specified
	if ac.commonAccessLogFilename != "" {
		// Create if missing
		f, err := os.OpenFile(ac.commonAccessLogFilename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
		if err != nil {
			return err
		}
		f.Close()
	}
	// Touch the combined access log, if specified
	if ac.combinedAccessLogFilename != "" {
		// Create if missing
		f, err := os.OpenFile(ac.combinedAccessLogFilename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
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
			logrus.Warnf("Could not log to %s: %s", ac.serverLogFile, errJSONLog)
		} else {
			// Log to the given log filename
			logrus.SetFormatter(&logrus.JSONFormatter{})
			logrus.SetOutput(f)
		}
	} else if ac.quietMode {
		// If quiet mode is enabled and no log file has been specified, disable logging
		logrus.SetOutput(io.Discard)
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
		logrus.Error(err)
	}
	// Then switch to stderr and log the message there as well
	logrus.SetOutput(os.Stderr)
	// Use the standard formatter
	logrus.SetFormatter(&logrus.TextFormatter{})
	// Log and exit
	logrus.Fatalln(err.Error())
}

// Abrupt exit
func (ac *Config) abruptExit(msg string) {
	// Log to file, if a log file is used
	if ac.serverLogFile != "" {
		logrus.Info(msg)
	}
	// Then switch to stderr and log the message there as well
	logrus.SetOutput(os.Stderr)
	// Use the standard formatter
	logrus.SetFormatter(&logrus.TextFormatter{})
	// Log and exit
	logrus.Info(msg)
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
		case ".amber", ".lua", ".po2", ".pongo2", ".tl", ".tpl":
			return false
		default:
			return true
		}
	case cachemode.Images:
		switch ext {
		case ".apng", ".bmp", ".gif", ".ico", ".jpeg", ".jpg", ".png", ".svg", ".webp":
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
		case ".amber", ".gcss", ".happ", ".js", ".jsx", ".lua", ".md", ".po2", ".pongo2", ".scss", ".tl", ".tpl":
			return false
		default:
			return true
		}
	}
}

// hasHandlers checks if the given filename contains "handle(" or "handle ("
func hasHandlers(fn string) bool {
	data, err := os.ReadFile(fn)
	return err == nil && (bytes.Contains(data, []byte("handle(")) || bytes.Contains(data, []byte("handle (")))
}

// has checks if a given slice of strings contains a given string
func has(sl []string, e string) bool {
	for _, s := range sl {
		if e == s {
			return true
		}
	}
	return false
}

// unique removes all repeated elements from a slice of strings
func unique(sl []string) []string {
	var nl []string
	for _, s := range sl {
		if !has(nl, s) {
			nl = append(nl, s)
		}
	}
	return nl
}

// MustServe sets up a server with handlers
func (ac *Config) MustServe(mux *http.ServeMux) error {
	var err error

	defer ac.Close()

	// Output what we are attempting to access and serve
	if ac.verboseMode {
		logrus.Info("Accessing " + ac.serverDirOrFilename)
	}

	// Check if the given directory really is a directory
	if !ac.fs.IsDir(ac.serverDirOrFilename) {
		// It is not a directory
		serverFile := ac.serverDirOrFilename

		// Return with an error if the file to serve does not exist
		if !ac.fs.Exists(serverFile) {
			return fmt.Errorf("file does not exist: %s", serverFile)
		}

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
			webApplicationExtractionDir := "/dev/shm" // extract to memory, if possible
			testfile := filepath.Join(webApplicationExtractionDir, "canary")
			if _, err := os.Create(testfile); err == nil { // success
				os.Remove(testfile)
			} else {
				// Could not create the test file
				// Use the server temp dir (typically /tmp) instead of /dev/shm
				webApplicationExtractionDir = ac.serverTempDir
			}
			// Extract the web application
			if extractErr := unzip.Extract(serverFile, webApplicationExtractionDir); extractErr != nil {
				return extractErr
			}
			// Use the directory where the file was extracted as the server directory
			ac.serverDirOrFilename = webApplicationExtractionDir
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
				ac.serverConfigurationFilenames = append(ac.serverConfigurationFilenames, configFilename)
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

	}

	// Make a few changes to the defaults if we are serving a single file
	if ac.singleFileMode {
		ac.debugMode = true
		ac.serveJustHTTP = true
	}

	to := vt.NewTextOutput(runtime.GOOS != "windows", !ac.quietMode)

	// Console output
	if !ac.quietMode && !ac.singleFileMode && !ac.simpleMode && !ac.noBanner {
		// Output a colorful ansi logo if a proper terminal is available
		fmt.Println(Banner(ac.versionString, ac.description))
	} else if !ac.quietMode {
		timestamp := time.Now().Format("2006-01-02 15:04")
		to.OutputTags("<cyan>" + ac.versionString + "<darkgray> - " + timestamp + "<off>")
		// colorstring.Println("[cyan]" + ac.versionString + "[dark_gray] - " + timestamp + "[reset]")
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
		ac.luapool.Shutdown()
	})

	// TODO: save repl history + close luapool + close logs ++ at shutdown

	if ac.singleFileMode && (filepath.Ext(ac.serverDirOrFilename) == ".lua" || ac.onlyLuaMode) {
		ac.luaServerFilename = ac.serverDirOrFilename
		if ac.luaServerFilename == "index.lua" || ac.luaServerFilename == "data.lua" {
			// Friendly message to new users
			if !hasHandlers(ac.luaServerFilename) {
				logrus.Warnf("Found no handlers in %s", ac.luaServerFilename)
				logrus.Info("How to implement \"Hello, World!\" in " + ac.luaServerFilename + " file:\n\nhandle(\"/\", function()\n  print(\"Hello, World!\")\nend)\n")
			}
		}
		ac.serverDirOrFilename = filepath.Dir(ac.serverDirOrFilename)
		// Make it possible to read other files from the Lua script
		ac.singleFileMode = false
	}

	ac.serverConfigurationFilenames = unique(ac.serverConfigurationFilenames)

	// Color scheme
	arrowColor := "<lightblue>"
	filenameColor := "<white>"
	luaOutputColor := "<darkgray>"
	dashLineColor := "<red>"

	// Create a Colorize struct that will not reset colors after colorizing
	// strings meant for the terminal.
	// c := colorstring.Colorize{Colors: colorstring.DefaultColors, Reset: false}

	if (len(ac.serverConfigurationFilenames) > 0) && !ac.quietMode && !ac.onlyLuaMode {
		fmt.Println(to.Tags(dashLineColor + strings.Repeat("-", 49) + "<off>"))
	}

	// Read server configuration script, if present.
	// The scripts may change global variables.
	var ranConfigurationFilenames []string
	for _, filename := range unique(ac.serverConfigurationFilenames) {
		if ac.fs.Exists(filename) {
			// Dividing line between the banner and output from any of the configuration scripts
			if !ac.quietMode && !ac.onlyLuaMode {
				// Output the configuration filename
				to.Println(arrowColor + "-> " + filenameColor + filename + "<off>")
				fmt.Print(to.Tags(luaOutputColor))
			} else if ac.verboseMode {
				logrus.Info("Running Lua configuration file: " + filename)
			}
			withHandlerFunctions := true
			errConf := ac.RunConfiguration(filename, mux, withHandlerFunctions)
			if errConf != nil {
				if ac.perm != nil {
					logrus.Error("Could not use configuration script: " + filename)
					return errConf
				}
				if ac.verboseMode {
					logrus.Info("Skipping " + filename + " because the database backend is not in use.")
				}
			}
			ranConfigurationFilenames = append(ranConfigurationFilenames, filename)
		} else {
			if ac.verboseMode {
				logrus.Info("Looking for: " + filename)
			}
		}
	}
	// Only keep the active ones. Used when outputting server information.
	ac.serverConfigurationFilenames = ranConfigurationFilenames

	// Run the standalone Lua server, if specified
	if ac.luaServerFilename != "" {
		// Run the Lua server file and set up handlers
		if !ac.quietMode && !ac.onlyLuaMode {
			// Output the configuration filename
			to.Println(arrowColor + "-> " + filenameColor + ac.luaServerFilename + "<off>")
			fmt.Print(to.Tags(luaOutputColor))
		} else if ac.verboseMode {
			fmt.Println("Running Lua configuration file: " + ac.luaServerFilename)
		}
		withHandlerFunctions := true
		errLua := ac.RunConfiguration(ac.luaServerFilename, mux, withHandlerFunctions)
		if errLua != nil {
			logrus.Errorf("Error in %s (interpreted as a server script):\n%s\n", ac.luaServerFilename, errLua)
			return errLua
		}
	} else {
		// Register HTTP handler functions
		ac.RegisterHandlers(mux, "/", ac.serverDirOrFilename, ac.serverAddDomain)
	}

	// Set the values that has not been set by flags nor scripts
	// (and can be set by both)
	ranServerReadyFunction := ac.finalConfiguration(ac.serverHost)

	if !ac.quietMode && !ac.onlyLuaMode {
		to.Print("<off>")
	}

	// If no configuration files were being ran successfully,
	// output basic server information.
	if len(ac.serverConfigurationFilenames) == 0 {
		if !ac.quietMode && !ac.onlyLuaMode {
			fmt.Println(ac.Info())
		}
		ranServerReadyFunction = true
	}

	// Separator between the output of the configuration scripts and
	// the rest of the server output.
	if ranServerReadyFunction && (len(ac.serverConfigurationFilenames) > 0) && !ac.quietMode && !ac.onlyLuaMode {
		to.Tags(dashLineColor + strings.Repeat("-", 49) + "<off>")
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
			ac.fatalExit(fmt.Errorf("could not write to %s nor %s", ac.internalLogFilename, "internal.log"))
		}
	}
	defer internalLogFile.Close()
	log.SetOutput(internalLogFile)

	// Serve filesystem events in the background.
	// Used for reloading pages when the sources change.
	// Can also be used when serving a single file.
	if ac.autoRefresh {
		ac.refreshDuration, err = time.ParseDuration(ac.eventRefresh)
		if err != nil {
			logrus.Warnf("%s is an invalid duration. Using %s instead.", ac.eventRefresh, ac.defaultEventRefresh)
			// Ignore the error, since defaultEventRefresh is a constant and must be parseable
			ac.refreshDuration, _ = time.ParseDuration(ac.defaultEventRefresh)
		}
		recwatch.SetVerbose(ac.verboseMode)
		recwatch.LogError = func(err error) {
			logrus.Error(err)
		}
		recwatch.FatalExit = ac.fatalExit
		recwatch.Exists = ac.fs.Exists
		if ac.autoRefreshDir != "" {
			absdir, err := filepath.Abs(ac.autoRefreshDir)
			if err != nil {
				absdir = ac.autoRefreshDir
			}
			// Only watch the autoRefreshDir, recursively
			recwatch.EventServer(absdir, "*", ac.eventAddr, ac.defaultEventPath, ac.refreshDuration)
		} else {
			absdir, err := filepath.Abs(ac.serverDirOrFilename)
			if err != nil {
				absdir = ac.serverDirOrFilename
			}
			// Watch everything in the server directory, recursively
			recwatch.EventServer(absdir, "*", ac.eventAddr, ac.defaultEventPath, ac.refreshDuration)
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

	// Setup a signal handler for clearing the cache when USR1 is received, for some platforms
	platformdep.SetupSignals(ac.ClearCache, logrus.Infof)

	// Run the shutdown functions if graceful does not
	defer ac.GenerateShutdownFunction(nil)()

	// Serve HTTP, HTTP/2 and/or HTTPS
	return ac.Serve(mux, done, ready)
}
