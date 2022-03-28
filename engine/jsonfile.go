package engine

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/lua/jnode"
	"github.com/xyproto/datablock"
	"github.com/yuin/gopher-lua"
	"github.com/xyproto/jpath"
)

// For dealing with JSON documents and strings

const (
	// Identifier for the JFile class in Lua
	lJFileClass = "JFile"
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
// Returns a string value or an empty string.
func jfileGetString(L *lua.LState) int {
	jfile := checkJFile(L) // arg 1
	jsonpath := L.ToString(2)
	if jsonpath == "" {
		L.ArgError(2, "JSON path expected")
	}
	val, err := jfile.GetString(jsonpath)
	if err != nil {
		log.Error(err)
		val = ""
	}
	L.Push(lua.LString(val))
	return 1 // number of results
}

// Takes a JFile and a JSON path.
// Returns a JNode or nil.
func jfileGetNode(L *lua.LState) int {
	jfile := checkJFile(L) // arg 1
	jsonpath := L.ToString(2)
	if jsonpath == "" {
		L.ArgError(2, "JSON path expected")
	}
	node, err := jfile.GetNode(jsonpath)
	if err != nil {
		L.Push(lua.LNil)
		return 1 // number of results
	}

	// Return the JNode
	ud := L.NewUserData()
	ud.Value = node
	L.SetMetatable(ud, L.GetTypeMetatable(jnode.Class))
	L.Push(ud)
	return 1 // number of results
}

// Takes a JFile and a JSON path.
// Returns a value or nil.
func jfileGet(L *lua.LState) int {
	jfile := checkJFile(L) // arg 1
	jsonpath := L.ToString(2)
	if jsonpath == "" {
		L.ArgError(2, "JSON path expected")
	}

	// Will handle nil nodes below, so the error value can be ignored
	node, _ := jfile.GetNode(jsonpath)

	// Convert the JSON node to a Lua value, if possible
	var retval lua.LValue
	if node == jpath.NilNode {
		retval = lua.LNil
	} else if _, ok := node.CheckMap(); ok {
		// Return the JNode instead of converting the map
		log.Info("Returning a JSON node instead of a Lua map")
		ud := L.NewUserData()
		ud.Value = node
		L.SetMetatable(ud, L.GetTypeMetatable(jnode.Class))
		retval = ud
		//buf.WriteString(fmt.Sprintf("Map with %d elements.", len(m)))
	} else if _, ok := node.CheckList(); ok {
		log.Info("Returning a JSON node instead of a Lua map")
		// Return the JNode instead of converting the list
		ud := L.NewUserData()
		ud.Value = node
		L.SetMetatable(ud, L.GetTypeMetatable(jnode.Class))
		retval = ud
		//buf.WriteString(fmt.Sprintf("List with %d elements.", len(l)))
	} else if s, ok := node.CheckString(); ok {
		retval = lua.LString(s)
	} else if s, ok := node.CheckInt(); ok {
		retval = lua.LNumber(s)
	} else if b, ok := node.CheckBool(); ok {
		retval = lua.LBool(b)
	} else if i, ok := node.CheckInt64(); ok {
		retval = lua.LNumber(i)
	} else if u, ok := node.CheckUint64(); ok {
		retval = lua.LNumber(u)
	} else if f, ok := node.CheckFloat64(); ok {
		retval = lua.LNumber(f)
	} else {
		log.Error("Unknown JSON node type")
		return 0
	}
	// Return the LValue
	L.Push(retval)
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

// Create a new JSON file
func constructJFile(L *lua.LState, filename string, fperm os.FileMode, fs *datablock.FileStat) (*lua.LUserData, error) {
	fullFilename := filename
	// Check if the file exists
	if !fs.Exists(fullFilename) {
		// Create an empty JSON document/file
		if err := ioutil.WriteFile(fullFilename, []byte("[]\n"), fperm); err != nil {
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
	"getstring":  jfileGetString,
	"getnode":    jfileGetNode,
	"get":        jfileGet,
	"set":        jfileSet,
	"delkey":     jfileDelKey,
	"string":     jfileJSON, // undocumented
}

// LoadJFile makes functions related to building a library of Lua code available
func (ac *Config) LoadJFile(L *lua.LState, scriptdir string) {

	// Register the JFile class and the methods that belongs with it.
	mt := L.NewTypeMetatable(lJFileClass)
	mt.RawSetH(lua.LString("__index"), mt)
	L.SetFuncs(mt, jfileMethods)

	// The constructor for new Libraries takes only an optional id
	L.SetGlobal("JFile", L.NewFunction(func(L *lua.LState) int {
		// Get the filename and schema
		filename := L.ToString(1)

		// Construct a new JFile
		userdata, err := constructJFile(L, filepath.Join(scriptdir, filename), ac.defaultPermissions, ac.fs)
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
