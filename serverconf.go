package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/xyproto/datablock"
	postgres "github.com/xyproto/permissionHSTORE"
	bolt "github.com/xyproto/permissionbolt"
	redis "github.com/xyproto/permissions2"
	mariadb "github.com/xyproto/permissionsql"
	"github.com/xyproto/pinterface"
	"github.com/xyproto/simpleredis"
	"github.com/yuin/gopher-lua"
)

type algernonConfig struct {

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
	postgresDSN        string // connection string
	postgresDatabase   string // database name
	redisAddr          string
	redisDBindex       int
	redisAddrSpecified bool

	limitRequests       int64 // rate limit to this many requests per client per second
	disableRateLimiting bool

	// For the version flag
	showVersion bool

	// Caching
	cacheSize             uint64
	cacheMode             cacheModeSetting
	cacheCompression      bool
	cacheMaxEntitySize    uint64
	cacheCompressionSpeed bool // Compression speed over compactness
	noCache               bool
	noHeaders             bool

	// Output
	quietMode bool

	// If a single Lua file is provided, or Server() is used.
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

	perm pinterface.IPermissions

	luapool *lStatePool

	cache *datablock.FileCache

	// Workaround for rendering Pongo2 pages without concurrency issues
	pongomutex *sync.RWMutex

	// Temporary directory
	serverTempDir string
}

func newAlgernonConfig() *algernonConfig {
	return &algernonConfig{
		shutdownTimeout: 10 * time.Second,

		defaultWebColonPort:       ":3000",
		defaultRedisColonPort:     ":6379",
		defaultEventColonPort:     ":5553",
		defaultEventRefresh:       "350ms",
		defaultEventPath:          "/fs",
		defaultLimit:              10,
		defaultPermissions:        0660,
		defaultCacheSize:          1 * MiB,         // 1 MiB
		defaultCacheMaxEntitySize: 64 * KiB,        // 64 KB
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

		// Mutex for rendering Pongo2 pages
		pongomutex: &sync.RWMutex{},
	}
}

// Initialize a temporary directory, handle flags, output version and handle profiling
func (ac *algernonConfig) init() {
	// Temporary directory that might be used for logging, databases or file extraction
	serverTempDir, err := ioutil.TempDir("", "algernon")
	if err != nil {
		log.Fatalln(err)
	}

	// Set several configuration variables, based on the given flags and arguments
	ac.handleFlags(serverTempDir)

	// Version
	if ac.showVersion {
		if !ac.quietMode {
			fmt.Println(versionString)
		}
		os.Exit(0)
	}

	// CPU profiling
	if ac.profileCPU != "" {
		f, errProfile := os.Create(ac.profileCPU)
		if errProfile != nil {
			log.Fatal(errProfile)
		}
		go func() {
			log.Info("Profiling CPU usage")
			pprof.StartCPUProfile(f)
		}()
		atShutdown(func() {
			pprof.StopCPUProfile()
			log.Info("Done profiling CPU usage")
		})
	}

	// Memory profiling at server shutdown
	if ac.profileMem != "" {
		atShutdown(func() {
			f, errProfile := os.Create(ac.profileMem)
			if errProfile != nil {
				log.Fatal(errProfile)
			}
			defer f.Close()
			log.Info("Saving heap profile to ", ac.profileMem)
			pprof.WriteHeapProfile(f)
		})
	}

	ac.serverTempDir = serverTempDir

	// Create a cache struct for reading files (contains functions that can
	// be used for reading files, also when caching is disabled).
	// The final argument is for compressing with "fast" instead of "best".
	ac.cache = datablock.NewFileCache(ac.cacheSize, ac.cacheCompression, ac.cacheMaxEntitySize, ac.cacheCompressionSpeed)
}

// Write a status message to a buffer, given a name and a bool
func writeStatus(buf *bytes.Buffer, title string, flags map[string]bool) {

	// Check that at least one of the bools are true

	found := false
	for _, value := range flags {
		if value {
			found = true
			break
		}
	}
	if !found {
		return
	}

	// Write the overview to the buffer

	buf.WriteString(title + ":")
	// Spartan way of lining up the columns
	if len(title) < 7 {
		buf.WriteString("\t")
	}
	buf.WriteString("\t\t[")
	var enabledFlags []string
	// Add all enabled flags to the list
	for name, enabled := range flags {
		if enabled {
			enabledFlags = append(enabledFlags, name)
		}
	}
	buf.WriteString(strings.Join(enabledFlags, ", "))
	buf.WriteString("]\n")
}

