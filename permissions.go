package main

import (
	"net/http"

	"github.com/xyproto/permissions2"
	"github.com/yuin/gopher-lua"
)

// Server-wide functions. Does not make sense in `index.lua`.

/*
LUA functions related to permissions
------------------------------------

These can be placed in `server.lua` in the directory where the server is instructed to run from.

* `ClearPaths()` resets the URL prefixes and sets every path as *public*.
* `AddUserPath(string)` adds a URL prefix as a path that has *user* rights.
* `AddAdminPath(string)` adds a URL prefix as a path that has *admin* rights.
* `DenyPage(string)` creates a new page for "permission denied". Takes a message or a HTML page.

*/

// Make functions related to permissions available to Lua scripts
func exportPermissions(w http.ResponseWriter, req *http.Request, L *lua.LState, perm *permissions.Permissions) {

	// Clear the default path prefixes. This makes everything public.
	L.SetGlobal("ClearPaths", L.NewFunction(func(L *lua.LState) int {
		perm.Clear()
		return 0 // number of results
	}))
	// Registers a path prefix, for instance "/secret", as having *user* rights.
	L.SetGlobal("AddUserPath", L.NewFunction(func(L *lua.LState) int {
		path := L.ToString(1)
		perm.AddUserPath(path)
		return 0 // number of results
	}))
	// Registers a path prefix, for instance "/secret", as having *admin* rights.
	L.SetGlobal("AddAdminPath", L.NewFunction(func(L *lua.LState) int {
		path := L.ToString(1)
		perm.AddAdminPath(path)
		return 0 // number of results
	}))
	// Sets a custom static "permissions denied" page.
	// Takes a string that is an HTML page.
	L.SetGlobal("DenyPage", L.NewFunction(func(L *lua.LState) int {
		html := L.ToString(1)
		// Custom handler for when permissions are denied
		perm.SetDenyFunction(func(w http.ResponseWriter, req *http.Request) {
			http.Error(w, html, http.StatusForbidden)
		})
		return 0 // number of results
	}))

}
