package main

import (
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strings"

	"github.com/xyproto/permissions2"
	"github.com/xyproto/pinterface"
	"github.com/yuin/gopher-lua"
)

// Write a status message to a buffer, given a name and a bool
func writeStatus(buf *bytes.Buffer, title string, flags map[string]bool) {
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

// Make functions related to server configuration and permissions available
func exportServerConfigFunctions(L *lua.LState, perm pinterface.IPermissions, filename string, luapool *lStatePool) {

	// Registers a path prefix, for instance "/secret",
	// as having *admin* rights.
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
			exportCommonFunctions(w, req, filename, perm, L, luapool, nil, nil)

			// Then run the given Lua function
			L.Push(luaDenyFunc)
			if err := L.PCall(0, lua.MultRet, nil); err != nil {
				// Non-fatal error
				log.Error("Permission denied handler failed:", err)
				// Use the default permission handler from now on if the lua function fails
				perm.SetDenyFunction(permissions.PermissionDenied)
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

	L.SetGlobal("ServerInfo", L.NewFunction(func(L *lua.LState) int {
		var buf bytes.Buffer

		if !singleFileMode {
			buf.WriteString("Server directory:\t" + serverDir + "\n")
		} else {
			buf.WriteString("Filename:\t\t" + serverDir + "\n")
		}
		buf.WriteString("Server address:\t\t" + serverAddr + "\n")
		buf.WriteString("Database:\t\t" + dbName + "\n")

		// Write the status of flags that can be toggled
		writeStatus(&buf, "Options", map[string]bool{
			"Debug":        debugMode,
			"Production":   productionMode,
			"Auto-refresh": autoRefreshMode,
			"Dev":          devMode,
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
		buf.WriteString(fmt.Sprintf("Request limit:\t\t%d/sec\n", limitRequests))
		if redisDBindex != 0 {
			buf.WriteString(fmt.Sprintf("Redis database index:\t%d\n", redisDBindex))
		}
		buf.WriteString("Server configuration:\t" + serverConfScript + "\n")
		if internalLogFilename != "/dev/null" {
			buf.WriteString("Internal log file:\t" + internalLogFilename + "\n")
		}
		// Return the string, but drop the final newline
		L.Push(lua.LString(buf.String()[:len(buf.String())-1]))
		return 1 // number of results
	}))

}
