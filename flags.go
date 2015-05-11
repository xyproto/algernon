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
)

var (
	// List of configuration filenames to check
	serverConfigurationFilenames = []string{"/etc/algernon/server.lua"}

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
	debugMode, verboseMode, productionMode bool

	// For the Server-Sent Event (SSE) server
	eventAddr    string
	eventRefresh string // Event server refresh, ie "300ms"
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
  --dir=DIRECTORY              The server directory
  --addr=[HOST][:PORT]         Server host and port (ie. ":443")
  --prod                       Serve HTTP/2+HTTPS on port 443, serve regular
                               HTTP on port 80, use /srv/algernon as the server
							   directory and disable debug mode.
  --cert=FILENAME              TLS certificate, if using HTTPS
  --key=FILENAME               TLS key, if using HTTPS
  --redis=[HOST][:PORT]        Connect to a remote Redis database (ie. ":6379")
  --dbindex=INDEX              Redis database index (0 is default)
  --conf=FILENAME              Lua script with additional configuration
  --http2log=FILENAME          Save the verbose HTTP/2 log
  --httponly                   Serve plain HTTP
  --http2only                  Serve HTTP/2, without HTTPS (not recommended)
  --debug                      Enable debug mode
  --verbose                    Slightly more verbose logging
  --version                    Show application name and version
  --eventserver=[HOST][:PORT]  Start a Server-Sent Event (SSE) server for
                               pushing events whenever a file changes.
  --eventrefresh=DURATION      How often the event server should refresh
                               (ie. \"300ms\").
  --help                       Application help
`)
}

// Parse the flags, return the default hostname
func handleFlags() string {
	flag.Usage = usage

	// The default for running the redis server on Windows is to listen
	// to "localhost:port", but not just ":port".
	host := ""
	if runtime.GOOS == "windows" {
		host = "localhost"
	}

	// Commandline flag configuration

	flag.StringVar(&serverDir, "dir", ".", "Server directory")
	flag.StringVar(&serverAddr, "addr", "", "Server [host][:port] (ie \":443\")")
	flag.StringVar(&serverCert, "cert", "cert.pem", "Server certificate")
	flag.StringVar(&serverKey, "key", "key.pem", "Server key")
	flag.StringVar(&redisAddr, "redis", host+defaultRedisColonPort, "Redis [host][:port] (ie \":6379\")")
	flag.IntVar(&redisDBindex, "dbindex", 0, "Redis database index")
	flag.StringVar(&serverConfScript, "conf", "server.lua", "Server configuration")
	flag.StringVar(&serverHTTP2log, "http2log", "/dev/null", "HTTP/2 log")
	flag.BoolVar(&serveJustHTTP2, "http2only", false, "Serve HTTP/2, not HTTPS + HTTP/2")
	flag.BoolVar(&serveJustHTTP, "httponly", false, "Serve plain old HTTP")
	flag.BoolVar(&productionMode, "prod", false, "Production mode")
	flag.BoolVar(&debugMode, "debug", false, "Debug mode")
	flag.BoolVar(&verboseMode, "verbose", false, "Verbose logging")
	flag.StringVar(&eventAddr, "eventserver", "", "SSE [host][:port] (ie \":5553\")")
	flag.StringVar(&eventRefresh, "eventrefresh", "300ms", "Event refresh interval in milliseconds (ie \"300ms\")")

	flag.Parse()

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
