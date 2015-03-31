package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/yuin/gopher-lua"
)

func exportBasicSystem(L *lua.LState) {

	// Return the version string
	L.SetGlobal("version", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(version_string))
		return 1 // number of results
	}))

	// Log text with the "Info" log type
	L.SetGlobal("log", L.NewFunction(func(L *lua.LState) int {
		buf := arguments2buffer(L)
		// Log the combined text
		log.Info(buf.String())
		return 0 // number of results
	}))

	// Log text with the "Warn" log type
	L.SetGlobal("warn", L.NewFunction(func(L *lua.LState) int {
		buf := arguments2buffer(L)
		// Log the combined text
		log.Warn(buf.String())
		return 0 // number of results
	}))

}

// Make functions related to HTTP requests and responses available to Lua scripts.
// Filename can be an empty string.
func exportBasicWeb(w http.ResponseWriter, req *http.Request, L *lua.LState, filename string) {

	// Print text to the webpage that is being served
	L.SetGlobal("print", L.NewFunction(func(L *lua.LState) int {
		var buf bytes.Buffer
		top := L.GetTop()
		for i := 1; i <= top; i++ {
			buf.WriteString(L.Get(i).String())
			if i != top {
				buf.WriteString("\t")
			}
		}
		buf.WriteString("\n")
		// Write the combined text to the http.ResponseWriter
		w.Write(buf.Bytes())
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
			// Also include a separator and a filename
			fn := L.ToString(1)
			scriptdir += sep + fn
		}
		if sep != "/" {
			// For operating systems that use backslash
			scriptdir = strings.Replace(scriptdir, "\\", "/", -1)
		}
		L.Push(lua.LString(scriptdir))
		return 1 // number of results
	}))

	// Get the full filename of a given file that is in the directory
	// where the server is running (root directory for the server).
	// If no filename is given, the directory where the server is
	// currently running is returned.
	L.SetGlobal("serverdir", L.NewFunction(func(L *lua.LState) int {
		serverdir, err := os.Getwd()
		if err != nil {
			// Could not retrieve a directory
			serverdir = ""
		} else if L.GetTop() == 1 {
			// Also include a separator and a filename
			fn := L.ToString(1)
			serverdir += sep + fn
		}
		if sep != "/" {
			// For operating systems that use backslash
			serverdir = strings.Replace(serverdir, "\\", "/", -1)
		}
		L.Push(lua.LString(serverdir))
		return 1 // number of results
	}))
}
