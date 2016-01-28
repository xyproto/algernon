package main

import (
	"bytes"
	"encoding/json"
	"net/http" // For sending JSON requests
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/xyproto/jpath"
	"github.com/yuin/gopher-lua"
)

// For dealing with JSON documents and strings

const (
	// Identifier for the JNode class in Lua
	lJNodeClass = "JNode"

	// Prefix when indenting JSON
	indentPrefix = ""
)

// Get the first argument, "self", and cast it from userdata to a library (which is really a hash map).
func checkJNode(L *lua.LState) *jpath.Node {
	ud := L.CheckUserData(1)
	if jnode, ok := ud.Value.(*jpath.Node); ok {
		return jnode
	}
	L.ArgError(1, "JSON node expected")
	return nil
}

// Takes a JNode, a JSON path (optional) and JSON data.
// Stores the JSON data. Returns true if successful.
func jnodeAdd(L *lua.LState) int {
	jnode := checkJNode(L) // arg 1
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
	err := jnode.AddJSON(jsonpath, []byte(jsondata))
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

// Takes a JNode and a JSON path.
// Returns a value or an empty string.
func jnodeGetNode(L *lua.LState) int {
	jnode := checkJNode(L) // arg 1
	jsonpath := L.ToString(2)
	if jsonpath == "" {
		L.ArgError(2, "JSON path expected")
	}
	node := jnode.GetNode(jsonpath)
	ud := L.NewUserData()
	ud.Value = node
	L.SetMetatable(ud, L.GetTypeMetatable(lJNodeClass))
	L.Push(ud)
	return 1 // number of results
}

// Takes a JNode and a JSON path.
// Returns a value or an empty string.
func jnodeGetString(L *lua.LState) int {
	jnode := checkJNode(L) // arg 1
	jsonpath := L.ToString(2)
	if jsonpath == "" {
		L.ArgError(2, "JSON path expected")
	}
	node := jnode.GetNode(jsonpath)
	L.Push(lua.LString(node.String()))
	return 1 // number of results
}

// Take a JNode, a JSON path and a string.
// Returns nothing
func jnodeSet(L *lua.LState) int {
	jnode := checkJNode(L) // arg 1
	jsonpath := L.ToString(2)
	if jsonpath == "" {
		L.ArgError(2, "JSON path expected")
	}
	sval := L.ToString(3)
	if sval == "" {
		L.ArgError(3, "String value expected")
	}
	jnode.Set(jsonpath, sval)
	return 0 // number of results
}

// Take a JNode and a JSON path.
// Remove a key from a map. Return true if successful.
func jnodeDelKey(L *lua.LState) int {
	jnode := checkJNode(L) // arg 1
	jsonpath := L.ToString(2)
	if jsonpath == "" {
		L.ArgError(2, "JSON path expected")
	}
	err := jnode.DelKey(jsonpath)
	if err != nil {
		log.Error(err)
	}
	L.Push(lua.LBool(nil == err))
	return 1 // number of results
}

// Given a JNode, return the JSON document.
// May return an empty string.
func jnodeJSON(L *lua.LState) int {
	jnode := checkJNode(L) // arg 1

	data, err := jnode.PrettyJSON()
	retval := ""
	if err == nil { // ok
		retval = string(data)
	}
	L.Push(lua.LString(retval))
	return 1 // number of results
}

// Send JSON to host. First argument: URL
// Second argument (optional) Auth token.
// Returns a string that starts with FAIL if it fails.
// Returns the HTTP status code if it works out.
func jnodeSendToURL(L *lua.LState) int {
	jnode := checkJNode(L) // arg 1

	posturl := L.ToString(2)
	if posturl == "" {
		L.ArgError(2, "URL for sending a JSON POST requests to expected")
	}

	if !strings.HasPrefix(posturl, "http") {
		L.ArgError(2, "URL must start with http or https")
	}

	top := L.GetTop()
	authtoken := ""
	if top == 3 {
		// Optional
		authtoken = L.ToString(3)
	}

	// Render JSON
	jsonData, err := jnode.JSON()
	if err != nil {
		L.Push(lua.LString("FAIL: " + err.Error()))
		return 1 // number of results
	}

	// Set up request
	client := &http.Client{}
	r, _ := http.NewRequest("POST", posturl, bytes.NewReader(jsonData))
	if authtoken != "" {
		r.Header.Add("Authorization", "auth_token=\""+authtoken+"\"")
	}
	r.Header.Add("Content-Type", "application/json")

	// Send request and return result
	resp, err := client.Do(r)
	if err != nil || resp.Status != "200 OK" {
		L.Push(lua.LString("FAIL: " + resp.Status))
		return 1 // number of results
	}

	L.Push(lua.LString(resp.Status))
	return 1 // number of results
}

// String representation
//func jnodeToString(L *lua.LState) int {
//	L.Push(lua.LString("JSON node"))
//	return 1 // number of results
//}

// Create a new JSON node
func constructJNode(L *lua.LState) (*lua.LUserData, error) {
	// Create a new JNode
	jnode := jpath.NewNode()
	// Create a new userdata struct
	ud := L.NewUserData()
	ud.Value = jnode
	L.SetMetatable(ud, L.GetTypeMetatable(lJNodeClass))
	return ud, nil
}

// The hash map methods that are to be registered
var jnodeMethods = map[string]lua.LGFunction{
	"__tostring": jnodeJSON,
	"add":        jnodeAdd,
	"getstring":  jnodeGetString,
	"get":        jnodeGetNode,
	"set":        jnodeSet,
	"delkey":     jnodeDelKey,
	"string":     jnodeJSON, // undocumented
	"send":       jnodeSendToURL,
}

// Make functions related JSON nodes
func exportJNode(L *lua.LState) {

	// Register the JNode class and the methods that belongs with it.
	mt := L.NewTypeMetatable(lJNodeClass)
	mt.RawSetH(lua.LString("__index"), mt)
	L.SetFuncs(mt, jnodeMethods)

	// The constructor for new Libraries takes only an optional id
	L.SetGlobal("JNode", L.NewFunction(func(L *lua.LState) int {
		// Construct a new JNode
		userdata, err := constructJNode(L)
		if err != nil {
			log.Error(err)
			L.Push(lua.LString(err.Error()))
			return 1 // Number of returned values
		}

		// Return the Lua JNode object
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
