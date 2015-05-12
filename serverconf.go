package main

import (
	"bytes"
	log "github.com/Sirupsen/logrus"
	"net/http"
	"os"
	"strconv"

	"github.com/xyproto/permissions2"
	"github.com/yuin/gopher-lua"
)

// Write a status message to a buffer, given a name and a bool
func writeStatus(buf *bytes.Buffer, name string, enabled bool) {
	extraTab := ""
	if len(name) <= 14 { // Spartan way of lining up the columns
		extraTab = "\t"
	}
	if enabled {
		buf.WriteString(name + ":\t" + extraTab + "Enabled\n")
	} else {
		buf.WriteString(name + ":\t" + extraTab + "Disabled\n")
	}
}

// Make functions related to server configuration and permissions available
func exportServerConfigFunctions(L *lua.LState, perm *permissions.Permissions, filename string, luapool *lStatePool) {

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
			// Set up a new Lua state with the current http.ResponseWriter and *http.Request
			exportCommonFunctions(w, req, filename, perm, L, luapool, nil)

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

	// Set debug mode to true or false
	L.SetGlobal("SetDebug", L.NewFunction(func(L *lua.LState) int {
		debugMode = L.ToBool(1)
		return 0 // number of results
	}))

	// Set verbose to true or false
	L.SetGlobal("SetVerbose", L.NewFunction(func(L *lua.LState) int {
		verboseMode = L.ToBool(1)
		return 0 // number of results
	}))

	// Set a access log filename. If blank, the log will go to the console (or browser, if debug mode is set).
	L.SetGlobal("LogTo", L.NewFunction(func(L *lua.LState) int {
		filename := L.ToString(1)
		// Log to stderr if an empty filename is given
		if filename == "" {
			log.SetOutput(os.Stderr)
			L.Push(lua.LBool(true))
			return 1
		}
		// Try opening/creating the given filename, for appending
		f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatal(err)
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		// Set the file to log to and return
		log.SetOutput(f)
		L.Push(lua.LBool(true))
		return 1 // number of results
	}))

	L.SetGlobal("ServerInfo", L.NewFunction(func(L *lua.LState) int {
		// Using a buffer is faster for gathering larger amounts
		// of text but there is no need for optimization here.
		var buf bytes.Buffer
		buf.WriteString("Server directory:\t" + serverDir + "\n")
		buf.WriteString("Server address:\t\t" + serverAddr + "\n")
		writeStatus(&buf, "Debug mode", debugMode)
		writeStatus(&buf, "Auto-refresh", autoRefresh)
		writeStatus(&buf, "Production mode", productionMode)
		buf.WriteString("TLS certificate:\t" + serverCert + "\n")
		buf.WriteString("TLS key:\t\t" + serverKey + "\n")
		if autoRefresh {
			buf.WriteString("Event server:\t\t" + eventAddr + "\n")
		}
		if redisAddr != defaultRedisColonPort {
			buf.WriteString("Redis address:\t\t" + redisAddr + "\n")
		}
		if redisDBindex != 0 {
			buf.WriteString("Redis database index:\t" + strconv.Itoa(redisDBindex) + "\n")
		}
		buf.WriteString("Server configuration:\t" + serverConfScript + "\n")
		if serverHTTP2log != "/dev/null" {
			buf.WriteString("HTTP/2 log file:\t" + serverHTTP2log + "\n")
		}
		// Return the string, but drop the final newline
		L.Push(lua.LString(buf.String()[:len(buf.String())-1]))
		return 1 // number of results
	}))

}
