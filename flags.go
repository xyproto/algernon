package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

// TODO: Find a good external package for handling configuration and
//       another one for handling long and short flags.

const (
	defaultWebColonPort       = ":3000"
	defaultRedisColonPort     = ":6379"
	defaultEventColonPort     = ":5553"
	defaultEventRefresh       = "350ms"
	defaultEventPath          = "/fs"
	defaultLimit              = 10
	defaultPermissions        = 0660
	defaultCacheSize          = MiB      // 1 MiB
	defaultCacheMaxEntitySize = 64 * KiB // 64 KB
)

var (
	// Default rate limit, as a string
	defaultLimitString = strconv.Itoa(defaultLimit)

	// Default Bolt database file, for some operating systems
	defaultBoltFilename = "/tmp/algernon.db"

	// Default log file, for some operating systems
	defaultLogFile = "/tmp/algernon.log"

	// List of configuration filenames to check
	serverConfigurationFilenames = []string{"/etc/algernon/serverconf.lua"}

	// Configuration that is exposed to the server configuration script(s)
	serverDir, serverAddr, serverCert, serverKey, serverConfScript, internalLogFilename, serverLogFile string

	// If only HTTP/2 or HTTP
	serveJustHTTP2, serveJustHTTP bool

	// Configuration that may only be set in the server configuration script(s)
	serverAddrLua          string
	serverReadyFunctionLua func()

	// Server modes
	debugMode, verboseMode, productionMode, serverMode bool

	// For the Server-Sent Event (SSE) server
	eventAddr    string // Host and port to serve Server-Sent Events on
	eventRefresh string // The duration of an event cycle

	// Enable the event server and inject JavaScript to reload pages when sources change
	autoRefreshMode bool

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
	redisAddr          string
	redisDBindex       int
	redisAddrSpecified bool

	limitRequests       int64 // rate limit to this many requests per client per second
	disableRateLimiting bool

	// For the version flag
	showVersion bool

	// Caching
	cacheSize          uint64
	cacheMode          cacheModeSetting
	cacheCompression   bool
	cacheMaxEntitySize uint64

	// Output
	quietMode bool

	// If a single "server.lua" file is provided, or Server() is used.
	luaServerFilename string

	// Used in the HTTP headers as "Server"
	serverHeaderName string

	// CPU profile filename
	profileCPU string

	// Memory profile filename
	profileMem string

	// Assume files will not be removed from the server directory while
	// Algernon is running. This allows caching of costly os.Stat calls.
	cacheFileStat bool

	// Look for files in the directory with the same name as the requested hostname
	serverAddDomain bool
)

