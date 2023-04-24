package engine

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/xyproto/algernon/cachemode"
	"github.com/xyproto/algernon/themes"
	"github.com/xyproto/algernon/utils"
	"github.com/xyproto/datablock"
	"github.com/xyproto/env"
)

// Parse the flags, return the default hostname
func (ac *Config) handleFlags(serverTempDir string) {
	var (
		// The short version of some flags
		serveJustHTTPShort, autoRefreshShort, productionModeShort,
		debugModeShort, serverModeShort, useBoltShort, devModeShort,
		showVersionShort, quietModeShort, cacheFileStatShort, simpleModeShort,
		noBannerShort, quitAfterFirstRequestShort, verboseModeShort,
		serveJustQUICShort, onlyLuaModeShort, redirectShort bool
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
	flag.StringVar(&ac.serverConfScript, "conf", "serverconf.lua", "Server configuration written in Lua")
	flag.StringVar(&ac.serverLogFile, "log", "", "Server log file")
	flag.StringVar(&ac.internalLogFilename, "internal", os.DevNull, "Internal log file")
	flag.BoolVar(&ac.serveJustHTTP2, "http2only", false, "Serve HTTP/2, not HTTPS + HTTP/2")
	flag.BoolVar(&ac.serveJustHTTP, "httponly", false, "Serve plain old HTTP")
	flag.BoolVar(&ac.productionMode, "prod", false, "Production mode (when running as a system service)")
	flag.BoolVar(&ac.debugMode, "debug", false, "Debug mode")
	flag.BoolVar(&ac.verboseMode, "verbose", false, "Verbose logging")
	flag.BoolVar(&ac.redirectHTTP, "redirect", false, "Redirect HTTP traffic to HTTPS if both are enabled")
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
	if quicEnabled {
		flag.BoolVar(&ac.serveJustQUIC, "quic", false, "Serve just QUIC")
	}
	flag.BoolVar(&noDatabase, "nodb", false, "No database backend")
	flag.BoolVar(&ac.onlyLuaMode, "lua", false, "Only present the Lua REPL")
	flag.StringVar(&ac.combinedAccessLogFilename, "accesslog", "", "Combined access log filename")
	flag.StringVar(&ac.commonAccessLogFilename, "ncsa", "", "NCSA access log filename")
	flag.BoolVar(&ac.clearDefaultPathPrefixes, "clear", false, "Clear the default URI prefixes for handling permissions")
	flag.StringVar(&ac.cookieSecret, "cookiesecret", "", "Secret to be used when setting and getting login cookies")
	flag.BoolVar(&ac.useCertMagic, "letsencrypt", false, "Use Let's Encrypt for all served domains and serve regular HTTPS")
	flag.StringVar(&ac.dirBaseURL, "dirbaseurl", "", "Base URL for the directory listing (optional)")

	// The short versions of some flags
	flag.BoolVar(&serveJustHTTPShort, "t", false, "Serve plain old HTTP")
	flag.BoolVar(&autoRefreshShort, "a", false, "Enable the auto-refresh feature")
	flag.BoolVar(&serverModeShort, "s", false, "Server mode (non-interactive)")
	flag.BoolVar(&useBoltShort, "b", false, "Use the default Bolt filename")
	flag.BoolVar(&productionModeShort, "p", false, "Production mode (when running as a system service)")
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
	if quicEnabled {
		flag.BoolVar(&serveJustQUICShort, "u", false, "Serve just QUIC")
	}
	flag.BoolVar(&onlyLuaModeShort, "l", false, "Only present the Lua REPL")
	flag.BoolVar(&redirectShort, "r", false, "Redirect HTTP traffic to HTTPS, if both are enabled")

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
	if quicEnabled {
		ac.serveJustQUIC = ac.serveJustQUIC || serveJustQUICShort
	}
	ac.onlyLuaMode = ac.onlyLuaMode || onlyLuaModeShort
	ac.redirectHTTP = ac.redirectHTTP || redirectShort

	// Serve a single Markdown file once, and open it in the browser
	if ac.markdownMode {
		ac.quietMode = true
		ac.openURLAfterServing = true
		ac.quitAfterFirstRequest = true
	}

	// If only using the Lua REPL, don't include the banner, and don't serve anything
	if ac.onlyLuaMode {
		ac.noBanner = true
		ac.debugMode = true
		ac.serverConfScript = ""
	}

	// Check if IGNOREEOF is set
	if ignoreEOF := env.Int("IGNOREEOF", 0); ignoreEOF > 1 {
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
		ac.serveJustHTTP = true
		// serverLogFile = defaultLogFile
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

	if ac.onlyLuaMode {
		// Use a random database, so that several lua REPLs can be started without colliding,
		// but only if the current default bolt database file can not be opened.
		if ac.boltFilename != os.DevNull && !utils.CanRead(ac.boltFilename) {
			tempFile, err := os.CreateTemp("", "algernon_repl*.db")
			if err == nil { // no issue
				ac.boltFilename = tempFile.Name()
			}
		}
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

	// Clean up path in ac.serverDirOrFilename
	// .Rel calls .Clean on the result.
	if pwd, err := os.Getwd(); err == nil { // no error
		if cleanPath, err := filepath.Rel(pwd, ac.serverDirOrFilename); err == nil { // no error
			ac.serverDirOrFilename = cleanPath
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

	// CertMagic and Let's Encrypt
	if ac.useCertMagic {
		log.Info("Use Cert Magic")
		if dirEntries, err := os.ReadDir(ac.serverDirOrFilename); err != nil {
			log.Error("Could not use Cert Magic:" + err.Error())
			ac.useCertMagic = false
		} else {
			// log.Infof("Looping over %v files", len(files))
			for _, dirEntry := range dirEntries {
				basename := filepath.Base(dirEntry.Name())
				dirOrSymlink := dirEntry.IsDir() || ((dirEntry.Type() & os.ModeSymlink) == os.ModeSymlink)
				// TODO: Confirm that the symlink is a symlink to a directory, if it's a symlink
				if dirOrSymlink && strings.Contains(basename, ".") && !strings.HasPrefix(basename, ".") && !strings.HasSuffix(basename, ".old") {
					ac.certMagicDomains = append(ac.certMagicDomains, basename)
				}
			}
			// Using Let's Encrypt implies --domain, to search for suitable directories in the directory to be served
			ac.serverAddDomain = true
		}
	}
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
