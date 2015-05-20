package main

import (
	"flag"
	"fmt"
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
	// List of configuration filenames to check
	serverConfigurationFilenames = []string{"/etc/algernon/serverconf.lua"}

	// Configuration that is exposed to the server configuration script(s)
	serverDir, serverAddr, serverCert, serverKey, serverConfScript, serverHTTP2log string

	// If only HTTP/2 or HTTP
	serveJustHTTP2, serveJustHTTP bool

	// Configuration that may only be set in the server configuration script(s)
	serverAddrLua          string
	serverReadyFunctionLua func()

	// Redis configuration
	redisAddr    string
	redisDBindex int

	// Server modes
	debugMode, verboseMode, productionMode, interactiveMode bool

	// For the Server-Sent Event (SSE) server
	eventAddr    string
	eventRefresh string // Event server refresh, ie "350ms"

	// Enable the event server and inject JavaScript to reload pages when sources change
	autoRefresh bool

	// If serving a single file, like a lua script
	singleFileMode bool
)

func usage() {
	fmt.Println("\n" + versionString + "\n\n" + description)
	// Possible arguments are also, for backward compatibility:
	// server dir, server addr, certificate file, key file, redis addr and redis db index
	// They are not mentioned here, but are possible to use, in that strict order.
	fmt.Println(`

Syntax:
  algernon [flags] [server dir] [server addr]

Available flags:
  --help                       This help
  --version                    Application name and version
  --dir=DIRECTORY              Set the server directory
  --addr=[HOST][:PORT]         Server host and port ("` + defaultWebColonPort + `" is default)
  -a, --autorefresh            Enable the event server and auto-refresh feature.
  --prod                       Serve HTTP/2+HTTPS on port 443. Serve regular
                               HTTP on port 80. Use /srv/algernon as the server
                               directory. Disable debug mode and auto-refresh.
  -d, --debug                  Enable debug mode
  --cert=FILENAME              TLS certificate, if using HTTPS
  --key=FILENAME               TLS key, if using HTTPS
  --redis=[HOST][:PORT]        Connect to a remote Redis database ("` + defaultRedisColonPort + `")
  --dbindex=INDEX              Redis database index (0 is default)
  --conf=FILENAME              Lua script with additional configuration
  --http2log=FILENAME          Save the verbose HTTP/2 log
  -h, --httponly               Serve plain HTTP
  --http2only                  Serve HTTP/2, without HTTPS (not recommended)
  --verbose                    Slightly more verbose logging
  --eventserver=[HOST][:PORT]  SSE server address (for filesystem changes)
  --eventrefresh=DURATION      How often the event server should refresh
                               (the default is "` + defaultEventRefresh + `").
  -i                           Interactive mode
`)
}

// Parse the flags, return the default hostname
func handleFlags() string {
	// The short version of some flags
	var serveJustHTTPShort, autoRefreshShort, debugModeShort bool

	// The usage function that provides more help
	flag.Usage = usage

	// The default for running the redis server on Windows is to listen
	// to "localhost:port", but not just ":port".
	host := ""
	if runtime.GOOS == "windows" {
		host = "localhost"
		// Disable colors when logging, for some systems
		//log.SetFormatter(&log.TextFormatter{DisableColors: true})
	}

	// Commandline flag configuration

	flag.StringVar(&serverDir, "dir", ".", "Server directory")
	flag.StringVar(&serverAddr, "addr", "", "Server [host][:port] (ie \":443\")")
	flag.StringVar(&serverCert, "cert", "cert.pem", "Server certificate")
	flag.StringVar(&serverKey, "key", "key.pem", "Server key")
	flag.StringVar(&redisAddr, "redis", host+defaultRedisColonPort, "Redis [host][:port] (ie \""+defaultRedisColonPort+"\")")
	flag.IntVar(&redisDBindex, "dbindex", 0, "Redis database index")
	flag.StringVar(&serverConfScript, "conf", "serverconf.lua", "Server configuration")
	flag.StringVar(&serverHTTP2log, "http2log", "/dev/null", "HTTP/2 log")
	flag.BoolVar(&serveJustHTTP2, "http2only", false, "Serve HTTP/2, not HTTPS + HTTP/2")
	flag.BoolVar(&serveJustHTTP, "httponly", false, "Serve plain old HTTP")
	flag.BoolVar(&productionMode, "prod", false, "Production mode")
	flag.BoolVar(&debugMode, "debug", false, "Debug mode")
	flag.BoolVar(&verboseMode, "verbose", false, "Verbose logging")
	flag.BoolVar(&autoRefresh, "autorefresh", false, "Enable the auto-refresh feature")
	flag.StringVar(&eventAddr, "eventserver", "", "SSE [host][:port] (ie \""+defaultEventColonPort+"\")")
	flag.StringVar(&eventRefresh, "eventrefresh", defaultEventRefresh, "Event refresh interval in milliseconds (ie \""+defaultEventRefresh+"\")")
	flag.BoolVar(&interactiveMode, "i", false, "Interactive mode")

	// The short versions of some flags
	flag.BoolVar(&serveJustHTTPShort, "h", false, "Serve plain old HTTP")
	flag.BoolVar(&autoRefreshShort, "a", false, "Enable the auto-refresh feature")
	flag.BoolVar(&debugModeShort, "d", false, "Debug mode")

	flag.Parse()

	// Consider the long and short versions of some flags
	serveJustHTTP = serveJustHTTP || serveJustHTTPShort
	autoRefresh = autoRefresh || autoRefreshShort
	debugMode = debugMode || debugModeShort

	// Change several defaults if production mode is enabled
	if productionMode {
		// Use system directories
		serverDir = "/srv/algernon"
		serverCert = "/etc/algernon/cert.pem"
		serverKey = "/etc/algernon/key.pem"
	}

	// For backwards compatibility with earlier versions of algernon

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
