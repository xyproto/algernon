package engine

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/xyproto/algernon/cachemode"
	"github.com/xyproto/algernon/themes"
	"github.com/xyproto/datablock"
)

func generateUsageFunction(ac *Config) func() {
	return func() {
		fmt.Println("\n" + ac.versionString + "\n\n" + ac.description)
		// Possible arguments are also, for backward compatibility:
		// server dir, server addr, certificate file, key file, redis addr and redis db index
		// They are not mentioned here, but are possible to use, in that strict order.
		fmt.Println(`

Syntax:
  algernon [flags] [file or directory to serve] [host][:port]

Available flags:
  -h, --help                   This help text
  -v, --version                Application name and version
  --dir=DIRECTORY              Set the server directory
  --addr=[HOST][:PORT]         Server host and port ("` + ac.defaultWebColonPort + `" is default)
  -e, --dev                    Development mode: Enables Debug mode, uses
                               regular HTTP, Bolt and sets cache mode "dev".
  -p, --prod                   Serve HTTP/2+HTTPS on port 443. Serve regular
                               HTTP on port 80. Uses /srv/algernon for files.
                               Disables debug mode. Disables auto-refresh.
                               Enables server mode. Sets cache to "production".
  -a, --autorefresh            Enable event server and auto-refresh feature.
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
  --nocache                    Another way to disable the caching.
  --noheaders                  Don't use the security-related HTTP headers.
  --stricter                   Stricter HTTP headers (same origin policy).
  -n, --nobanner               Don't display a colorful banner at start.
  --ctrld                      Press ctrl-d twice to exit the REPL.
  --rawcache                   Disable cache compression.
  --watchdir=DIRECTORY         Enables auto-refresh for only this directory.
  --cert=FILENAME              TLS certificate, if using HTTPS.
  --key=FILENAME               TLS key, if using HTTPS.
  -d, --debug                  Enable debug mode (show errors in the browser).
  -b, --bolt                   Use "` + ac.defaultBoltFilename + `" for the Bolt database.
  --boltdb=FILENAME            Use a specific file for the Bolt database
  --redis=[HOST][:PORT]        Use "` + ac.defaultRedisColonPort + `" for the Redis database.
  --dbindex=INDEX              Redis database index (0 is default).
  --conf=FILENAME              Lua script with additional configuration.
  --log=FILENAME               Log to a file instead of to the console.
  --internal=FILENAME          Internal log file (can be a bit verbose).
  -t, --httponly               Serve regular HTTP.
  --http2only                  Serve HTTP/2, without HTTPS.
  --maria=DSN                  Use the given MariaDB or MySQL host/database.
  --mariadb=NAME               Use the given MariaDB or MySQL database name.
  --postgres=DSN               Use the given PostgreSQL host/database.
  --postgresdb=NAME            Use the given PostgreSQL database name.
  --clear                      Clear the default URI prefixes that are used
                               when handling permissions.
  -V, --verbose                Slightly more verbose logging.
  --eventserver=[HOST][:PORT]  SSE server address (filesystem changes as events).
  --eventrefresh=DURATION      How often the event server should refresh
                               (the default is "` + ac.defaultEventRefresh + `").
  --limit=N                    Limit clients to N requests per second
                               (the default is ` + ac.defaultLimitString + `).
  --nolimit                    Disable rate limiting.
  --nodb                       No database backend. (same as --boltdb=` + os.DevNull + `).
  --largesize=N                Threshold for not reading static files into memory, in bytes.
  --timeout=N                  Timeout when serving files, in seconds.
  -l, --lua                    Don't serve anything, just present the Lua REPL.
  --luapath                    Lua module directory (the default is ` + ac.defaultLuaModuleDirectory + `).
  -s, --server                 Server mode (disable debug + interactive mode).
  -q, --quiet                  Don't output anything to stdout or stderr.
  --servername=STRING          Custom HTTP header value for the Server field.
  -o, --open=EXECUTABLE        Open the served URL with ` + ac.defaultOpenExecutable + `, or with the
                               given application.
  -z, --quit                   Quit after the first request has been served.
  -m                           View the given Markdown file in the browser.
                               Quits after the file has been served once.
                               ("-m" is equivalent to "-q -o -z").
  --theme=NAME                 Builtin theme to use for Markdown, error pages,
                               directory listings and HyperApp apps.
                               Possible values are: light, dark, bw, redbox, wing,
                               material, neon or werc.
  -c, --statcache              Speed up responses by caching os.Stat.
                               Only use if served files will not be removed.
  --accesslog=FILENAME         Access log filename. Logged in Combined Log Format (CLF).
  --ncsa=FILENAME              Alternative access log filename. Logged in Common Log Format (NCSA).
  --cookiesecret=STRING        Secret that will be used for login cookies.
  -x, --simple                 Serve as regular HTTP, enable server mode and
                               disable all features that requires a database.
  --domain                     Serve files from the subdirectory with the same
                               name as the requested domain.
  -u                           Serve over QUIC.


Example usage:

  For auto-refreshing a webpage while developing:
    algernon --dev --httponly --debug --autorefresh --bolt --server . :4000

  Serve /srv/mydomain.com and /srv/otherweb.com over HTTP and HTTPS + HTTP/2:
    algernon -c --domain --server --cachesize 67108864 --prod /srv

  Serve the current dir over QUIC, port 7000, no banner:
    algernon -s -u -n . :7000

  Serve the current directory over HTTP, port 3000. No limits, cache,
  permissions or database connections:
    algernon -x
`)
	}
}

