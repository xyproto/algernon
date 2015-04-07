package main

import (
	log "github.com/Sirupsen/logrus"
	"net/http"
	"os"
	"strconv"

	"github.com/xyproto/permissions2"
	"github.com/yuin/gopher-lua"
)

var (
	DEBUG_MODE bool
)

// Make functions related to server configuration and permissions available
func exportServerConf(L *lua.LState, perm *permissions.Permissions, luapool *lStatePool, filename string) {

	// Registers a path prefix, for instance "/secret",
	// as having *admin* rights.
	L.SetGlobal("SetAddr", L.NewFunction(func(L *lua.LState) int {
		SERVER_ADDR = L.ToString(1)
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

	// Sets a lua script as a custom "permissions denied" page handler.
	L.SetGlobal("DenyHandler", L.NewFunction(func(L *lua.LState) int {
		luaDenyFunc := L.ToFunction(1)

		// Custom handler for when permissions are denied
		perm.SetDenyFunction(func(w http.ResponseWriter, req *http.Request) {
			// Set up a new Lua state with the current http.ResponseWriter and *http.Request
			exportCommonFunctions(w, req, filename, perm, L)

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

	// Set debug mode to true or false
	L.SetGlobal("SetDebug", L.NewFunction(func(L *lua.LState) int {
		DEBUG_MODE = L.ToBool(1)
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
		s := "Server directory:\t" + SERVER_DIR + "\n"
		s += "Server address:\t\t" + SERVER_ADDR + "\n"
		s += "TLS certificate:\t" + SERVER_CERT + "\n"
		s += "TLS key:\t\t" + SERVER_KEY + "\n"
		s += "Redis address:\t\t" + REDIS_ADDR + "\n"
		if REDIS_DB != 0 {
			s += "Redis database index:\t" + strconv.Itoa(REDIS_DB) + "\n"
		}
		s += "Server configuration:\t" + SERVER_CONF_SCRIPT + "\n"
		if SERVER_HTTP2_LOG != "/dev/null" {
			s += "HTTP/2 log file:\t" + SERVER_HTTP2_LOG + "\n"
		}
		L.Push(lua.LString(s))
		return 1 // number of results
	}))
}
