package engine

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	sqlite "github.com/terminar/permissionsqlite/v2"
	"github.com/xyproto/algernon/utils"
	lua "github.com/xyproto/gopher-lua"
	bolt "github.com/xyproto/permissionbolt/v2"
	redis "github.com/xyproto/permissions"
	mariadb "github.com/xyproto/permissionsql/v2"
	"github.com/xyproto/pinterface"
	postgres "github.com/xyproto/pstore"
	"github.com/xyproto/simpleredis/v2"
)

// Info returns a string with various info about the current configuration
func (ac *Config) Info() string {
	var sb strings.Builder

	if !ac.singleFileMode {
		sb.WriteString("Server directory:\t" + ac.serverDirOrFilename + "\n")
	} else {
		sb.WriteString("Filename:\t\t" + ac.serverDirOrFilename + "\n")
	}
	if !ac.productionMode {
		sb.WriteString("Server address:\t\t" + ac.serverAddr + "\n")
	} // else port 80 and 443
	if ac.dbName == "" {
		sb.WriteString("Database:\t\tDisabled\n")
	} else {
		sb.WriteString("Database:\t\t" + ac.dbName + "\n")
	}
	if ac.luaServerFilename != "" {
		sb.WriteString("Server filename:\t" + ac.luaServerFilename + "\n")
	}

	// Write the status of flags that can be toggled
	utils.WriteStatus(&sb, "Options", map[string]bool{
		"Debug":        ac.debugMode,
		"Production":   ac.productionMode,
		"Auto-refresh": ac.autoRefresh,
		"Dev":          ac.devMode,
		"Server":       ac.serverMode,
		"StatCache":    ac.cacheFileStat,
	})

	sb.WriteString("Cache mode:\t\t" + ac.cacheMode.String() + "\n")
	if ac.cacheSize != 0 {
		sb.WriteString(fmt.Sprintf("Cache size:\t\t%d bytes\n", ac.cacheSize))
	}

	if ac.serverLogFile != "" {
		sb.WriteString("Log file:\t\t" + ac.serverLogFile + "\n")
	}
	if !(ac.serveJustHTTP2 || ac.serveJustHTTP) {
		sb.WriteString("TLS certificate:\t" + ac.serverCert + "\n")
		sb.WriteString("TLS key:\t\t" + ac.serverKey + "\n")
	}
	if ac.autoRefresh {
		sb.WriteString("Event server:\t\t" + ac.eventAddr + "\n")
	}
	if ac.autoRefreshDir != "" {
		sb.WriteString("Only watching:\t\t" + ac.autoRefreshDir + "\n")
	}
	if ac.redisAddr != ac.defaultRedisColonPort {
		sb.WriteString("Redis address:\t\t" + ac.redisAddr + "\n")
	}
	if ac.disableRateLimiting {
		sb.WriteString("Request limit:\t\tOff\n")
	} else {
		sb.WriteString(fmt.Sprintf("Request limit:\t\t%d/sec per visitor\n", ac.limitRequests))
	}
	if ac.redisDBindex != 0 {
		sb.WriteString(fmt.Sprintf("Redis database index:\t%d\n", ac.redisDBindex))
	}
	if ac.largeFileSize > 0 {
		sb.WriteString(fmt.Sprintf("Large file threshold:\t%v bytes\n", ac.largeFileSize))
	}
	if ac.writeTimeout > 0 {
		sb.WriteString(fmt.Sprintf("Large file timeout:\t%v sec\n", ac.writeTimeout))
	}
	if len(ac.serverConfigurationFilenames) > 0 {
		sb.WriteString(fmt.Sprintf("Server configuration:\t%v\n", ac.serverConfigurationFilenames))
	}
	if ac.internalLogFilename != os.DevNull {
		sb.WriteString("Internal log file:\t" + ac.internalLogFilename + "\n")
	}
	return strings.TrimSpace(sb.String())
}

