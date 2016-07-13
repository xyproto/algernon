package main

import (
	"bytes"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	bolt "github.com/xyproto/permissionbolt"
	redis "github.com/xyproto/permissions2"
	mariadb "github.com/xyproto/permissionsql"
	"github.com/xyproto/pinterface"
	"github.com/xyproto/simpleredis"
	"github.com/yuin/gopher-lua"
)

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

func serverInfo() string {
	var buf bytes.Buffer

	if !singleFileMode {
		buf.WriteString("Server directory:\t" + serverDir + "\n")
	} else {
		buf.WriteString("Filename:\t\t" + serverDir + "\n")
	}
	if !productionMode {
		buf.WriteString("Server address:\t\t" + serverAddr + "\n")
	} // else port 80 and 443
	if dbName == "" {
		buf.WriteString("Database:\t\tDisabled\n")
	} else {
		buf.WriteString("Database:\t\t" + dbName + "\n")
	}
	if luaServerFilename != "" {
		buf.WriteString("Server filename:\t" + luaServerFilename + "\n")
	}

	// Write the status of flags that can be toggled
	writeStatus(&buf, "Options", map[string]bool{
		"Debug":        debugMode,
		"Production":   productionMode,
		"Auto-refresh": autoRefreshMode,
		"Dev":          devMode,
		"Server":       serverMode,
		"StatCache":    cacheFileStat,
	})

	buf.WriteString("Cache mode:\t\t" + cacheMode.String() + "\n")
	if cacheSize != 0 {
		buf.WriteString(fmt.Sprintf("Cache size:\t\t%d bytes\n", cacheSize))
	}

	if serverLogFile != "" {
		buf.WriteString("Log file:\t\t" + serverLogFile + "\n")
	}
	if !(serveJustHTTP2 || serveJustHTTP) {
		buf.WriteString("TLS certificate:\t" + serverCert + "\n")
		buf.WriteString("TLS key:\t\t" + serverKey + "\n")
	}
	if autoRefreshMode {
		buf.WriteString("Event server:\t\t" + eventAddr + "\n")
	}
	if autoRefreshDir != "" {
		buf.WriteString("Only watching:\t\t" + autoRefreshDir + "\n")
	}
	if redisAddr != defaultRedisColonPort {
		buf.WriteString("Redis address:\t\t" + redisAddr + "\n")
	}
	if disableRateLimiting {
		buf.WriteString("Request limit:\t\tOff\n")
	} else {
		buf.WriteString(fmt.Sprintf("Request limit:\t\t%d/sec\n", limitRequests))
	}
	if redisDBindex != 0 {
		buf.WriteString(fmt.Sprintf("Redis database index:\t%d\n", redisDBindex))
	}
	if len(serverConfigurationFilenames) > 0 {
		buf.WriteString(fmt.Sprintf("Server configuration:\t%v\n", serverConfigurationFilenames))
	}
	if internalLogFilename != "/dev/null" {
		buf.WriteString("Internal log file:\t" + internalLogFilename + "\n")
	}
	infoString := buf.String()
	// Return without the final newline
	return infoString[:len(infoString)-1]
}

