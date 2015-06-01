package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
)

const (
	defaultWebColonPort   = ":3000"
	defaultRedisColonPort = ":6379"
	defaultEventColonPort = ":5553"
	defaultEventRefresh   = "350ms"
	defaultEventPath      = "/fs"
)

var (
	// Default Bolt database file, for some operating systems
	defaultBoltFilename = "/tmp/algernon.db"

	// Default log file, for some operating systems
	defaultLogFile = "/tmp/algernon.log"

	// List of configuration filenames to check
	serverConfigurationFilenames = []string{"/etc/algernon/serverconf.lua"}

	// Configuration that is exposed to the server configuration script(s)
	serverDir, serverAddr, serverCert, serverKey, serverConfScript, serverHTTP2log, serverLogFile string

	// If only HTTP/2 or HTTP
	serveJustHTTP2, serveJustHTTP bool

	// Configuration that may only be set in the server configuration script(s)
	serverAddrLua          string
	serverReadyFunctionLua func()

	// Server modes
	debugMode, verboseMode, productionMode, interactiveMode bool

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
	boltFilename    string
	useBolt         bool
	mariadbDSN      string // connection string
	mariadbDatabase string // database name
	redisAddr       string
	redisDBindex    int

	limitRequests       int64 // rate limit to this many requests per client per second
	disableRateLimiting bool

	// For the version flag
	showVersion bool
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
  --help                       This help
  -v, --version                    Application name and version
  --dir=DIRECTORY              Set the server directory
  --addr=[HOST][:PORT]         Server host and port ("` + defaultWebColonPort + `" is default)
  -e, --dev                    Development mode: Enable Debug mode, enables
                               interactive mode, uses regular HTTP, uses Bolt.
  -p, --prod                   Serve HTTP/2+HTTPS on port 443. Serve regular
                               HTTP on port 80. Use /srv/algernon as the server
                               directory. Disable debug mode and auto-refresh.
  -a, --autorefresh            Enable the event server and auto-refresh feature.
  --watchdir=DIRECTORY         Enables auto-refresh for only this directory.
  --cert=FILENAME              TLS certificate, if using HTTPS
  --key=FILENAME               TLS key, if using HTTPS
  --debug                      Enable debug mode (shows errors in the browser).
  -b, --bolt                   Use "` + defaultBoltFilename + `" as the Bolt database
  --boltdb=FILENAME            Use a specific file as the Bolt database
  --redis=[HOST][:PORT]        Use the given Redis database ("` + defaultRedisColonPort + `")
  --dbindex=INDEX              Redis database index (0 is default)
  --conf=FILENAME              Lua script with additional configuration
  --log=FILENAME               Log to a file instead of to the console
  --http2log=FILENAME          Additional HTTP/2 log (quite verbose)
  -h, --httponly               Serve plain HTTP
  --http2only                  Serve HTTP/2, without HTTPS (not recommended)
  --maria=DSN                  Use the given MariaDB or MySQL host
  --mariadb=NAME               Use the given MariaDB or MySQL database
  --verbose                    Slightly more verbose logging
  --eventserver=[HOST][:PORT]  SSE server address (for filesystem changes)
  --eventrefresh=DURATION      How often the event server should refresh
                               (the default is "` + defaultEventRefresh + `").
  --limit=N                    Limit clients to N requests per second
  --no-limit                   Disable rate limiting
  -i, --interactive            Interactive mode
`)
}

// Parse the flags, return the default hostname
func handleFlags(serverTempDir string) string {
	// The short version of some flags
	var serveJustHTTPShort, autoRefreshShort, productionModeShort,
		debugModeShort, interactiveModeShort, useBoltShort, devModeShort,
		showVersionShort bool

	// The usage function that provides more help
	flag.Usage = usage

	// The default for running the redis server on Windows is to listen
	// to "localhost:port", but not just ":port".
	host := ""
	if runtime.GOOS == "windows" {
		host = "localhost"
		// Disable colors when logging, for some systems
		//log.SetFormatter(&log.TextFormatter{DisableColors: true})

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
	flag.StringVar(&redisAddr, "redis", host+defaultRedisColonPort, "Redis [host][:port] (ie \""+defaultRedisColonPort+"\")")
	flag.IntVar(&redisDBindex, "dbindex", 0, "Redis database index")
	flag.StringVar(&serverConfScript, "conf", "serverconf.lua", "Server configuration")
	flag.StringVar(&serverLogFile, "log", "", "Server log file")
	flag.StringVar(&serverHTTP2log, "http2log", "/dev/null", "HTTP/2 log")
	flag.BoolVar(&serveJustHTTP2, "http2only", false, "Serve HTTP/2, not HTTPS + HTTP/2")
	flag.BoolVar(&serveJustHTTP, "httponly", false, "Serve plain old HTTP")
	flag.BoolVar(&productionMode, "prod", false, "Production mode")
	flag.BoolVar(&debugMode, "debug", false, "Debug mode")
	flag.BoolVar(&verboseMode, "verbose", false, "Verbose logging")
	flag.BoolVar(&autoRefresh, "autorefresh", false, "Enable the auto-refresh feature")
	flag.StringVar(&autoRefreshDir, "watchdir", "", "Directory to watch (also enables auto-refresh)")
	flag.StringVar(&eventAddr, "eventserver", "", "SSE [host][:port] (ie \""+defaultEventColonPort+"\")")
	flag.StringVar(&eventRefresh, "eventrefresh", defaultEventRefresh, "Event refresh interval (ie \""+defaultEventRefresh+"\")")
	flag.BoolVar(&interactiveMode, "interactive", false, "Interactive mode")
	flag.StringVar(&mariadbDSN, "maria", "", "MariaDB/MySQL connection string (DSN)")
	flag.StringVar(&mariadbDatabase, "mariadb", "", "MariaDB/MySQL database name")
	flag.BoolVar(&useBolt, "bolt", false, "Use the default Bolt filename")
	flag.StringVar(&boltFilename, "boltdb", "", "Bolt database filename")
	flag.Int64Var(&limitRequests, "limit", 1, "Limit clients to a number of requests per second")
	flag.BoolVar(&disableRateLimiting, "no-limit", false, "Disable rate limiting")
	flag.BoolVar(&devMode, "dev", false, "Development mode")
	flag.BoolVar(&showVersion, "version", false, "Version")

	// The short versions of some flags
	flag.BoolVar(&serveJustHTTPShort, "h", false, "Serve plain old HTTP")
	flag.BoolVar(&autoRefreshShort, "a", false, "Enable the auto-refresh feature")
	flag.BoolVar(&interactiveModeShort, "i", false, "Interactive mode")
	flag.BoolVar(&useBoltShort, "b", false, "Use the default Bolt filename")
	flag.BoolVar(&productionModeShort, "p", false, "Production mode")
	flag.BoolVar(&debugModeShort, "d", false, "Debug mode")
	flag.BoolVar(&devModeShort, "e", false, "Development mode")
	flag.BoolVar(&showVersionShort, "v", false, "Version")

	flag.Parse()

	// Accept both long and short versions of some flags
	serveJustHTTP = serveJustHTTP || serveJustHTTPShort
	autoRefresh = autoRefresh || autoRefreshShort
	debugMode = debugMode || debugModeShort
	interactiveMode = interactiveMode || interactiveModeShort
	useBolt = useBolt || useBoltShort
	productionMode = productionMode || productionModeShort
	devMode = devMode || devModeShort
	showVersion = showVersion || showVersionShort

	// Change several defaults if production mode is enabled
	if productionMode {
		// Use system directories
		serverDir = "/srv/algernon"
		serverCert = "/etc/algernon/cert.pem"
		serverKey = "/etc/algernon/key.pem"
	} else if devMode {
		// Change several defaults if development mode is enabled
		useBolt = true
		serveJustHTTP = true
		serverLogFile = defaultLogFile
		debugMode = true
		interactiveMode = true
		limitRequests = 1000 // Increase the rate limit considerably
	}

	// If a watch directory is given, enable the auto refresh feature
	if autoRefreshDir != "" {
		autoRefresh = true
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