// LoadServerConfigFunctions makes functions related to server configuration and
// permissions available to the given Lua struct.
func (ac *Config) LoadServerConfigFunctions(L *lua.LState, filename string) error {
	if ac.perm == nil {
		return errors.New("perm is nil when loading server config functions")
	}

	// Set a default host and port. Maybe useful for alg applications.
	L.SetGlobal("SetAddr", L.NewFunction(func(L *lua.LState) int {
		ac.serverAddrLua = L.ToString(1)
		return 0 // number of results
	}))

	// Set the default cookie secret. This is for the server config, before
	// the userstate has been instanciated.
	L.SetGlobal("SetCookieSecret", L.NewFunction(func(L *lua.LState) int {
		ac.cookieSecret = L.ToString(1)
		return 0 // number of results
	}))

	// Get the default cookie secret. THis is for the server config, before
	// the userstate has been instanciated.
	L.SetGlobal("CookieSecret", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(ac.cookieSecret))
		return 1 // number of results
	}))

	// Clear the default path prefixes. This makes everything public.
	L.SetGlobal("ClearPermissions", L.NewFunction(func(_ *lua.LState) int {
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

	// Add a new reverse proxy given a: path prefix, endpoint and endpoint URL
	L.SetGlobal("AddReverseProxy", L.NewFunction(func(L *lua.LState) int {
		var rp ReverseProxy

		rp.PathPrefix = L.ToString(1)
		endpointURLString := L.ToString(2)

		parsedURL, err := url.Parse(endpointURLString)
		if err != nil {
			logrus.Errorf("could not parse endpoint URL: %s: %v", endpointURLString, err)
		}
		rp.Endpoint = *parsedURL

		if ac.reverseProxyConfig == nil {
			ac.reverseProxyConfig = NewReverseProxyConfig()
		}
		ac.reverseProxyConfig.Add(&rp)

		return 0 // number of results
	}))

	// Sets a Lua function as a custom "permissions denied" page handler.
	L.SetGlobal("DenyHandler", L.NewFunction(func(L *lua.LState) int {
		luaDenyFunc := L.ToFunction(1)

		// Custom handler for when permissions are denied
		ac.perm.SetDenyFunction(func(w http.ResponseWriter, req *http.Request) {
			// Set up a new Lua state with the current http.ResponseWriter and *http.Request, without caching
			ac.LoadCommonFunctions(w, req, filename, L, nil, nil)

			// Then run the given Lua function
			L.Push(luaDenyFunc)
			if err := L.PCall(0, lua.MultRet, nil); err != nil {
				// Non-fatal error
				logrus.Error("Permission denied handler failed:", err)
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
				logrus.Error("The OnReady function failed:", err)
			}
		}
		return 0 // number of results
	}))

	// Set a access log filename. If blank, the log will go to the console (or browser, if debug mode is set).
	L.SetGlobal("LogTo", L.NewFunction(func(L *lua.LState) int {
		filename := L.ToString(1)
		ac.serverLogFile = filename
		// Log as JSON by default
		logrus.SetFormatter(&logrus.JSONFormatter{})
		// Log to stderr if an empty filename is given
		if filename == "" {
			logrus.SetOutput(os.Stderr)
			L.Push(lua.LBool(true))
			return 1 // number of results
		}
		// Try opening/creating the given filename, for appending
		f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, ac.defaultPermissions)
		if err != nil {
			logrus.Error(err)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		// Set the file to log to and return
		logrus.SetOutput(f)
		L.Push(lua.LBool(true))
		return 1 // number of results
	}))

	// Use a single Lua file as the server, instead of directory structure
	L.SetGlobal("ServerFile", L.NewFunction(func(L *lua.LState) int {
		givenFilename := L.ToString(1)
		serverFilename := filepath.Join(filepath.Dir(filename), givenFilename)
		if !ac.fs.Exists(serverFilename) {
			logrus.Error("Could not find", serverFilename)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		ac.luaServerFilename = serverFilename
		L.Push(lua.LBool(true))
		return 1 // number of results
	}))

	// Set the server directory
	L.SetGlobal("ServerDir", L.NewFunction(func(L *lua.LState) int {
		givenDirectory := L.ToString(1)
		if !ac.fs.Exists(givenDirectory) {
			logrus.Error("Could not find", givenDirectory)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		ac.serverDirOrFilename = filepath.Clean(givenDirectory)
		L.Push(lua.LBool(true))
		return 1 // number of results
	}))

	L.SetGlobal("ServerInfo", L.NewFunction(func(L *lua.LState) int {
		// Return the string, but drop the final newline
		L.Push(lua.LString(ac.Info()))
		return 1 // number of results
	}))

	return nil
}

// DatabaseBackend tries to retrieve a database backend, using one of the
// available permission middleware packages. It assign a name to dbName
// (used for the status output) and returns a IPermissions struct.
func (ac *Config) DatabaseBackend() (pinterface.IPermissions, error) {
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
				tempFile, errTemp := os.CreateTemp("", "algernon")
				if errTemp != nil {
					logrus.Fatal("Unable to find a temporary file to use:", errTemp)
				}
				ac.boltFilename = tempFile.Name() + ".db"
			} else {
				logrus.Errorf("Could not use Bolt as database backend: %s", err)
			}
		} else {
			ac.dbName = "Bolt (" + ac.boltFilename + ")"
		}
		// Try the new database filename if there was a timeout
		if ac.dbName == "" && ac.boltFilename != ac.defaultBoltFilename {
			perm, err = bolt.NewWithConf(ac.boltFilename)
			if err != nil {
				if err.Error() == "timeout" {
					logrus.Error("The Bolt database timed out!")
				} else {
					logrus.Errorf("Could not use Bolt as database backend: %s", err)
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
			logrus.Errorf("Could not use MariaDB/MySQL as database backend: %s", err)
		} else {
			// The connection string may contain a password, so don't include it in the dbName
			ac.dbName = "MariaDB/MySQL"
		}
	}
	if ac.dbName == "" && ac.sqliteConnectionString != "" {
		// example connection string: sqlite.db&cache=shared&mode=memory
		perm, err = sqlite.NewWithConf(ac.sqliteConnectionString)
		if err != nil {
			logrus.Errorf("Could not use SQLite as database backend: %s", err)
		} else {
			// The connection string may contain a password, so don't include it in the dbName
			ac.dbName = "SQLite"
		}
	}
	if ac.dbName == "" && ac.mariaDatabase != "" {
		// Given a database, but not a host, connect to localhost
		// New permissions middleware, using a MariaDB/MySQL database
		perm, err = mariadb.NewWithConf("test:@127.0.0.1/" + ac.mariaDatabase)
		if err != nil {
			if ac.mariaDatabase != "" {
				logrus.Errorf("Could not use MariaDB/MySQL as database backend: %s", err)
			} else {
				logrus.Warnf("Could not use MariaDB/MySQL as database backend: %s", err)
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
			logrus.Errorf("Could not use PostgreSQL as database backend: %s", err)
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
				logrus.Errorf("Could not use PostgreSQL as database backend: %s", err)
			} else {
				logrus.Warnf("Could not use PostgreSQL as database backend: %s", err)
			}
		} else {
			// The connection string may contain a password, so don't include it in the dbName
			ac.dbName = "PostgreSQL"
		}
	}
	if ac.dbName == "" && ac.redisAddrSpecified {
		// New permissions middleware, using a Redis database
		logrus.Info("Testing redis connection")
		if err := simpleredis.TestConnectionHost(ac.redisAddr); err != nil {
			logrus.Info("Redis connection failed")
			// Only output an error when a Redis host other than the default host+port was specified
			if ac.singleFileMode {
				logrus.Warnf("Could not use Redis as database backend: %s", err)
			} else {
				logrus.Errorf("Could not use Redis as database backend: %s", err)
			}
		} else {
			logrus.Info("Redis connection worked out")
			var err error
			logrus.Info("Connecting to Redis...")
			perm, err = redis.NewWithRedisConf2(ac.redisDBindex, ac.redisAddr)
			if err != nil {
				logrus.Warnf("Could not use Redis as database backend: %s", err)
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
				tempFile, errTemp := os.CreateTemp("", "algernon")
				if errTemp != nil {
					logrus.Fatal("Unable to find a temporary file to use:", errTemp)
				}
				ac.boltFilename = tempFile.Name() + ".db"
			} else {
				logrus.Errorf("Could not use Bolt as database backend: %s", err)
			}
		} else {
			ac.dbName = "Bolt (" + ac.boltFilename + ")"
		}
		// Try the new database filename if there was a timeout
		if ac.boltFilename != ac.defaultBoltFilename {
			perm, err = bolt.NewWithConf(ac.boltFilename)
			if err != nil {
				if err.Error() == "timeout" {
					logrus.Error("The Bolt database timed out!")
				} else {
					logrus.Errorf("Could not use Bolt as database backend: %s", err)
				}
			} else {
				ac.dbName = "Bolt, temporary"
			}
		}
	}
	if ac.dbName == "" {
		// This may typically happen if Algernon is already running
		return nil, errors.New("could not find a usable database backend")
	}

	if ac.verboseMode {
		logrus.Info("Database backend success: " + ac.dbName)
	}

	if perm != nil && ac.clearDefaultPathPrefixes {
		perm.Clear()
	}

	return perm, nil
}
