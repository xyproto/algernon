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

// Run a Lua file as a HTTP handler. Also has access to the userstate and permissions.
// Returns an error if there was a problem with running lua script, otherwise nil.
func runLua(w http.ResponseWriter, req *http.Request, filename string, perm *permissions.Permissions, luapool *lStatePool) error {
	// Retrieve a Lua state
	L := luapool.Get()
	defer luapool.Put(L)

	// Retrieve the userstate
	userstate := perm.UserState()

	// Make basic functions, like print, available to the Lua script
	exportBasic(w, req, L, filename)

	// Make the functions related to userstate available to the Lua script
	exportUserstate(w, req, L, userstate)

	// Simpleredis sets
	exportSet(w, req, L, userstate)

	// Run the script
	if err := L.DoFile(filename); err != nil {
		log.Println(err)
		return err
	}

	return nil
}
