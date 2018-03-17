package datastruct

import (
	"strings"

	"github.com/xyproto/algernon/lua/convert"
	"github.com/xyproto/pinterface"
	"github.com/yuin/gopher-lua"
)

// Identifier for the List class in Lua
const (
	lListClass = "LIST"

	// Prefix when indenting JSON
	indentPrefix = ""
)

// Get the first argument, "self", and cast it from userdata to a list.
func checkList(L *lua.LState) pinterface.IList {
	ud := L.CheckUserData(1)
	if list, ok := ud.Value.(pinterface.IList); ok {
		return list
	}
	L.ArgError(1, "list expected")
	return nil
}

// Create a new list.
// id is the name of the list.
// dbindex is the Redis database index (typically 0).
func newList(L *lua.LState, creator pinterface.ICreator, id string) (*lua.LUserData, error) {
	// Create a new list
	list, err := creator.NewList(id)
	if err != nil {
		return nil, err
	}
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
		// Return an empty table
		L.Push(L.NewTable())
		return 1 // Number of returned values
	}
	L.Push(convert.Strings2table(L, all))
	return 1 // Number of returned values
}

// Return the list as a JSON list (assumes the elements to be in JSON already)
// list::json() -> string
func listJSON(L *lua.LState) int {
	list := checkList(L) // arg 1
	all, err := list.GetAll()
	if err != nil {
		// Return an empty JSON list
		L.Push(lua.LString("[]"))
		return 1 // Number of returned values
	}

	L.Push(lua.LString("[" + strings.Join(all, ",") + "]"))
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
		// Return an empty table
		L.Push(L.NewTable())
		return 1 // Number of returned values
	}
	L.Push(convert.Strings2table(L, results))
	return 1 // Number of returned values
}

// Remove the list itself. Returns true if successful.
// list:remove() -> bool
func listRemove(L *lua.LState) int {
	list := checkList(L) // arg 1
	L.Push(lua.LBool(nil == list.Remove()))
	return 1 // Number of returned values
}

// Clear the list. Returns true if successful.
// list:clear() -> bool
func listClear(L *lua.LState) int {
	list := checkList(L) // arg 1
	L.Push(lua.LBool(nil == list.Clear()))
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
	"clear":      listClear,
	"json":       listJSON,
}

// LoadList makes functions related to HTTP requests and responses available to Lua scripts
func LoadList(L *lua.LState, creator pinterface.ICreator) {

	// Register the list class and the methods that belongs with it.
	mt := L.NewTypeMetatable(lListClass)
	mt.RawSetH(lua.LString("__index"), mt)
	L.SetFuncs(mt, listMethods)

	// The constructor for new lists takes a name and an optional redis db index
	L.SetGlobal("List", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)

		// Check if the optional argument is given
		if L.GetTop() == 2 {
			localDBIndex := L.ToInt(2)

			// Set the DB index, if possible
			switch rh := creator.(type) {
			case pinterface.IRedisCreator:
				rh.SelectDatabase(localDBIndex)
			}
		}

		// Create a new list in Lua
		userdata, err := newList(L, creator, name)
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
