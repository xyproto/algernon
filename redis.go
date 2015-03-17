package main

import (
	"net/http"

	"github.com/xyproto/permissions2"
	"github.com/xyproto/simpleredis"
	"github.com/yuin/gopher-lua"
)

// Look at fileMethods in gopher-lua's iolib.go for a way to
// expose datastructures to lua.

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

func setToString(L *lua.LState) int {
	//set := checkSet(L)
	//if set.Type() == lSet {
	//	L.Push(LString(set.name))
	//} else {
	//	L.Push(LString(""))
	//}
	L.Push(lua.LString("untitled"))
	return 1
}

func setSet(L *lua.LState) int {
	return 0
}

func setGet(L *lua.LState) int {
	return 0
}

var setMethods = map[string]lua.LGFunction{
	"__tostring": setToString,
	"set": setSet,
	"get": setGet,
}

func setNew(L *lua.LState) {
	mt := L.NewTypeMetatable(lSetClass)
	mt.RawSetH(LString("__index:"), mt)
	L.SetFuncs(mt, setMethods)
	//mt.RawSetH(LString("lines"), L.NewClosure(fileLines, L.NewFunction(fileLinesIter)))
}

// Make functions related to HTTP requests and responses available to Lua scripts
func exportRedisFunctions(w http.ResponseWriter, req *http.Request, L *lua.LState, userstate *permissions.UserState) {
	pool := userstate.Pool()
	dbindex := userstate.DatabaseIndex()

	// Return a Set in Lua
	L.SetGlobal("newset", L.NewFunction(func(L *lua.LState) int {
		name := L.ToString(1)
		set := simpleredis.NewSet(pool, name)
		set.SelectDatabase(dbindex)

		//
		ud := lua.LUserData(&set)
		L.Push(ud)
		return 1 // number of results
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
