package main

import (
	"log"
	"net/http"

	"github.com/xyproto/permissions2"
	"github.com/yuin/gopher-lua"
)

func strings2table(L *lua.LState, sl []string) *lua.LTable {
	table := L.NewTable()
	for _, element := range sl {
		table.Append(lua.LString(element))
	}
	return table
}

// Return a *lua.LState object that contains several exposed functions
func luaStateWithCommonFunctions(w http.ResponseWriter, req *http.Request, filename string, perm *permissions.Permissions, luapool *lStatePool) *lua.LState {
	// Retrieve a Lua state
	L := luapool.Get()
	defer luapool.Put(L)

	// Retrieve the userstate
	userstate := perm.UserState()

	// Make basic functions, like print, available to the Lua script
	exportBasic(w, req, L, filename)

	// Functions for rendering markdown or amber
	exportRenderFunctions(w, req, L)

	// Make the functions related to userstate available to the Lua script
	exportUserstate(w, req, L, userstate)

	// Simpleredis data structures
	exportList(L, userstate)
	exportSet(L, userstate)
	exportHash(L, userstate)
	exportKeyValue(L, userstate)

	return L
}

// Run a Lua file as a HTTP handler. Also has access to the userstate and permissions.
// Returns an error if there was a problem with running the lua script, otherwise nil.
func runLua(w http.ResponseWriter, req *http.Request, filename string, perm *permissions.Permissions, luapool *lStatePool) error {
	// Retrieve a Lua state
	L := luaStateWithCommonFunctions(w, req, filename, perm, luapool)

	// Run the script
	if err := L.DoFile(filename); err != nil {
		// TODO: Customizable
		log.Println(err)
		return err
	}

	return nil
}

// Run a Lua string as a HTTP handler. Also has access to the userstate and permissions.
// Returns an error if there was a problem with running the lua script, otherwise nil.
func runLuaString(w http.ResponseWriter, req *http.Request, script string, perm *permissions.Permissions, luapool *lStatePool) error {
	// Retrieve a Lua state
	L := luaStateWithCommonFunctions(w, req, "", perm, luapool)

	// Run the script
	if err := L.DoString(script); err != nil {
		// TODO: Customizable
		log.Println(err)
		return err
	}

	return nil
}

// Run a Lua file as a configuration script. Also has access to the userstate and permissions.
// Returns an error if there was a problem with running the lua script, otherwise nil.
func runConfiguration(filename string, perm *permissions.Permissions, luapool *lStatePool) error {
	// Retrieve a Lua state
	L := luapool.Get()
	defer luapool.Put(L)

	// Retrieve the userstate
	userstate := perm.UserState()

	// Server configuration functions
	exportServerConf(L, perm, luapool, filename)

	// Simpleredis data structures (could be used for storing server stats)
	exportList(L, userstate)
	exportSet(L, userstate)
	exportHash(L, userstate)
	exportKeyValue(L, userstate)

	// Run the script
	if err := L.DoFile(filename); err != nil {
		// TODO: Customizable
		log.Println(err)
		return err
	}

	return nil
}
