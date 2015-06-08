package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/xyproto/pinterface"
	"github.com/yuin/gopher-lua"
)

// Library class is for storing and loading Lua source code to and from a data structure.

const (
	defaultID = "__lua_code_library"

	// Identifier for the Library class in Lua
	lLibraryClass = "CODELIB"
)

// Get the first argument, "self", and cast it from userdata to a library (which is really a hash map).
func checkLibrary(L *lua.LState) pinterface.IKeyValue {
	ud := L.CheckUserData(1)
	if hash, ok := ud.Value.(pinterface.IKeyValue); ok {
		return hash
	}
	L.ArgError(1, "code library expected")
	return nil
}

// Given a namespace, register Lua code.
// Takes two strings, returns a true if it is successful.
func libAdd(L *lua.LState) int {
	lualib := checkLibrary(L) // arg 1
	namespace := L.ToString(2)
	if namespace == "" {
		L.ArgError(2, "namespace expected")
	}
	code := L.ToString(3)
	if code == "" {
		log.Warn("Empty Lua code given to codelib:add")
		L.Push(lua.LBool(false))
		return 1
	}
	// Append the new code to the old code, if any
	oldcode, err := lualib.Get(namespace)
	if err != nil {
		oldcode = ""
	} else {
		oldcode += "\n"
	}
	L.Push(lua.LBool(nil == lualib.Set(namespace, oldcode+code)))
	return 1 // number of results
}

// Given a namespace, register Lua code as the only code.
// Takes two strings, returns a true if it is successful.
func libSet(L *lua.LState) int {
	lualib := checkLibrary(L) // arg 1
	namespace := L.ToString(2)
	if namespace == "" {
		L.ArgError(2, "namespace expected")
	}
	code := L.ToString(3)
	if code == "" {
		log.Warn("Empty Lua code given to codelib:set")
		L.Push(lua.LBool(false))
		return 1
	}
	L.Push(lua.LBool(nil == lualib.Set(namespace, code)))
	return 1 // number of results
}

// Given a namespace, return Lua code, or an empty string.
func libGet(L *lua.LState) int {
	lualib := checkLibrary(L) // arg 1
	namespace := L.ToString(2)
	if namespace == "" {
		L.ArgError(2, "namespace expected")
	}
	// Retrieve the Lua code from the HashMap
	code, err := lualib.Get(namespace)
	if err != nil {
		// Return an empty string if there was an error
		L.Push(lua.LString(""))
	}
	// Return the requested Lua code
	L.Push(lua.LString(code))
	return 1 // number of results
}

// Given a namespace, fetch all registered Lua code as a string.
// Then run the Lua code for this LState.
// Returns true of successful.
func libImport(L *lua.LState) int {
	lualib := checkLibrary(L) // arg 1
	namespace := L.ToString(2)
	if namespace == "" {
		L.ArgError(2, "namespace expected")
	}

	code, err := lualib.Get(namespace)
	if err != nil {
		// Return false if there was an error
		L.Push(lua.LBool(false)) // error
		return 1
	}

	if err := L.DoString(code); err != nil {
		L.Close()

		log.Errorf("Error when importing Lua code:\n%s", err)
		L.Push(lua.LBool(false)) // error
		return 1                 // number of results
	}

	L.Push(lua.LBool(true)) // ok
	return 1                // number of results
}

// String representation
func libToString(L *lua.LState) int {
	L.Push(lua.LString("Code Library"))
	return 1 // number of results
}

// Clear the current code library
func libClear(L *lua.LState) int {
	lualib := checkLibrary(L) // arg 1
	L.Push(lua.LBool(nil == lualib.Remove()))
	return 1 // number of results
}

// Create a new code library.
// id is the name of the hash map.
func newCodeLibrary(L *lua.LState, creator pinterface.ICreator, id string) (*lua.LUserData, error) {
	// Create a new Lua Library (hash map)
	lualib, err := creator.NewKeyValue(id)
	if err != nil {
		return nil, err
	}
	// Create a new userdata struct
	ud := L.NewUserData()
	ud.Value = lualib
	L.SetMetatable(ud, L.GetTypeMetatable(lLibraryClass))
	return ud, nil
}

// The hash map methods that are to be registered
var libMethods = map[string]lua.LGFunction{
	"__tostring": libToString,
	"add":        libAdd,
	"set":        libSet,
	"get":        libGet,
	"import":     libImport,
	"clear":      libClear,
}

// Make functions related to building a library of Lua code available
func exportCodeLibrary(L *lua.LState, userstate pinterface.IUserState) {
	creator := userstate.Creator()

	// Register the Library class and the methods that belongs with it.
	mt := L.NewTypeMetatable(lLibraryClass)
	mt.RawSetH(lua.LString("__index"), mt)
	L.SetFuncs(mt, libMethods)

	// The constructor for new Libraries takes only an optional id
	L.SetGlobal("CodeLib", L.NewFunction(func(L *lua.LState) int {
		// Check if the optional argument is given
		id := defaultID
		if L.GetTop() == 1 {
			id = L.ToString(1)
		}

		// Create a new Library in Lua
		userdata, err := newCodeLibrary(L, creator, id)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			L.Push(lua.LNumber(1))
			return 3 // Number of returned values
		}

		// Return the hash map object
		L.Push(userdata)
		return 1 // number of results
	}))

}
