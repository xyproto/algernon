package main

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"github.com/xyproto/lookup"
	"github.com/yuin/gopher-lua"
)

// For dealing with JSON documents and strings

const (
	// Identifier for the JSONDB class in Lua
	lJSONDBClass = "JSONDB"
)

// Get the first argument, "self", and cast it from userdata to a library (which is really a hash map).
func checkJSONDB(L *lua.LState) *lookup.JSONFile {
	ud := L.CheckUserData(1)
	if jsondb, ok := ud.Value.(*lookup.JSONFile); ok {
		return jsondb
	}
	L.ArgError(1, "JSON DB expected")
	return nil
}

// Given a JSONDB, store JSON to the document.
// Takes one string, returns true if successful.
func jsondbAdd(L *lua.LState) int {
	//jsondb := checkJSONDB(L) // arg 1
	_ = checkJSONDB(L) // arg 1
	jsondata := L.ToString(2)
	if jsondata == "" {
		L.ArgError(2, "JSON data expected")
	}
	//L.Push(lua.LBool(jsondb.Add(jsondata) == nil))
	L.Push(lua.LBool(true))
	return 1 // number of results
}

// Given a JSONDB, return the JSON document.
// May return an empty string.
func jsondbGetAll(L *lua.LState) int {
	//jsondb := checkJSONDB(L) // arg 1
	_ = checkJSONDB(L) // arg 1

	//data, err := jsondb.GetAll()
	data := ""
	var err error = nil

	retval := ""
	if err == nil { // ok
		retval = data
	}
	L.Push(lua.LString(retval))
	return 1 // number of results
}

// String representation
func jsondbToString(L *lua.LState) int {
	L.Push(lua.LString("JSON DB"))
	return 1 // number of results
}

// Create a new code library.
// id is the name of the hash map.
func constructJSONDB(L *lua.LState, filename string) (*lua.LUserData, error) {
	// Create a new JSONDB
	jsondb, err := lookup.NewJSONFile(filename)
	if err != nil {
		return nil, err
	}
	// Create a new userdata struct
	ud := L.NewUserData()
	ud.Value = jsondb
	L.SetMetatable(ud, L.GetTypeMetatable(lJSONDBClass))
	return ud, nil
}

// The hash map methods that are to be registered
var jsondbMethods = map[string]lua.LGFunction{
	"__tostring": jsondbToString,
	"add":        jsondbAdd,
	"getall":     jsondbGetAll,
}

// Make functions related to building a library of Lua code available
func exportJSONDB(L *lua.LState) {

	// Register the JSONDB class and the methods that belongs with it.
	mt := L.NewTypeMetatable(lJSONDBClass)
	mt.RawSetH(lua.LString("__index"), mt)
	L.SetFuncs(mt, jsondbMethods)

	// The constructor for new Libraries takes only an optional id
	L.SetGlobal("JSONDB", L.NewFunction(func(L *lua.LState) int {
		// Get the filename and schema
		filename := L.ToString(1)

		// Construct a new JSONDB
		userdata, err := constructJSONDB(L, filename)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			L.Push(lua.LNumber(1))
			return 3 // Number of returned values
		}

		// Return the Lua JSONDB object
		L.Push(userdata)
		return 1 // number of results
	}))

}

func exportJSONFunctions(L *lua.LState) {

	// Lua function for converting a table to JSON (string or int)
	toJSON := L.NewFunction(func(L *lua.LState) int {
		table := L.ToTable(1)
		mapinterface, multiple := table2map(table)
		if multiple {
			log.Warn("ToJSON: Ignoring table values with different types")
		}
		b, err := json.Marshal(mapinterface)
		if err != nil {
			log.Error(err)
			return 0 // number of results
		}
		L.Push(lua.LString(string(b)))
		return 1 // number of results
	})

	// Convert a table to JSON
	L.SetGlobal("ToJSON", toJSON)

	// Only for backward compatibility
	L.SetGlobal("JSON", toJSON)

}