// Make functions related to server configuration and permissions available
// Can not handle perm == nil
func exportServerConfigFunctions(L *lua.LState, perm pinterface.IPermissions, filename string, luapool *lStatePool) {

	// Set a default host and port. Maybe useful for alg applications.
	L.SetGlobal("SetAddr", L.NewFunction(func(L *lua.LState) int {
		serverAddrLua = L.ToString(1)
		return 0 // number of results
	}))

	// Clear the default path prefixes. This makes everything public.
	L.SetGlobal("ClearPermissions", L.NewFunction(func(L *lua.LState) int {
		perm.Clear()
		return 0 // number of results
	}))

	// Registers a path prefix, for instance "/secret",
	// as having *user* rights.
	L.SetGlobal("AddUserPrefix", L.NewFunction(func(L *lua.LState) int {
		path := L.ToString(1)
		perm.AddUserPath(path)
		return 0 // number of results
	}))

	// Registers a path prefix, for instance "/secret",
	// as having *admin* rights.
	L.SetGlobal("AddAdminPrefix", L.NewFunction(func(L *lua.LState) int {
		path := L.ToString(1)
		perm.AddAdminPath(path)
		return 0 // number of results
	}))

	// Sets a Lua function as a custom "permissions denied" page handler.
	L.SetGlobal("DenyHandler", L.NewFunction(func(L *lua.LState) int {
		luaDenyFunc := L.ToFunction(1)

		// Custom handler for when permissions are denied
		perm.SetDenyFunction(func(w http.ResponseWriter, req *http.Request) {
			// Set up a new Lua state with the current http.ResponseWriter and *http.Request, without caching
			exportCommonFunctions(w, req, filename, perm, L, luapool, nil, nil, nil)

			// Then run the given Lua function
			L.Push(luaDenyFunc)
			if err := L.PCall(0, lua.MultRet, nil); err != nil {
				// Non-fatal error
				log.Error("Permission denied handler failed:", err)
				// Use the default permission handler from now on if the lua function fails
				perm.SetDenyFunction(redis.PermissionDenied)
				perm.DenyFunction()(w, req)
			}
		})
		return 0 // number of results
	}))

	// Sets a Lua function to be run once the server is done parsing configuration and arguments.
	L.SetGlobal("OnReady", L.NewFunction(func(L *lua.LState) int {
		luaReadyFunc := L.ToFunction(1)

		// Custom handler for when permissions are denied.
		// Put the *lua.LState in a closure.
		serverReadyFunctionLua = func() {
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
		serverLogFile = filename
		// Log as JSON by default
		log.SetFormatter(&log.JSONFormatter{})
		// Log to stderr if an empty filename is given
		if filename == "" {
			log.SetOutput(os.Stderr)
			L.Push(lua.LBool(true))
			return 1 // number of results
		}
		// Try opening/creating the given filename, for appending
		f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, defaultPermissions)
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
		serverfilename := filepath.Join(filepath.Dir(filename), givenFilename)
		if !fs.exists(filename) {
			log.Error("Could not find", serverfilename)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		luaServerFilename = serverfilename
		L.Push(lua.LBool(true))
		return 1 // number of results
	}))

	L.SetGlobal("ServerInfo", L.NewFunction(func(L *lua.LState) int {
		// Return the string, but drop the final newline
		L.Push(lua.LString(serverInfo()))
		return 1 // number of results
	}))

}

// Use one of the databases for the permission middleware,
// assign a name to dbName (used for the status output) and
// return a Permissions struct.
func aquirePermissions() (pinterface.IPermissions, error) {
	var (
		err  error
		perm pinterface.IPermissions
	)

	// If Bolt is to be used and no filename is given
	if useBolt && (boltFilename == "") {
		boltFilename = defaultBoltFilename
	}

	if boltFilename != "" {
		// New permissions middleware, using a Bolt database
		perm, err = bolt.NewWithConf(boltFilename)
		if err != nil {
			if err.Error() == "timeout" {
				tempFile, err := ioutil.TempFile("", "algernon")
				if err != nil {
					log.Fatal("Unable to find a temporary file to use:", err)
				} else {
					boltFilename = tempFile.Name() + ".db"
				}
			} else {
				log.Errorf("Could not use Bolt as database backend: %s", err)
			}
		} else {
			dbName = "Bolt (" + boltFilename + ")"
		}
		// Try the new database filename if there was a timeout
		if dbName == "" && boltFilename != defaultBoltFilename {
			perm, err = bolt.NewWithConf(boltFilename)
			if err != nil {
				if err.Error() == "timeout" {
					log.Error("The Bolt database timed out!")
				} else {
					log.Errorf("Could not use Bolt as database backend: %s", err)
				}
			} else {
				dbName = "Bolt, temporary"
			}
		}
	}
	if dbName == "" && mariadbDSN != "" {
		// New permissions middleware, using a MariaDB/MySQL database
		perm, err = mariadb.NewWithDSN(mariadbDSN, mariaDatabase)
		if err != nil {
			log.Errorf("Could not use MariaDB/MySQL as database backend: %s", err)
		} else {
			// The connection string may contain a password, so don't include it in the dbName
			dbName = "MariaDB/MySQL"
		}
	}
	if dbName == "" && mariaDatabase != "" {
		// Given a database, but not a host, connect to localhost
		// New permissions middleware, using a MariaDB/MySQL database
		perm, err = mariadb.NewWithConf("test:@127.0.0.1/" + mariaDatabase)
		if err != nil {
			if mariaDatabase != "" {
				log.Errorf("Could not use MariaDB/MySQL as database backend: %s", err)
			} else {
				log.Warnf("Could not use MariaDB/MySQL as database backend: %s", err)
			}
		} else {
			// The connection string may contain a password, so don't include it in the dbName
			dbName = "MariaDB/MySQL"
		}
	}
	if dbName == "" {
		// New permissions middleware, using a Redis database
		if err := simpleredis.TestConnectionHost(redisAddr); err != nil {
			// Only output an error when a Redis host other than the default host+port was specified
			if redisAddrSpecified {
				if singleFileMode {
					log.Warnf("Could not use Redis as database backend: %s", err)
				} else {
					log.Errorf("Could not use Redis as database backend: %s", err)
				}
			}
		} else {
			var err error
			perm, err = redis.NewWithRedisConf2(redisDBindex, redisAddr)
			if err != nil {
				log.Warnf("Could not use Redis as database backend: %s", err)
			} else {
				dbName = "Redis"
			}
		}
	}
	if dbName == "" && boltFilename == "" {
		boltFilename = defaultBoltFilename
		perm, err = bolt.NewWithConf(boltFilename)
		if err != nil {
			if err.Error() == "timeout" {
				tempFile, err := ioutil.TempFile("", "algernon")
				if err != nil {
					log.Fatal("Unable to find a temporary file to use:", err)
				} else {
					boltFilename = tempFile.Name() + ".db"
				}
			} else {
				log.Errorf("Could not use Bolt as database backend: %s", err)
			}
		} else {
			dbName = "Bolt (" + boltFilename + ")"
		}
		// Try the new database filename if there was a timeout
		if boltFilename != defaultBoltFilename {
			perm, err = bolt.NewWithConf(boltFilename)
			if err != nil {
				if err.Error() == "timeout" {
					log.Error("The Bolt database timed out!")
				} else {
					log.Errorf("Could not use Bolt as database backend: %s", err)
				}
			} else {
				dbName = "Bolt, temporary"
			}
		}
	}
	if dbName == "" {
		// This may typically happen if Algernon is already running
		return nil, errors.New("Could not find a usable database backend")
	}

	return perm, nil
}