func (ac *algernonConfig) Info() string {
	var buf bytes.Buffer

	if !ac.singleFileMode {
		buf.WriteString("Server directory:\t" + ac.serverDirOrFilename + "\n")
	} else {
		buf.WriteString("Filename:\t\t" + ac.serverDirOrFilename + "\n")
	}
	if !ac.productionMode {
		buf.WriteString("Server address:\t\t" + ac.serverAddr + "\n")
	} // else port 80 and 443
	if ac.dbName == "" {
		buf.WriteString("Database:\t\tDisabled\n")
	} else {
		buf.WriteString("Database:\t\t" + ac.dbName + "\n")
	}
	if ac.luaServerFilename != "" {
		buf.WriteString("Server filename:\t" + ac.luaServerFilename + "\n")
	}

	// Write the status of flags that can be toggled
	writeStatus(&buf, "Options", map[string]bool{
		"Debug":        ac.debugMode,
		"Production":   ac.productionMode,
		"Auto-refresh": ac.autoRefreshMode,
		"Dev":          ac.devMode,
		"Server":       ac.serverMode,
		"StatCache":    ac.cacheFileStat,
	})

	buf.WriteString("Cache mode:\t\t" + ac.cacheMode.String() + "\n")
	if ac.cacheSize != 0 {
		buf.WriteString(fmt.Sprintf("Cache size:\t\t%d bytes\n", ac.cacheSize))
	}

	if ac.serverLogFile != "" {
		buf.WriteString("Log file:\t\t" + ac.serverLogFile + "\n")
	}
	if !(ac.serveJustHTTP2 || ac.serveJustHTTP) {
		buf.WriteString("TLS certificate:\t" + ac.serverCert + "\n")
		buf.WriteString("TLS key:\t\t" + ac.serverKey + "\n")
	}
	if ac.autoRefreshMode {
		buf.WriteString("Event server:\t\t" + ac.eventAddr + "\n")
	}
	if ac.autoRefreshDir != "" {
		buf.WriteString("Only watching:\t\t" + ac.autoRefreshDir + "\n")
	}
	if ac.redisAddr != ac.defaultRedisColonPort {
		buf.WriteString("Redis address:\t\t" + ac.redisAddr + "\n")
	}
	if ac.disableRateLimiting {
		buf.WriteString("Request limit:\t\tOff\n")
	} else {
		buf.WriteString(fmt.Sprintf("Request limit:\t\t%d/sec\n", ac.limitRequests))
	}
	if ac.redisDBindex != 0 {
		buf.WriteString(fmt.Sprintf("Redis database index:\t%d\n", ac.redisDBindex))
	}
	if len(ac.serverConfigurationFilenames) > 0 {
		buf.WriteString(fmt.Sprintf("Server configuration:\t%v\n", ac.serverConfigurationFilenames))
	}
	if ac.internalLogFilename != "/dev/null" {
		buf.WriteString("Internal log file:\t" + ac.internalLogFilename + "\n")
	}
	infoString := buf.String()
	// Return without the final newline
	return infoString[:len(infoString)-1]
}

