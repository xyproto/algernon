package main

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"github.com/xyproto/jpath"
	"github.com/yuin/gopher-lua"
	"io/ioutil"
	"path/filepath"
	"strings"
)

// For dealing with JSON documents and strings

const (
	// Identifier for the JFile class in Lua
	lJFileClass = "JFile"

	// Prefix when indenting JSON
	indentPrefix = ""
)

// Get the first argument, "self", and cast it from userdata to a library (which is really a hash map).
func checkJFile(L *lua.LState) *jpath.JFile {
	ud := L.CheckUserData(1)
	if jfile, ok := ud.Value.(*jpath.JFile); ok {
		return jfile
	}
	L.ArgError(1, "JSON file expected")
	return nil
}

// Takes a JFile, a JSON path (optional) and JSON data.
// Stores the JSON data. Returns true if successful.
func jfileAdd(L *lua.LState) int {
	jfile := checkJFile(L) // arg 1
	top := L.GetTop()
	jsonpath := "x"
	jsondata := ""
	if top == 2 {
		jsondata = L.ToString(2)
		if jsondata == "" {
			L.ArgError(2, "JSON data expected")
		}
	} else if top == 3 {
		jsonpath = L.ToString(2)
		// Check for { to help avoid allowing JSON data as a JSON path
		if jsonpath == "" || strings.Contains(jsonpath, "{") {
			L.ArgError(2, "JSON path expected")
		}
		jsondata = L.ToString(3)
		if jsondata == "" {
			L.ArgError(3, "JSON data expected")
		}
	}
	err := jfile.AddJSON(jsonpath, []byte(jsondata))
	if err != nil {
		if top == 2 || strings.HasPrefix(err.Error(), "invalid character") {
			log.Error("JSON data: ", err)
		} else {
			log.Error(err)
		}
	}
	L.Push(lua.LBool(err == nil))
	return 1 // number of results
}

// Takes a JFile and a JSON path.
// Returns a value or an empty string.
func jfileGet(L *lua.LState) int {
	jfile := checkJFile(L) // arg 1
	jsonpath := L.ToString(2)
	if jsonpath == "" {
		L.ArgError(2, "JSON path expected")
	}
	val, err := jfile.GetString(jsonpath)
	if err != nil {
		log.Error(err)
	}
	L.Push(lua.LString(val))
	return 1 // number of results
}

// Take a JFile, a JSON path and a string.
// Returns a value or an empty string.
func jfileSet(L *lua.LState) int {
	jfile := checkJFile(L) // arg 1
	jsonpath := L.ToString(2)
	if jsonpath == "" {
		L.ArgError(2, "JSON path expected")
	}
	sval := L.ToString(3)
	if sval == "" {
		L.ArgError(3, "String value expected")
	}
	err := jfile.SetString(jsonpath, sval)
	if err != nil {
		log.Error(err)
	}
	L.Push(lua.LBool(err == nil))
	return 1 // number of results
}

// Take a JFile and a JSON path.
// Remove a key from a map. Return true if successful.
func jfileDelKey(L *lua.LState) int {
	jfile := checkJFile(L) // arg 1
	jsonpath := L.ToString(2)
	if jsonpath == "" {
		L.ArgError(2, "JSON path expected")
	}
	err := jfile.DelKey(jsonpath)
	if err != nil {
		log.Error(err)
	}
	L.Push(lua.LBool(nil == err))
	return 1 // number of results
}

// Given a JFile, return the JSON document.
// May return an empty string.
func jfileJSON(L *lua.LState) int {
	jfile := checkJFile(L) // arg 1

	data, err := jfile.JSON()
	retval := ""
	if err == nil { // ok
		retval = string(data)
	}
	L.Push(lua.LString(retval))
	return 1 // number of results
}

//// String representation
//func jfileToString(L *lua.LState) int {
//	L.Push(lua.LString("JSON file"))
//	return 1 // number of results
//}

// Create a new JSON file
func constructJFile(L *lua.LState, filename string) (*lua.LUserData, error) {
	fullFilename := filename
	// Check if the file exists
	if !fs.exists(fullFilename) {
		// Create an empty JSON document/file
		if err := ioutil.WriteFile(fullFilename, []byte("[]\n"), defaultPermissions); err != nil {
			return nil, err
		}
	}
	// Create a new JFile
	jfile, err := jpath.NewFile(fullFilename)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	// Create a new userdata struct
	ud := L.NewUserData()
	ud.Value = jfile
	L.SetMetatable(ud, L.GetTypeMetatable(lJFileClass))
	return ud, nil
}

// The hash map methods that are to be registered
var jfileMethods = map[string]lua.LGFunction{
	"__tostring": jfileJSON,
	"add":        jfileAdd,
	"get":        jfileGet,
	"set":        jfileSet,
	"delkey":     jfileDelKey,
	"string":     jfileJSON, // undocumented
}

// Make functions related to building a library of Lua code available
func exportJFile(L *lua.LState, scriptdir string) {

	// Register the JFile class and the methods that belongs with it.
	mt := L.NewTypeMetatable(lJFileClass)
	mt.RawSetH(lua.LString("__index"), mt)
	L.SetFuncs(mt, jfileMethods)

	// The constructor for new Libraries takes only an optional id
	L.SetGlobal("JFile", L.NewFunction(func(L *lua.LState) int {
		// Get the filename and schema
		filename := L.ToString(1)

		// Construct a new JFile
		userdata, err := constructJFile(L, filepath.Join(scriptdir, filename))
		if err != nil {
			log.Error(err)
			L.Push(lua.LString(err.Error()))
			return 1 // Number of returned values
		}

		// Return the Lua JFile object
		L.Push(userdata)
		return 1 // number of results
	}))

}

func exportJSONFunctions(L *lua.LState) {

	// Lua function for converting a table to JSON (string or int)
	toJSON := L.NewFunction(func(L *lua.LState) int {
		var (
			b   []byte
			err error
		)
		table := L.ToTable(1)

		// Convert the Lua table to a map that can be used when converting
		// to JSON (map[string]interface{})
		mapinterface := table2interfacemap(table)

		// If an optional argument is supplied, indent the given number of spaces
		if L.GetTop() == 2 {
			indentLevel := L.ToInt(2)
			indentString := ""
			for i := 0; i < indentLevel; i++ {
				indentString += " "
			}
			b, err = json.MarshalIndent(mapinterface, indentPrefix, indentString)
		} else {
			b, err = json.Marshal(mapinterface)
		}
		if err != nil {
			log.Error(err)
			return 0 // number of results
		}
		L.Push(lua.LString(string(b)))
		return 1 // number of results
	})

	// Convert a table to JSON
	L.SetGlobal("toJSON", toJSON)
	L.SetGlobal("ToJSON", toJSON) // Alias for backward compatibility
	L.SetGlobal("JSON", toJSON)   // Alias for backward compatibility

}