func usage() {
	fmt.Println("\n" + versionString + "\n\n" + description)
	// Possible arguments are also, for backward compatibility:
	// server dir, server addr, certificate file, key file, redis addr and redis db index
	// They are not mentioned here, but are possible to use, in that strict order.
	fmt.Println(`

Syntax:
  algernon [flags] [file or directory to serve]

Available flags:
  -h, --help                   This help text
  -v, --version                Application name and version
  --dir=DIRECTORY              Set the server directory
  --addr=[HOST][:PORT]         Server host and port ("` + defaultWebColonPort + `" is default)
  -e, --dev                    Development mode: Enables Debug mode, uses
                               regular HTTP, Bolt and sets cache mode "dev".
  -p, --prod                   Serve HTTP/2+HTTPS on port 443. Serve regular
                               HTTP on port 80. Uses /srv/algernon for files.
                               Disables debug mode. Disables auto-refresh.
                               Enables server mode. Sets cache to "production".
  -a, --autorefresh            Enable the event server and auto-refresh feature.
                               Sets cache mode to "images".
  --cache=MODE                 Sets a cache mode. The default is "on".
                               "on"      - Cache everything.
                               "dev"     - Everything, except Amber,
                                           Lua, GCSS, Markdown and JSX.
                               "prod"    - Everything, except Amber and Lua.
                               "small"   - Like "prod", but only files <= 64KB.
                               "images"  - Only images (png, jpg, gif, svg).
                               "off"     - Disable caching.
  --cachesize=N                Set the total cache size, in bytes.
  --rawcache                   Disable cache compression.
  --watchdir=DIRECTORY         Enables auto-refresh for only this directory.
  --cert=FILENAME              TLS certificate, if using HTTPS.
  --key=FILENAME               TLS key, if using HTTPS.
  -d, --debug                  Enable debug mode (display errors in the browser).
  -b, --bolt                   Use "` + defaultBoltFilename + `"
                               as the Bolt database.
  --boltdb=FILENAME            Use a specific file as the Bolt database
  --redis=[HOST][:PORT]        Use "` + defaultRedisColonPort + `"
                               as the Redis database.
  --dbindex=INDEX              Redis database index (0 is default).
  --conf=FILENAME              Lua script with additional configuration.
  --log=FILENAME               Log to a file instead of to the console.
  --internal=FILENAME          Internal log file (verbose when HTTP/2 is enabled)
  -t, --httponly               Serve regular HTTP
  --http2only                  Serve HTTP/2, without HTTPS (not recommended)
  --maria=DSN                  Use the given MariaDB or MySQL host
  --mariadb=NAME               Use the given MariaDB or MySQL database
  --verbose                    Slightly more verbose logging
  --eventserver=[HOST][:PORT]  SSE server address (for filesystem changes)
  --eventrefresh=DURATION      How often the event server should refresh
                               (the default is "` + defaultEventRefresh + `").
  --limit=N                    Limit clients to N requests per second
                               (the default is ` + defaultLimitString + `).
  --nolimit                    Disable rate limiting.
  -s, --server                 Server mode (disable debug + interactive mode).
  -q, --quiet                  Don't output anything to stdout or stderr.
  --servername                 Custom HTTP header value for the Server field.
  -c, --statcache              Speed up responses by caching os.Stat.
                               Only use if served files will not be removed.
  --domain                     Serve files from the subdirectory with the same
                               name as the requested domain.
`)
}

