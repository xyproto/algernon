package engine

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/russross/blackfriday/v2"
	log "github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/lua/convert"
	"github.com/xyproto/algernon/utils"
	"github.com/xyproto/gopher-lua"
)

// FutureStatus is useful when redirecting in combination with writing to a
// buffer before writing to a client. May contain more fields in the future.
type FutureStatus struct {
	code int // Buffered HTTP status code
}

// LoadBasicSystemFunctions loads functions related to logging, markdown and the
// current server directory into the given Lua state
func (ac *Config) LoadBasicSystemFunctions(L *lua.LState) {

	// Return the version string
	L.SetGlobal("version", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(ac.versionString))
		return 1 // number of results
	}))

	// Log text with the "Info" log type
	L.SetGlobal("log", L.NewFunction(func(L *lua.LState) int {
		buf := convert.Arguments2buffer(L, false)
		// Log the combined text
		log.Info(buf.String())
		return 0 // number of results
	}))

	// Log text with the "Warn" log type
	L.SetGlobal("warn", L.NewFunction(func(L *lua.LState) int {
		buf := convert.Arguments2buffer(L, false)
		// Log the combined text
		log.Warn(buf.String())
		return 0 // number of results
	}))

	// Log text with the "Error" log type
	L.SetGlobal("err", L.NewFunction(func(L *lua.LState) int {
		buf := convert.Arguments2buffer(L, false)
		// Log the combined text
		log.Error(buf.String())
		return 0 // number of results
	}))

	// Sleep for the given number of seconds (can be a float)
	L.SetGlobal("sleep", L.NewFunction(func(L *lua.LState) int {
		// Extract the correct number of nanoseconds
		duration := time.Duration(float64(L.ToNumber(1)) * 1000000000.0)
		// Wait and block the current thread of execution.
		time.Sleep(duration)
		return 0
	}))

	// Return the current unixtime, with an attempt at nanosecond resolution
	L.SetGlobal("unixnano", L.NewFunction(func(L *lua.LState) int {
		// Extract the correct number of nanoseconds
		L.Push(lua.LNumber(time.Now().UnixNano()))
		return 1 // number of results
	}))

	// Convert Markdown to HTML
	L.SetGlobal("markdown", L.NewFunction(func(L *lua.LState) int {
		// Retrieve all the function arguments as a bytes.Buffer
		buf := convert.Arguments2buffer(L, true)
		// Convert the buffer to markdown and output the translated string
		html := strings.TrimSpace(string(blackfriday.Run(buf.Bytes())))
		L.Push(lua.LString(html))
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
			serverdir = filepath.Join(serverdir, fn)
		}
		L.Push(lua.LString(serverdir))
		return 1 // number of results
	}))

}

