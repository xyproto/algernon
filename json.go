package main

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"github.com/yuin/gopher-lua"
)

// For dealing with JSON documents and strings

const (
	// Identifier for the JSONDB class in Lua
	lJSONDBClass = "JSONDB"
)

type JSONDB struct {
	filename string
	schema   *lua.LTable
}

func (j *JSONDB) Add(data string) error {
	println("TO IMPLEMENT: ADD", data, "TO", j.filename)
	return nil
}

func newJSONDB(filename string, schema *lua.LTable) (*JSONDB, error) {
	if err := touch(filename); err != nil {
		return nil, err
	}
	return &JSONDB{filename, schema}, nil
}

// Get the first argument, "self", and cast it from userdata to a library (which is really a hash map).
func checkJSONDB(L *lua.LState) *JSONDB {
	ud := L.CheckUserData(1)
	if jsondb, ok := ud.Value.(*JSONDB); ok {
		return jsondb
	}
	L.ArgError(1, "JSON DB expected")
	return nil
}

// Given a JSONDB, store JSON to the document.
// Takes one string, returns true if successful.
func jsondbAdd(L *lua.LState) int {
	jsondb := checkJSONDB(L) // arg 1
	jsondata := L.ToString(2)
	if jsondata == "" {
		L.ArgError(2, "JSON data expected")
	}
	L.Push(lua.LBool(jsondb.Add(jsondata) == nil))
	return 1 // number of results
}

// String representation
func jsondbToString(L *lua.LState) int {
	L.Push(lua.LString("JSON DB"))
	return 1 // number of results
}

// Create a new code library.
// id is the name of the hash map.
func constructJSONDB(L *lua.LState, filename string, schema *lua.LTable) (*lua.LUserData, error) {
	// Create a new JSONDB
	jsondb, err := newJSONDB(filename, schema)
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
		schema := L.ToTable(2)

		// Construct a new JSONDB
		userdata, err := constructJSONDB(L, filename, schema)
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

	// Convert a table to JSON
	L.SetGlobal("JSON", L.NewFunction(func(L *lua.LState) int {
		table := L.ToTable(1)
		mapinterface, multiple := table2map(table)
		if multiple {
			log.Warn("JSON: Ignoring table values with different types")
		}
		b, err := json.Marshal(mapinterface)
		if err != nil {
			log.Error(err)
			return 0 // number of results
		}
		L.Push(lua.LString(string(b)))
		return 1 // number of results
	}))

}
