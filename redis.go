package main

import (
	"net/http"

	"github.com/xyproto/permissions2"
	"github.com/xyproto/simpleredis"
	"github.com/yuin/gopher-lua"
)

// TODO: Use LUserData for storing the simpleredis types in Lua

/*

NewSet(name)
NewList(name)
NewHashMap(name)

Objects in Lua:

Set = {}
s = Set;

function Set:add (name)
	self.name = name
end
*/

// Make functions related to HTTP requests and responses available to Lua scripts
func exportRedisFunctions(w http.ResponseWriter, req *http.Request, L *lua.LState, userstate *permissions.UserState) {
	pool := userstate.Pool()
	dbindex := userstate.DatabaseIndex()

	// Return a Set in Lua
	L.SetGlobal("NewSet", L.NewFunction(func(L *lua.LState) int {
		name := L.ToString(1)
		set := simpleredis.NewSet(pool, name)
		set.SelectDatabase(dbindex)

		return 0 // number of results
	}))

	// TODO: Find a way to store the Go simpleredis Set in lua. Perhaps as a pointer.

	// WIP
	L.SetGlobal("Set::GetAll", L.NewFunction(func(L *lua.LState) int {
		// Dummy values
		results := []string{}
		var err error = nil
		//
		var table *lua.LTable
		if err != nil {
			table = L.NewTable()
		} else {
			table = strings2table(L, results)
		}
		L.Push(table)
		return 1 // number of results
	}))

}