// LoadBasicWeb loads functions related to handling requests, outputting data to
// the browser, setting headers, pretty printing and dealing with the directory
// where files are being served, into the given Lua state.
func (ac *Config) LoadBasicWeb(w http.ResponseWriter, req *http.Request, L *lua.LState, filename string, flushFunc func(), httpStatus *FutureStatus) {

	// Print text to the web page that is being served. Add a newline.
	L.SetGlobal("print", L.NewFunction(func(L *lua.LState) int {
		var buf bytes.Buffer
		top := L.GetTop()
		for i := 1; i <= top; i++ {
			buf.WriteString(L.Get(i).String())
			if i != top {
				buf.WriteString("\t")
			}
		}
		// Final newline
		buf.WriteString("\n")

		// Write the combined text to the http.ResponseWriter
		buf.WriteTo(w)

		return 0 // number of results
	}))

	// Pretty print text to the web page that is being served. Add a newline.
	L.SetGlobal("pprint", L.NewFunction(func(L *lua.LState) int {
		var buf bytes.Buffer
		top := L.GetTop()
		for i := 1; i <= top; i++ {
			convert.PprintToWriter(&buf, L.Get(i))
			if i != top {
				buf.WriteString("\t")
			}
		}

		// Final newline
		buf.WriteString("\n")

		// Write the combined text to the http.ResponseWriter
		buf.WriteTo(w)

		return 0 // number of results
	}))

	// Pretty print to string
	L.SetGlobal("ppstr", L.NewFunction(func(L *lua.LState) int {
		var buf bytes.Buffer
		top := L.GetTop()
		for i := 1; i <= top; i++ {
			convert.PprintToWriter(&buf, L.Get(i))
			if i != top {
				buf.WriteString("\t")
			}
		}

		// Return the string
		L.Push(lua.LString(buf.String()))

		return 1 // number of results
	}))

	// Flush the ResponseWriter.
	// Needed in debug mode, where ResponseWriter is buffered.
	L.SetGlobal("flush", L.NewFunction(func(L *lua.LState) int {
		if flushFunc != nil {
			flushFunc()
		}
		return 0 // number of results
	}))

	// Set the Content-Type for the page
	L.SetGlobal("content", L.NewFunction(func(L *lua.LState) int {
		lv := L.ToString(1)
		w.Header().Add("Content-Type", lv)
		return 0 // number of results
	}))

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

	// Return the HTTP headers as a table
	L.SetGlobal("headers", L.NewFunction(func(L *lua.LState) int {
		luaTable := L.NewTable()
		for key := range req.Header {
			L.RawSet(luaTable, lua.LString(key), lua.LString(req.Header.Get(key)))
		}
		if req.Host != "" {
			L.RawSet(luaTable, lua.LString("Host"), lua.LString(req.Host))
		}
		L.Push(luaTable)
		return 1 // number of results
	}))

	// Return the HTTP header in the request, for a given key/string
	L.SetGlobal("header", L.NewFunction(func(L *lua.LState) int {
		key := L.ToString(1)
		value := req.Header.Get(key)
		L.Push(lua.LString(value))
		return 1 // number of results
	}))

	// Set the HTTP header in the request, for a given key and value
	L.SetGlobal("setheader", L.NewFunction(func(L *lua.LState) int {
		key := L.ToString(1)
		value := L.ToString(2)
		w.Header().Set(key, value)
		return 0 // number of results
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
		if httpStatus != nil {
			httpStatus.code = code
		}
		w.WriteHeader(code)
		return 0 // number of results
	}))

	// Throw an error/exception in Lua
	L.SetGlobal("throw", L.GetGlobal("error"))

	// Set a HTTP status code and print a message (optional)
	L.SetGlobal("error", L.NewFunction(func(L *lua.LState) int {
		code := int(L.ToNumber(1))
		if httpStatus != nil {
			httpStatus.code = code
		}
		w.WriteHeader(code)
		if L.GetTop() == 2 {
			message := L.ToString(2)
			fmt.Fprint(w, message)
		}
		return 0 // number of results
	}))

	// Get the full filename of a given file that is in the directory
	// of the script that is about to be run. If no filename is given,
	// the directory of the script is returned.
	L.SetGlobal("scriptdir", L.NewFunction(func(L *lua.LState) int {
		scriptpath, err := filepath.Abs(filename)
		if err != nil {
			scriptpath = filename
		}
		scriptdir := filepath.Dir(scriptpath)
		scriptpath = scriptdir
		top := L.GetTop()
		if top == 1 {
			// Also include a separator and a filename
			fn := L.ToString(1)
			scriptpath = filepath.Join(scriptdir, fn)
		}
		// Now have the correct absolute scriptpath
		L.Push(lua.LString(scriptpath))
		return 1 // number of results
	}))

	// Given a filename, return the URL path
	L.SetGlobal("file2url", L.NewFunction(func(L *lua.LState) int {
		fn := L.ToString(1)
		targetpath := strings.TrimPrefix(filepath.Join(filepath.Dir(filename), fn), ac.serverDirOrFilename)
		if utils.Pathsep != "/" {
			// For operating systems that use another path separator for files than for URLs
			targetpath = strings.Replace(targetpath, utils.Pathsep, "/", utils.EveryInstance)
		}
		withSlashPrefix := path.Join("/", targetpath)
		L.Push(lua.LString(withSlashPrefix))
		return 1 // number of results
	}))

	// Retrieve a table with keys and values from the form in the request
	L.SetGlobal("formdata", L.NewFunction(func(L *lua.LState) int {
		// Place the form data in a map
		m := make(map[string]string)
		req.ParseForm()
		for key, values := range req.Form {
			m[key] = values[0]
		}
		// Convert the map to a table and return it
		L.Push(convert.Map2table(L, m))
		return 1 // number of results
	}))

	// Retrieve a table with keys and values from the URL in the request
	L.SetGlobal("urldata", L.NewFunction(func(L *lua.LState) int {

		var valueMap url.Values
		var err error

		if L.GetTop() == 1 {
			// If given an argument
			rawurl := L.ToString(1)
			valueMap, err = url.ParseQuery(rawurl)
			// Log error as warning if there are issues.
			// An empty Value map will then be used.
			if err != nil {
				log.Error(err)
				// return 0
			}
		} else {
			// If not given an argument
			valueMap = req.URL.Query() // map[string][]string
		}

		// Place the Value data in a map, using the first values
		// if there are many values for a given key.
		m := make(map[string]string)
		for key, values := range valueMap {
			m[key] = values[0]
		}
		// Convert the map to a table and return it
		L.Push(convert.Map2table(L, m))
		return 1 // number of results
	}))

	// Redirect a request (as found, by default)
	L.SetGlobal("redirect", L.NewFunction(func(L *lua.LState) int {
		newurl := L.ToString(1)
		httpStatusCode := http.StatusFound
		if L.GetTop() == 2 {
			httpStatusCode = int(L.ToNumber(2))
		}
		if httpStatus != nil {
			httpStatus.code = httpStatusCode
		}
		http.Redirect(w, req, newurl, httpStatusCode)
		return 0 // number of results
	}))

	// Permanently redirect a request, which is the same as redirect(url, 301)
	L.SetGlobal("permanent_redirect", L.NewFunction(func(L *lua.LState) int {
		newurl := L.ToString(1)
		httpStatusCode := http.StatusMovedPermanently
		if httpStatus != nil {
			httpStatus.code = httpStatusCode
		}
		http.Redirect(w, req, newurl, httpStatusCode)
		return 0 // number of results
	}))

	// Run the given Lua file (replacement for the built-in dofile, to look in the right directory)
	// Returns whatever the Lua file returns when it is being run.
	L.SetGlobal("dofile", L.NewFunction(func(L *lua.LState) int {
		givenFilename := L.ToString(1)
		luaFilename := filepath.Join(filepath.Dir(filename), givenFilename)
		if !ac.fs.Exists(luaFilename) {
			log.Error("Could not find:", luaFilename)
			return 0 // number of results
		}
		if err := L.DoFile(luaFilename); err != nil {
			log.Errorf("Error running %s: %s\n", luaFilename, err)
			return 0 // number of results
		}
		// Retrieve the returned value from the script
		retval := L.Get(-1)
		L.Pop(1)
		// Return the value returned from the script
		L.Push(retval)
		return 1 // number of results
	}))

}
