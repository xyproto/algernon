package main

import (
	"bytes"
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
func checkLibrary(L *lua.LState) pinterface.IHashMap {
	ud := L.CheckUserData(1)
	if hash, ok := ud.Value.(pinterface.IHashMap); ok {
		return hash
	}
	L.ArgError(1, "code library expected")
	return nil
}

// Given a namespace and an id, register Lua code.
// Takes three strings, returns a true if it is successful.
func libRegister(L *lua.LState) int {
	lualib := checkLibrary(L) // arg 1
	namespace := L.ToString(2)
	if namespace == "" {
		L.ArgError(2, "namespace expected")
	}
	id := L.ToString(3)
	if id == "" {
		L.ArgError(3, "id expected")
	}
	code := L.ToString(4)
	if code == "" {
		L.ArgError(4, "Lua code expected")
	}
	// Return true if there were no problems
	L.Push(lua.LBool(nil == lualib.Set(namespace, id, code)))
	return 1 // number of results
}

// Given a namespace and an id, return Lua code, or an empty string.
func libGetFunction(L *lua.LState) int {
	lualib := checkLibrary(L) // arg 1
	namespace := L.ToString(2)
	if namespace == "" {
		L.ArgError(2, "namespace expected")
	}
	id := L.ToString(3)
	if id == "" {
		L.ArgError(3, "id expected")
	}
	// Retrieve the Lua code from the HashMap
	code, err := lualib.Get(namespace, id)
	if err != nil {
		// Return an empty string if there was an error
		L.Push(lua.LString(""))
	}
	// Return the requested Lua code
	L.Push(lua.LString(code))
	return 1 // number of results
}

// Given a namespace, return all registered Lua code as a string.
// May return an empty string.
func libImport(L *lua.LState) int {
	var (
		allcode bytes.Buffer
		code    string
		err     error
	)
	lualib := checkLibrary(L) // arg 1
	namespace := L.ToString(2)
	if namespace == "" {
		L.ArgError(2, "namespace expected")
	}
	keys, err := lualib.GetAll()
	if err != nil {
		// Return an empty string if there was an error
		L.Push(lua.LString(""))
	}
	// Retrieve the code from all the keys in the given namespace
	for _, key := range keys {
		code, err = lualib.Get(namespace, key)
		if err != nil {
			continue
		}
		allcode.WriteString(code + "\n\n")
	}
	// Returned all the requested Lua code, joined with double newlines
	L.Push(lua.LString(allcode.String()))
	return 1 // number of results
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
	lualib, err := creator.NewHashMap(id)
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
	"set":        libRegister,
	"get":        libGetFunction,
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
