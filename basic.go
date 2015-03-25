package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"

	"github.com/yuin/gopher-lua"
)

// Make functions related to HTTP requests and responses available to Lua scripts.
// Filename can be an empty string.
func exportBasic(w http.ResponseWriter, req *http.Request, L *lua.LState, filename string) {

	// Print text to the webpage that is being served
	L.SetGlobal("print", L.NewFunction(func(L *lua.LState) int {
		top := L.GetTop()
		for i := 1; i <= top; i++ {
			fmt.Fprint(w, L.Get(i).String())
			if i != top {
				fmt.Fprint(w, "\t")
			}
		}
		fmt.Fprint(w, "\n")
		return 0 // number of results
	}))

	// Set the Content-Type for the page
	setContentType := L.NewFunction(func(L *lua.LState) int {
		lv := L.ToString(1)
		w.Header().Add("Content-Type", lv)
		return 0 // number of results
	})
	// Register the function
	L.SetGlobal("content", setContentType)
	// Register an alias, for backwards compatibility
	L.SetGlobal("setContentType", setContentType)

	// Return the current URL Path
	L.SetGlobal("urlpath", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(req.URL.Path))
		return 1 // number of results
	}))

	// Return the current HTTP method (GET, POST etc)
	L.SetGlobal("method", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(req.Method))
		return 1 // number of results
	}))

	// Return the HTTP header in the request, for a given key/string
	L.SetGlobal("header", L.NewFunction(func(L *lua.LState) int {
		key := L.ToString(1)
		value := req.Header.Get(key)
		L.Push(lua.LString(value))
		return 1 // number of results
	}))

	// Return the HTTP body in the request
	L.SetGlobal("body", L.NewFunction(func(L *lua.LState) int {
		body, err := ioutil.ReadAll(req.Body)
		var result lua.LString
		if err != nil {
			result = lua.LString("")
		} else {
			result = lua.LString(string(body))
		}
		L.Push(result)
		return 1 // number of results
	}))

	// Print markdown text as html
	L.SetGlobal("mprint", L.NewFunction(func(L *lua.LState) int {
		top := L.GetTop()
		for i := 1; i <= top; i++ {
			fmt.Fprint(w, markdown(L.Get(i).String()))
		}
		fmt.Fprint(w, markdown("\n"))
		return 0 // number of results
	}))

	// Return the version string
	L.SetGlobal("version", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(version_string))
		return 1 // number of results
	}))

	// Set the HTTP status code (must come before print)
	L.SetGlobal("status", L.NewFunction(func(L *lua.LState) int {
		code := int(L.ToNumber(1))
		w.WriteHeader(code)
		return 0 // number of results
	}))

	// Print a message and set the HTTP status code
	L.SetGlobal("error", L.NewFunction(func(L *lua.LState) int {
		message := L.ToString(1)
		code := int(L.ToNumber(2))
		w.WriteHeader(code)
		fmt.Fprint(w, message)
		return 0 // number of results
	}))

	// Get the full filename of a given file that is in the directory
	// of the script that is about to be run.
	// If no filename is given, the directory where the script lies
	// is returned.
	L.SetGlobal("scriptdir", L.NewFunction(func(L *lua.LState) int {
		scriptdir := path.Dir(filename)
		top := L.GetTop()
		if top == 1 {
			fn := L.ToString(1)
			L.Push(lua.LString(scriptdir + sep + fn))
		} else {
			L.Push(lua.LString(scriptdir))
		}
		return 1 // number of results
	}))

	// Get the full filename of a given file that is in the directory
	// where the server is running (root directory for the server).
	// If no filename is given, the directory where the server is
	// currently running is returned.
	L.SetGlobal("serverdir", L.NewFunction(func(L *lua.LState) int {
		var result string
		serverdir, err := os.Getwd()
		if err != nil {
			result = ""
		} else if L.GetTop() == 1 {
			fn := L.ToString(1)
			result = serverdir + sep + fn
		} else {
			result = serverdir
		}
		L.Push(lua.LString(result))
		return 1 // number of results
	}))
}