// Parse the flags, return the default hostname
func handleFlags(serverTempDir string) string {
	var (
		// The short version of some flags
		serveJustHTTPShort, autoRefreshShort, productionModeShort,
		debugModeShort, serverModeShort, useBoltShort, devModeShort,
		showVersionShort, quietModeShort, cacheFileStatShort bool
		// Used when setting the cache mode
		cacheModeString string
		// Used if disabling cache compression
		rawCache bool
	)

	// The usage function that provides more help (for --help or -h)
	flag.Usage = usage

	// The default for running the redis server on Windows is to listen
	// to "localhost:port", but not just ":port".
	host := ""
	if runtime.GOOS == "windows" {
		host = "localhost"
		// Default Bolt database file
		defaultBoltFilename = filepath.Join(serverTempDir, "algernon.db")
		// Default log file
		defaultLogFile = filepath.Join(serverTempDir, "algernon.log")
	}

	// Commandline flag configuration

	flag.StringVar(&serverDir, "dir", ".", "Server directory")
	flag.StringVar(&serverAddr, "addr", "", "Server [host][:port] (ie \":443\")")
	flag.StringVar(&serverCert, "cert", "cert.pem", "Server certificate")
	flag.StringVar(&serverKey, "key", "key.pem", "Server key")
	flag.StringVar(&redisAddr, "redis", "", "Redis [host][:port] (ie \""+defaultRedisColonPort+"\")")
	flag.IntVar(&redisDBindex, "dbindex", 0, "Redis database index")
	flag.StringVar(&serverConfScript, "conf", "serverconf.lua", "Server configuration")
	flag.StringVar(&serverLogFile, "log", "", "Server log file")
	flag.StringVar(&internalLogFilename, "internal", os.DevNull, "Internal log file")
	flag.BoolVar(&serveJustHTTP2, "http2only", false, "Serve HTTP/2, not HTTPS + HTTP/2")
	flag.BoolVar(&serveJustHTTP, "httponly", false, "Serve plain old HTTP")
	flag.BoolVar(&productionMode, "prod", false, "Production mode")
	flag.BoolVar(&debugMode, "debug", false, "Debug mode")
	flag.BoolVar(&verboseMode, "verbose", false, "Verbose logging")
	flag.BoolVar(&autoRefreshMode, "autorefresh", false, "Enable the auto-refresh feature")
	flag.StringVar(&autoRefreshDir, "watchdir", "", "Directory to watch (also enables auto-refresh)")
	flag.StringVar(&eventAddr, "eventserver", "", "SSE [host][:port] (ie \""+defaultEventColonPort+"\")")
	flag.StringVar(&eventRefresh, "eventrefresh", defaultEventRefresh, "Event refresh interval (ie \""+defaultEventRefresh+"\")")
	flag.BoolVar(&serverMode, "server", false, "Server mode (disable interactive mode)")
	flag.StringVar(&mariadbDSN, "maria", "", "MariaDB/MySQL connection string (DSN)")
	flag.StringVar(&mariaDatabase, "mariadb", "", "MariaDB/MySQL database name")
	flag.BoolVar(&useBolt, "bolt", false, "Use the default Bolt filename")
	flag.StringVar(&boltFilename, "boltdb", "", "Bolt database filename")
	flag.Int64Var(&limitRequests, "limit", defaultLimit, "Limit clients to a number of requests per second")
	flag.BoolVar(&disableRateLimiting, "nolimit", false, "Disable rate limiting")
	flag.BoolVar(&devMode, "dev", false, "Development mode")
	flag.BoolVar(&showVersion, "version", false, "Version")
	flag.StringVar(&cacheModeString, "cache", "", "Cache everything but Amber, Lua, GCSS and Markdown")
	flag.Uint64Var(&cacheSize, "cachesize", defaultCacheSize, "Cache size, in bytes")
	flag.BoolVar(&quietMode, "quiet", false, "Quiet")
	flag.BoolVar(&rawCache, "rawcache", false, "Disable cache compression")
	flag.StringVar(&serverHeaderName, "servername", versionString, "Server header name")
	flag.StringVar(&profileCPU, "cpuprofile", "", "Write CPU profile to file")
	flag.StringVar(&profileMem, "memprofile", "", "Write memory profile to file")
	flag.BoolVar(&cacheFileStat, "statcache", false, "Cache os.Stat")
	flag.BoolVar(&serverAddDomain, "domain", false, "Look for files in the directory named the same as the hostname")

	// The short versions of some flags
	flag.BoolVar(&serveJustHTTPShort, "t", false, "Serve plain old HTTP")
	flag.BoolVar(&autoRefreshShort, "a", false, "Enable the auto-refresh feature")
	flag.BoolVar(&serverModeShort, "s", false, "Server mode (disable interactive mode)")
	flag.BoolVar(&useBoltShort, "b", false, "Use the default Bolt filename")
	flag.BoolVar(&productionModeShort, "p", false, "Production mode")
	flag.BoolVar(&debugModeShort, "d", false, "Debug mode")
	flag.BoolVar(&devModeShort, "e", false, "Development mode")
	flag.BoolVar(&showVersionShort, "v", false, "Version")
	flag.BoolVar(&quietModeShort, "q", false, "Quiet")
	flag.BoolVar(&cacheFileStatShort, "c", false, "Cache os.Stat")

	flag.Parse()

	// Accept both long and short versions of some flags
	serveJustHTTP = serveJustHTTP || serveJustHTTPShort
	autoRefreshMode = autoRefreshMode || autoRefreshShort
	debugMode = debugMode || debugModeShort
	serverMode = serverMode || serverModeShort
	useBolt = useBolt || useBoltShort
	productionMode = productionMode || productionModeShort
	devMode = devMode || devModeShort
	showVersion = showVersion || showVersionShort
	quietMode = quietMode || quietModeShort
	cacheFileStat = cacheFileStat || cacheFileStatShort

	// Disable verbose mode if quiet mode has been enabled
	if quietMode {
		verboseMode = false
	}

	// Enable cache compression unless raw cache is specified
	cacheCompression = !rawCache

	redisAddrSpecified = redisAddr != ""
	if redisAddr == "" {
		// The default host and port
		redisAddr = host + defaultRedisColonPort
	}

	// May be overriden by devMode
	if serverMode {
		debugMode = false
	}

	// TODO: If flags are set in addition to -p or -e, don't override those
	//       when -p or -e is set.

	// Change several defaults if production mode is enabled
	if productionMode {
		// Use system directories
		serverDir = "/srv/algernon"
		serverCert = "/etc/algernon/cert.pem"
		serverKey = "/etc/algernon/key.pem"
		cacheMode = cacheModeProduction
		serverMode = true
	} else if devMode {
		// Change several defaults if development mode is enabled
		useBolt = true
		serveJustHTTP = true
		//serverLogFile = defaultLogFile
		debugMode = true
		// TODO: Make it possible to set --limit to the default limit also when -e is used
		if limitRequests == defaultLimit {
			limitRequests = 700 // Increase the rate limit considerably
		}
		cacheMode = cacheModeDevelopment
	}

	// If a watch directory is given, enable the auto refresh feature
	if autoRefreshDir != "" {
		autoRefreshMode = true
	}

	// If auto-refresh is enabled, change the caching
	if autoRefreshMode {
		if cacheModeString == "" {
			// Disable caching by default, when auto-refresh is enabled
			cacheMode = cacheModeOff
		}
	}

	// The cache flag overrides the settings from the other modes
	if cacheModeString != "" {
		cacheMode = newCacheModeSetting(cacheModeString)
	}

	// Disable cache entirely if cacheSize is set to 0
	if cacheSize == 0 {
		cacheMode = cacheModeOff
	}

	// Set cacheSize to 0 if the cache is disabled
	if cacheMode == cacheModeOff {
		cacheSize = 0
	}

	// If cache mode is unset, use the dev mode
	if cacheMode == cacheModeUnset {
		cacheMode = cacheModeDefault
	}

	if cacheMode == cacheModeSmall {
		cacheMaxEntitySize = defaultCacheMaxEntitySize
	}

	// For backwards compatibility with previous versions of algernon

	if len(flag.Args()) >= 1 {
		serverDir = flag.Args()[0]
	}
	if len(flag.Args()) >= 2 {
		serverAddr = flag.Args()[1]
	}
	if len(flag.Args()) >= 3 {
		serverCert = flag.Args()[2]
	}
	if len(flag.Args()) >= 4 {
		serverKey = flag.Args()[3]
	}
	if len(flag.Args()) >= 5 {
		redisAddr = flag.Args()[4]
		redisAddrSpecified = true
	}
	if len(flag.Args()) >= 6 {
		// Convert the dbindex from string to int
		DBindex, err := strconv.Atoi(flag.Args()[5])
		if err != nil {
			redisDBindex = DBindex
		}
	}

	// Add the serverConfScript to the list of configuration scripts to be read and executed
	serverConfigurationFilenames = append(serverConfigurationFilenames, serverConfScript)

	return host
}

// Set the values that has not been set by flags nor scripts (and can be set by both)
// Returns true if a "ready function" has been run.
func finalConfiguration(host string) bool {

	// Set the server host and port (commandline flags overrides Lua configuration)
	if serverAddr == "" {
		if serverAddrLua != "" {
			serverAddr = serverAddrLua
		} else {
			serverAddr = host + defaultWebColonPort
		}
	}

	// Set the event server host and port
	if eventAddr == "" {
		eventAddr = host + defaultEventColonPort
	}

	// Turn off debug mode if production mode is enabled
	if productionMode {
		// Turn off debug mode
		debugMode = false
	}

	hasReadyFunction := serverReadyFunctionLua != nil

	// Run the Lua function specified with the OnReady function, if available
	if hasReadyFunction {
		// Useful for outputting configuration information after both
		// configuration scripts have been run and flags have been parsed
		serverReadyFunctionLua()
	}

	return hasReadyFunction
}
