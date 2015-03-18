package main

import (
	"net/http"
	"strings"

	"github.com/xyproto/permissions2"
	"github.com/xyproto/simpleredis"
	"github.com/yuin/gopher-lua"
)

// Identifier for the List class in Lua
const lListClass = "LIST"

// Get the first argument, "self", and cast it from userdata to a list.
func checkList(L *lua.LState) *simpleredis.List {
	ud := L.CheckUserData(1)
	if list, ok := ud.Value.(*simpleredis.List); ok {
		return list
	}
	L.ArgError(1, "list expected")
	return nil
}

// Create a new list.
// id is the name of the list.
// dbindex is the Redis database index (typically 0).
func newList(L *lua.LState, pool *simpleredis.ConnectionPool, id string, dbindex int) (*lua.LUserData, error) {
	// Create a new simpleredis set
	list := simpleredis.NewList(pool, id)
	list.SelectDatabase(dbindex)
	// Create a new userdata struct
	ud := L.NewUserData()
	ud.Value = list
	L.SetMetatable(ud, L.GetTypeMetatable(lListClass))
	return ud, nil
}

// String representation
// Returns the entire list as a comma separated string
// tostring(list) -> string
func listToString(L *lua.LState) int {
	list := checkList(L) // arg 1
	all, err := list.GetAll()
	if err != nil {
		L.Push(lua.LString(""))
		return 1 // Number of returned values
	}
	L.Push(lua.LString(strings.Join(all, ", ")))
	return 1 // Number of returned values
}

// Add an element to the list
// list:add(string)
func listAdd(L *lua.LState) int {
	list := checkList(L) // arg 1
	value := L.ToString(2)
	list.Add(value)
	return 0 // Number of returned values
}

// Get all members of the list
// list::getall() -> table
func listGetAll(L *lua.LState) int {
	list := checkList(L) // arg 1
	all, err := list.GetAll()
	if err != nil {
		L.Push(lua.LString(""))
		return 1 // Number of returned values
	}
	L.Push(strings2table(L, all))
	return 1 // Number of returned values
}

// Get the last element of the list
// The returned value can be empty
// list::getlast() -> string
func listGetLast(L *lua.LState) int {
	list := checkList(L) // arg 1
	value, err := list.GetLast()
	if err != nil {
		value = ""
	}
	L.Push(lua.LString(value))
	return 1 // Number of returned values
}

// Get the N last elements of the list
// list::getlastn(number) -> table
func listGetLastN(L *lua.LState) int {
	list := checkList(L)    // arg 1
	n := int(L.ToNumber(2)) // arg 2
	results, err := list.GetLastN(n)
	if err != nil {
		L.Push(lua.LString(""))
		return 1 // Number of returned values
	}
	L.Push(strings2table(L, results))
	return 1 // Number of returned values
}

// Remove the list itself. Returns true if it worked out.
// list:remove() -> bool
func listRemove(L *lua.LState) int {
	list := checkList(L) // arg 1
	L.Push(lua.LBool(nil == list.Remove()))
	return 1 // Number of returned values
}

// The list methods that are to be registered
var listMethods = map[string]lua.LGFunction{
	"__tostring": listToString,
	"add":        listAdd,
	"getall":     listGetAll,
	"getlast":    listGetLast,
	"getlastn":   listGetLastN,
	"remove":     listRemove,
}

// Make functions related to HTTP requests and responses available to Lua scripts
func exportList(w http.ResponseWriter, req *http.Request, L *lua.LState, userstate *permissions.UserState) {
	pool := userstate.Pool()
	dbindex := userstate.DatabaseIndex()

	// Register the list class and the methods that belongs with it.
	mt := L.NewTypeMetatable(lListClass)
	mt.RawSetH(lua.LString("__index"), mt)
	L.SetFuncs(mt, listMethods)

	// The constructor for new lists takes a name and an optional redis db index
	L.SetGlobal("NewList", L.NewFunction(func(L *lua.LState) int {
		name := L.ToString(1)

		// Check if the optional argument is given
		localDBIndex := dbindex
		if L.GetTop() == 2 {
			localDBIndex = L.ToInt(2)
		}

		// Create a new list in Lua
		userdata, err := newList(L, pool, name, localDBIndex)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			L.Push(lua.LNumber(1))
			return 3 // Number of returned values
		}

		// Return the list object
		L.Push(userdata)
		return 1 // Number of returned values
	}))

}