// Parse the flags, return the default hostname
func (ac *Config) handleFlags(serverTempDir string) {
	var (
		// The short version of some flags
		serveJustHTTPShort, autoRefreshShort, productionModeShort,
		debugModeShort, serverModeShort, useBoltShort, devModeShort,
		showVersionShort, quietModeShort, cacheFileStatShort, simpleModeShort,
		noBannerShort, quitAfterFirstRequestShort, verboseModeShort,
		serveJustQUICShort, serveNothingShort bool
		// Used when setting the cache mode
		cacheModeString string
		// Used if disabling cache compression
		rawCache bool
		// Used if disabling the database backend
		noDatabase bool
	)

	// The usage function that provides more help (for --help or -h)
	flag.Usage = generateUsageFunction(ac)

	// The default for running the redis server on Windows is to listen
	// to "localhost:port", but not just ":port".
	host := ""
	if runtime.GOOS == "windows" {
		host = "localhost"
		// Default Bolt database file
		ac.defaultBoltFilename = filepath.Join(serverTempDir, "algernon.db")
		// Default log file
		ac.defaultLogFile = filepath.Join(serverTempDir, "algernon.log")
	}

	// Commandline flag configuration

	flag.StringVar(&ac.serverDirOrFilename, "dir", ".", "Server directory")
	flag.StringVar(&ac.serverAddr, "addr", "", "Server [host][:port] (ie \":443\")")
	flag.StringVar(&ac.serverCert, "cert", "cert.pem", "Server certificate")
	flag.StringVar(&ac.serverKey, "key", "key.pem", "Server key")
	flag.StringVar(&ac.redisAddr, "redis", "", "Redis [host][:port] (ie \""+ac.defaultRedisColonPort+"\")")
	flag.IntVar(&ac.redisDBindex, "dbindex", 0, "Redis database index")
	flag.StringVar(&ac.serverConfScript, "conf", "serverconf.lua", "Server configuration")
	flag.StringVar(&ac.serverLogFile, "log", "", "Server log file")
	flag.StringVar(&ac.internalLogFilename, "internal", os.DevNull, "Internal log file")
	flag.BoolVar(&ac.serveJustHTTP2, "http2only", false, "Serve HTTP/2, not HTTPS + HTTP/2")
	flag.BoolVar(&ac.serveJustHTTP, "httponly", false, "Serve plain old HTTP")
	flag.BoolVar(&ac.productionMode, "prod", false, "Production mode")
	flag.BoolVar(&ac.debugMode, "debug", false, "Debug mode")
	flag.BoolVar(&ac.verboseMode, "verbose", false, "Verbose logging")
	flag.BoolVar(&ac.autoRefresh, "autorefresh", false, "Enable the auto-refresh feature")
	flag.StringVar(&ac.autoRefreshDir, "watchdir", "", "Directory to watch (also enables auto-refresh)")
	flag.StringVar(&ac.eventAddr, "eventserver", "", "SSE [host][:port] (ie \""+ac.defaultEventColonPort+"\")")
	flag.StringVar(&ac.eventRefresh, "eventrefresh", ac.defaultEventRefresh, "Event refresh interval (ie \""+ac.defaultEventRefresh+"\")")
	flag.BoolVar(&ac.serverMode, "server", false, "Server mode (disable interactive mode)")
	flag.StringVar(&ac.mariadbDSN, "maria", "", "MariaDB/MySQL connection string (DSN)")
	flag.StringVar(&ac.mariaDatabase, "mariadb", "", "MariaDB/MySQL database name")
	flag.StringVar(&ac.postgresDSN, "postgres", "", "PostgreSQL connection string (DSN)")
	flag.StringVar(&ac.postgresDatabase, "postgresdb", "", "PostgreSQL database name")
	flag.BoolVar(&ac.useBolt, "bolt", false, "Use the default Bolt filename")
	flag.StringVar(&ac.boltFilename, "boltdb", "", "Bolt database filename")
	flag.Int64Var(&ac.limitRequests, "limit", ac.defaultLimit, "Limit clients to a number of requests per second")
	flag.BoolVar(&ac.disableRateLimiting, "nolimit", false, "Disable rate limiting")
	flag.BoolVar(&ac.devMode, "dev", false, "Development mode")
	flag.BoolVar(&ac.showVersion, "version", false, "Version")
	flag.StringVar(&cacheModeString, "cache", "", "Cache everything but Amber, Lua, GCSS and Markdown")
	flag.Uint64Var(&ac.cacheSize, "cachesize", ac.defaultCacheSize, "Cache size, in bytes")
	flag.Uint64Var(&ac.largeFileSize, "largesize", ac.defaultLargeFileSize, "Threshold for not reading static files into memory, in bytes")
	flag.Uint64Var(&ac.writeTimeout, "timeout", 10, "Timeout when writing to a client, in seconds")
	flag.BoolVar(&ac.quietMode, "quiet", false, "Quiet")
	flag.BoolVar(&rawCache, "rawcache", false, "Disable cache compression")
	flag.StringVar(&ac.serverHeaderName, "servername", ac.versionString, "Server header name")
	flag.StringVar(&ac.profileCPU, "cpuprofile", "", "Write CPU profile to file")
	flag.StringVar(&ac.profileMem, "memprofile", "", "Write memory profile to file")
	flag.StringVar(&ac.traceFilename, "tracefile", "", "Write the trace to file")
	flag.BoolVar(&ac.cacheFileStat, "statcache", false, "Cache os.Stat")
	flag.BoolVar(&ac.serverAddDomain, "domain", false, "Look for files in the directory named the same as the hostname")
	flag.BoolVar(&ac.simpleMode, "simple", false, "Serve a directory of files over HTTP")
	flag.StringVar(&ac.openExecutable, "open", "", "Open URL after serving, with an application")
	flag.BoolVar(&ac.quitAfterFirstRequest, "quit", false, "Quit after the first request")
	flag.BoolVar(&ac.noCache, "nocache", false, "Disable caching")
	flag.BoolVar(&ac.noHeaders, "noheaders", false, "Don't set any HTTP headers by default")
	flag.BoolVar(&ac.stricterHeaders, "stricter", false, "Stricter HTTP headers")
	flag.StringVar(&ac.defaultTheme, "theme", themes.DefaultTheme, "Theme for Markdown and directory listings")
	flag.BoolVar(&ac.noBanner, "nobanner", false, "Don't show a banner at start")
	flag.BoolVar(&ac.ctrldTwice, "ctrld", false, "Press ctrl-d twice to exit")
	flag.BoolVar(&ac.serveJustQUIC, "quic", false, "Serve just QUIC")
	flag.BoolVar(&noDatabase, "nodb", false, "No database backend")
	flag.BoolVar(&ac.serveNothing, "lua", false, "Only present the Lua REPL")
	flag.StringVar(&ac.combinedAccessLogFilename, "accesslog", "", "Combined access log filename")
	flag.StringVar(&ac.commonAccessLogFilename, "ncsa", "", "NCSA access log filename")
	flag.BoolVar(&ac.clearDefaultPathPrefixes, "clear", false, "Clear the default URI prefixes for handling permissions")
	flag.StringVar(&ac.cookieSecret, "cookiesecret", "", "Secret to be used when setting and getting login cookies")
	flag.StringVar(&ac.luaModuleDirectory, "luapath", ac.defaultLuaModuleDirectory, "Lua module directory")

	// The short versions of some flags
	flag.BoolVar(&serveJustHTTPShort, "t", false, "Serve plain old HTTP")
	flag.BoolVar(&autoRefreshShort, "a", false, "Enable the auto-refresh feature")
	flag.BoolVar(&serverModeShort, "s", false, "Server mode (disable interactive mode)")
	flag.BoolVar(&useBoltShort, "b", false, "Use the default Bolt filename")
	flag.BoolVar(&productionModeShort, "p", false, "Production mode")
	flag.BoolVar(&debugModeShort, "d", false, "Debug mode")
	flag.BoolVar(&devModeShort, "e", false, "Development mode")
	flag.BoolVar(&showVersionShort, "v", false, "Version")
	flag.BoolVar(&verboseModeShort, "V", false, "Verbose")
	flag.BoolVar(&quietModeShort, "q", false, "Quiet")
	flag.BoolVar(&cacheFileStatShort, "c", false, "Cache os.Stat")
	flag.BoolVar(&simpleModeShort, "x", false, "Simple mode")
	flag.BoolVar(&ac.openURLAfterServing, "o", false, "Open URL after serving")
	flag.BoolVar(&quitAfterFirstRequestShort, "z", false, "Quit after the first request")
	flag.BoolVar(&ac.markdownMode, "m", false, "Markdown mode")
	flag.BoolVar(&noBannerShort, "n", false, "Don't show a banner at start")
	flag.BoolVar(&serveJustQUICShort, "u", false, "Serve just QUIC")
	flag.BoolVar(&serveNothingShort, "l", false, "Only present the Lua REPL")

	flag.Parse()

	// Accept both long and short versions of some flags
	ac.serveJustHTTP = ac.serveJustHTTP || serveJustHTTPShort
	ac.autoRefresh = ac.autoRefresh || autoRefreshShort
	ac.debugMode = ac.debugMode || debugModeShort
	ac.serverMode = ac.serverMode || serverModeShort
	ac.useBolt = ac.useBolt || useBoltShort
	ac.productionMode = ac.productionMode || productionModeShort
	ac.devMode = ac.devMode || devModeShort
	ac.showVersion = ac.showVersion || showVersionShort
	ac.quietMode = ac.quietMode || quietModeShort
	ac.cacheFileStat = ac.cacheFileStat || cacheFileStatShort
	ac.simpleMode = ac.simpleMode || simpleModeShort
	ac.openURLAfterServing = ac.openURLAfterServing || (ac.openExecutable != "")
	ac.quitAfterFirstRequest = ac.quitAfterFirstRequest || quitAfterFirstRequestShort
	ac.verboseMode = ac.verboseMode || verboseModeShort
	ac.noBanner = ac.noBanner || noBannerShort
	ac.serveJustQUIC = ac.serveJustQUIC || serveJustQUICShort
	ac.serveNothing = ac.serveNothing || serveNothingShort // "Lua mode"

	// Serve a single Markdown file once, and open it in the browser
	if ac.markdownMode {
		ac.quietMode = true
		ac.openURLAfterServing = true
		ac.quitAfterFirstRequest = true
	}

	// If only using the Lua REPL, don't include the banner, and don't serve anything
	if ac.serveNothing {
		ac.noBanner = true
		ac.debugMode = true
		ac.serverConfScript = ""
	}

	// Check if IGNOREEOF is set
	ignoreEOF, err := strconv.Atoi(os.Getenv("IGNOREEOF"))
	if err != nil {
		ignoreEOF = 0
	}
	if ignoreEOF > 1 {
		ac.ctrldTwice = true
	}

	// Disable verbose mode if quiet mode has been enabled
	if ac.quietMode {
		ac.verboseMode = false
	}

	// Enable cache compression unless raw cache is specified
	ac.cacheCompression = !rawCache

	ac.redisAddrSpecified = ac.redisAddr != ""
	if ac.redisAddr == "" {
		// The default host and port
		ac.redisAddr = host + ac.defaultRedisColonPort
	}

	// May be overridden by devMode
	if ac.serverMode {
		ac.debugMode = false
	}

	if noDatabase {
		ac.boltFilename = os.DevNull
	}

	// TODO: If flags are set in addition to -p or -e, don't override those
	//       when -p or -e is set.

	// Change several defaults if production mode is enabled
	switch {
	case ac.productionMode:
		// Use system directories
		ac.serverDirOrFilename = "/srv/algernon"
		ac.serverCert = "/etc/algernon/cert.pem"
		ac.serverKey = "/etc/algernon/key.pem"
		ac.cacheMode = cachemode.Production
		ac.serverMode = true
	case ac.devMode:
		// Change several defaults if development mode is enabled
		ac.useBolt = true
		ac.serveJustHTTP = true
		//serverLogFile = defaultLogFile
		ac.debugMode = true
		// TODO: Make it possible to set --limit to the default limit also when -e is used
		if ac.limitRequests == ac.defaultLimit {
			ac.limitRequests = 700 // Increase the rate limit considerably
		}
		ac.cacheMode = cachemode.Development
	case ac.simpleMode:
		ac.useBolt = true
		ac.boltFilename = os.DevNull
		ac.serveJustHTTP = true
		ac.serverMode = true
		ac.cacheMode = cachemode.Off
		ac.noCache = true
		ac.disableRateLimiting = true
		ac.clearDefaultPathPrefixes = true
		ac.noHeaders = true
		ac.writeTimeout = 3600 * 24
	}

	// If a watch directory is given, enable the auto refresh feature
	if ac.autoRefreshDir != "" {
		ac.autoRefresh = true
	}

	// If nocache is given, disable the cache
	if ac.noCache {
		ac.cacheMode = cachemode.Off
		ac.cacheFileStat = false
	}

	// Convert the request limit to a string
	ac.limitRequestsString = strconv.FormatInt(ac.limitRequests, 10)

	// If auto-refresh is enabled, change the caching
	if ac.autoRefresh {
		if cacheModeString == "" {
			// Disable caching by default, when auto-refresh is enabled
			ac.cacheMode = cachemode.Off
			ac.cacheFileStat = false
		}
	}

	// The cache flag overrides the settings from the other modes
	if cacheModeString != "" {
		ac.cacheMode = cachemode.New(cacheModeString)
	}

	// Disable cache entirely if cacheSize is set to 0
	if ac.cacheSize == 0 {
		ac.cacheMode = cachemode.Off
	}

	// Set cacheSize to 0 if the cache is disabled
	if ac.cacheMode == cachemode.Off {
		ac.cacheSize = 0
	}

	// If cache mode is unset, use the dev mode
	if ac.cacheMode == cachemode.Unset {
		ac.cacheMode = cachemode.Default
	}

	if ac.cacheMode == cachemode.Small {
		ac.cacheMaxEntitySize = ac.defaultCacheMaxEntitySize
	}

	// For backward compatibility with previous versions of Algernon
	// TODO: Remove, in favor of a better config/flag system
	serverAddrChanged := false
	if len(flag.Args()) >= 1 {
		// Only override the default server directory if Algernon can find it
		firstArg := flag.Args()[0]
		fs := datablock.NewFileStat(ac.cacheFileStat, ac.defaultStatCacheRefresh)
		// Interpret as a file or directory
		if fs.IsDir(firstArg) || fs.Exists(firstArg) {
			if strings.HasSuffix(firstArg, string(os.PathSeparator)) {
				ac.serverDirOrFilename = firstArg[:len(firstArg)-1]
			} else {
				ac.serverDirOrFilename = firstArg
			}
		} else if strings.Contains(firstArg, ":") {
			// Interpret as the server address
			ac.serverAddr = firstArg
			serverAddrChanged = true
		} else if _, err := strconv.Atoi(firstArg); err == nil { // no error
			// Is a number. Interpret as the server address
			ac.serverAddr = ":" + firstArg
			serverAddrChanged = true
		}
	}

	// TODO: Replace the code below with a good config/flag package.
	shift := 0
	if serverAddrChanged {
		shift = 1
	}
	if len(flag.Args()) >= 2 {
		secondArg := flag.Args()[1]
		if strings.Contains(secondArg, ":") {
			ac.serverAddr = secondArg
		} else if _, err := strconv.Atoi(secondArg); err == nil { // no error
			// Is a number. Interpret as the server address.
			ac.serverAddr = ":" + secondArg
		} else if len(flag.Args()) >= 3-shift {
			ac.serverCert = flag.Args()[2-shift]
		}
	}
	if len(flag.Args()) >= 4-shift {
		ac.serverKey = flag.Args()[3-shift]
	}
	if len(flag.Args()) >= 5-shift {
		ac.redisAddr = flag.Args()[4-shift]
		ac.redisAddrSpecified = true
	}
	if len(flag.Args()) >= 6-shift {
		// Convert the dbindex from string to int
		DBindex, err := strconv.Atoi(flag.Args()[5-shift])
		if err != nil {
			ac.redisDBindex = DBindex
		}
	}

	// Use the default openExecutable if none is set
	if ac.openURLAfterServing && ac.openExecutable == "" {
		ac.openExecutable = ac.defaultOpenExecutable
	}

	// Add the serverConfScript to the list of configuration scripts to be read and executed
	if ac.serverConfScript != "" && ac.serverConfScript != os.DevNull {
		ac.serverConfigurationFilenames = append(ac.serverConfigurationFilenames, ac.serverConfScript, filepath.Join(ac.serverDirOrFilename, ac.serverConfScript))
	}

	ac.serverHost = host

	// Get the absolute path of the Lua module directory, and quit with an error if it doesn't exist
	luaDir, err := filepath.Abs(ac.luaModuleDirectory)
	if err != nil {
		// Unlikely
		ac.fatalExit(err)
	}
	fileInfo, err := os.Stat(luaDir)
	if err != nil {
		// Unlikely
		ac.fatalExit(err)
	}
	if !fileInfo.IsDir() {
		// Not a directory
		ac.fatalExit(fmt.Errorf("%s is not a directory", luaDir))
	}
	// All is well
	ac.luaModuleDirectory = luaDir
}

// Set the values that has not been set by flags nor scripts (and can be set by both)
// Returns true if a "ready function" has been run.
func (ac *Config) finalConfiguration(host string) bool {

	// Set the server host and port (commandline flags overrides Lua configuration)
	if ac.serverAddr == "" {
		if ac.serverAddrLua != "" {
			ac.serverAddr = ac.serverAddrLua
		} else {
			ac.serverAddr = host + ac.defaultWebColonPort
		}
	}

	// Set the event server host and port
	if ac.eventAddr == "" {
		ac.eventAddr = host + ac.defaultEventColonPort
	}

	// Turn off debug mode if production mode is enabled
	if ac.productionMode {
		// Turn off debug mode
		ac.debugMode = false
	}

	hasReadyFunction := ac.serverReadyFunctionLua != nil

	// Run the Lua function specified with the OnReady function, if available
	if hasReadyFunction {
		// Useful for outputting configuration information after both
		// configuration scripts have been run and flags have been parsed
		ac.serverReadyFunctionLua()
	}

	return hasReadyFunction
}