// Make functions related to server configuration and permissions available
// Can not handle perm == nil
func (ac *algernonConfig) exportServerConfigFunctions(L *lua.LState, filename string) {

	// Set a default host and port. Maybe useful for alg applications.
	L.SetGlobal("SetAddr", L.NewFunction(func(L *lua.LState) int {
		ac.serverAddrLua = L.ToString(1)
		return 0 // number of results
	}))

	// Clear the default path prefixes. This makes everything public.
	L.SetGlobal("ClearPermissions", L.NewFunction(func(L *lua.LState) int {
		ac.perm.Clear()
		return 0 // number of results
	}))

	// Registers a path prefix, for instance "/secret",
	// as having *user* rights.
	L.SetGlobal("AddUserPrefix", L.NewFunction(func(L *lua.LState) int {
		path := L.ToString(1)
		ac.perm.AddUserPath(path)
		return 0 // number of results
	}))

	// Registers a path prefix, for instance "/secret",
	// as having *admin* rights.
	L.SetGlobal("AddAdminPrefix", L.NewFunction(func(L *lua.LState) int {
		path := L.ToString(1)
		ac.perm.AddAdminPath(path)
		return 0 // number of results
	}))

	// Sets a Lua function as a custom "permissions denied" page handler.
	L.SetGlobal("DenyHandler", L.NewFunction(func(L *lua.LState) int {
		luaDenyFunc := L.ToFunction(1)

		// Custom handler for when permissions are denied
		ac.perm.SetDenyFunction(func(w http.ResponseWriter, req *http.Request) {
			// Set up a new Lua state with the current http.ResponseWriter and *http.Request, without caching
			ac.exportCommonFunctions(w, req, filename, L, nil, nil)

			// Then run the given Lua function
			L.Push(luaDenyFunc)
			if err := L.PCall(0, lua.MultRet, nil); err != nil {
				// Non-fatal error
				log.Error("Permission denied handler failed:", err)
				// Use the default permission handler from now on if the lua function fails
				ac.perm.SetDenyFunction(redis.PermissionDenied)
				ac.perm.DenyFunction()(w, req)
			}
		})
		return 0 // number of results
	}))

	// Sets a Lua function to be run once the server is done parsing configuration and arguments.
	L.SetGlobal("OnReady", L.NewFunction(func(L *lua.LState) int {
		luaReadyFunc := L.ToFunction(1)

		// Custom handler for when permissions are denied.
		// Put the *lua.LState in a closure.
		ac.serverReadyFunctionLua = func() {
			// Run the given Lua function
			L.Push(luaReadyFunc)
			if err := L.PCall(0, lua.MultRet, nil); err != nil {
				// Non-fatal error
				log.Error("The OnReady function failed:", err)
			}
		}
		return 0 // number of results
	}))

	// Set a access log filename. If blank, the log will go to the console (or browser, if debug mode is set).
	L.SetGlobal("LogTo", L.NewFunction(func(L *lua.LState) int {
		filename := L.ToString(1)
		ac.serverLogFile = filename
		// Log as JSON by default
		log.SetFormatter(&log.JSONFormatter{})
		// Log to stderr if an empty filename is given
		if filename == "" {
			log.SetOutput(os.Stderr)
			L.Push(lua.LBool(true))
			return 1 // number of results
		}
		// Try opening/creating the given filename, for appending
		f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, ac.defaultPermissions)
		if err != nil {
			log.Error(err)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		// Set the file to log to and return
		log.SetOutput(f)
		L.Push(lua.LBool(true))
		return 1 // number of results
	}))

	// Use a single Lua file as the server, instead of directory structure
	L.SetGlobal("ServerFile", L.NewFunction(func(L *lua.LState) int {
		givenFilename := L.ToString(1)
		serverFilename := filepath.Join(filepath.Dir(filename), givenFilename)
		if !fs.Exists(serverFilename) {
			log.Error("Could not find", serverFilename)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		ac.luaServerFilename = serverFilename
		L.Push(lua.LBool(true))
		return 1 // number of results
	}))

	L.SetGlobal("ServerInfo", L.NewFunction(func(L *lua.LState) int {
		// Return the string, but drop the final newline
		L.Push(lua.LString(ac.Info()))
		return 1 // number of results
	}))

}

// Use one of the databases for the permission middleware,
// assign a name to dbName (used for the status output) and
// return a Permissions struct.
func (ac *algernonConfig) databaseBackend() (pinterface.IPermissions, error) {
	var (
		err  error
		perm pinterface.IPermissions
	)

	// If Bolt is to be used and no filename is given
	if ac.useBolt && (ac.boltFilename == "") {
		ac.boltFilename = ac.defaultBoltFilename
	}

	if ac.boltFilename != "" {
		// New permissions middleware, using a Bolt database
		perm, err = bolt.NewWithConf(ac.boltFilename)
		if err != nil {
			if err.Error() == "timeout" {
				tempFile, errTemp := ioutil.TempFile("", "algernon")
				if errTemp != nil {
					log.Fatal("Unable to find a temporary file to use:", errTemp)
				} else {
					ac.boltFilename = tempFile.Name() + ".db"
				}
			} else {
				log.Errorf("Could not use Bolt as database backend: %s", err)
			}
		} else {
			ac.dbName = "Bolt (" + ac.boltFilename + ")"
		}
		// Try the new database filename if there was a timeout
		if ac.dbName == "" && ac.boltFilename != ac.defaultBoltFilename {
			perm, err = bolt.NewWithConf(ac.boltFilename)
			if err != nil {
				if err.Error() == "timeout" {
					log.Error("The Bolt database timed out!")
				} else {
					log.Errorf("Could not use Bolt as database backend: %s", err)
				}
			} else {
				ac.dbName = "Bolt, temporary"
			}
		}
	}
	if ac.dbName == "" && ac.mariadbDSN != "" {
		// New permissions middleware, using a MariaDB/MySQL database
		perm, err = mariadb.NewWithDSN(ac.mariadbDSN, ac.mariaDatabase)
		if err != nil {
			log.Errorf("Could not use MariaDB/MySQL as database backend: %s", err)
		} else {
			// The connection string may contain a password, so don't include it in the dbName
			ac.dbName = "MariaDB/MySQL"
		}
	}
	if ac.dbName == "" && ac.mariaDatabase != "" {
		// Given a database, but not a host, connect to localhost
		// New permissions middleware, using a MariaDB/MySQL database
		perm, err = mariadb.NewWithConf("test:@127.0.0.1/" + ac.mariaDatabase)
		if err != nil {
			if ac.mariaDatabase != "" {
				log.Errorf("Could not use MariaDB/MySQL as database backend: %s", err)
			} else {
				log.Warnf("Could not use MariaDB/MySQL as database backend: %s", err)
			}
		} else {
			// The connection string may contain a password, so don't include it in the dbName
			ac.dbName = "MariaDB/MySQL"
		}
	}
	if ac.dbName == "" && ac.postgresDSN != "" {
		// New permissions middleware, using a PostgreSQL database
		perm, err = postgres.NewWithDSN(ac.postgresDSN, ac.postgresDatabase)
		if err != nil {
			log.Errorf("Could not use PostgreSQL as database backend: %s", err)
		} else {
			// The connection string may contain a password, so don't include it in the dbName
			ac.dbName = "PostgreSQL"
		}
	}
	if ac.dbName == "" && ac.postgresDatabase != "" {
		// Given a database, but not a host, connect to localhost
		// New permissions middleware, using a PostgreSQL database
		perm, err = postgres.NewWithConf("postgres:@127.0.0.1/" + ac.postgresDatabase)
		if err != nil {
			if ac.postgresDatabase != "" {
				log.Errorf("Could not use PostgreSQL as database backend: %s", err)
			} else {
				log.Warnf("Could not use PostgreSQL as database backend: %s", err)
			}
		} else {
			// The connection string may contain a password, so don't include it in the dbName
			ac.dbName = "PostgreSQL"
		}
	}
	if ac.dbName == "" && ac.redisAddr != "" {
		// New permissions middleware, using a Redis database
		log.Info("Testing redis connection")
		if err := simpleredis.TestConnectionHost(ac.redisAddr); err != nil {
			log.Info("Redis connection failed")
			// Only output an error when a Redis host other than the default host+port was specified
			if ac.redisAddrSpecified {
				if ac.singleFileMode {
					log.Warnf("Could not use Redis as database backend: %s", err)
				} else {
					log.Errorf("Could not use Redis as database backend: %s", err)
				}
			}
		} else {
			log.Info("Redis connection worked out")
			var err error
			log.Info("Connecting to Redis...")
			perm, err = redis.NewWithRedisConf2(ac.redisDBindex, ac.redisAddr)
			if err != nil {
				log.Warnf("Could not use Redis as database backend: %s", err)
			} else {
				ac.dbName = "Redis"
			}
		}
	}
	if ac.dbName == "" && ac.boltFilename == "" {
		ac.boltFilename = ac.defaultBoltFilename
		perm, err = bolt.NewWithConf(ac.boltFilename)
		if err != nil {
			if err.Error() == "timeout" {
				tempFile, errTemp := ioutil.TempFile("", "algernon")
				if errTemp != nil {
					log.Fatal("Unable to find a temporary file to use:", errTemp)
				} else {
					ac.boltFilename = tempFile.Name() + ".db"
				}
			} else {
				log.Errorf("Could not use Bolt as database backend: %s", err)
			}
		} else {
			ac.dbName = "Bolt (" + ac.boltFilename + ")"
		}
		// Try the new database filename if there was a timeout
		if ac.boltFilename != ac.defaultBoltFilename {
			perm, err = bolt.NewWithConf(ac.boltFilename)
			if err != nil {
				if err.Error() == "timeout" {
					log.Error("The Bolt database timed out!")
				} else {
					log.Errorf("Could not use Bolt as database backend: %s", err)
				}
			} else {
				ac.dbName = "Bolt, temporary"
			}
		}
	}
	if ac.dbName == "" {
		// This may typically happen if Algernon is already running
		return nil, errors.New("Could not find a usable database backend")
	}

	if ac.verboseMode {
		log.Info("Database backend success: " + ac.dbName)
	}

	return perm, nil
}
