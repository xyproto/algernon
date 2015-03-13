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
	L := luapool.Get()
	defer luapool.Put(L)

	// Make basic functions, like print, available to the Lua script
	exportBasicFunctions(w, req, L, filename)

	exportPermissions(w, req, L, perm)

	// Make the functions related to userstate available to the Lua script
	exportUserstate(w, req, L, perm.UserState())

	// Run the script
	if err := L.DoFile(filename); err != nil {
		log.Println(err)
		return err
	}

	return nil
}
