package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/russross/blackfriday"
	"github.com/xyproto/permissions2"
	"github.com/yuin/gopher-lua"
)

func strings2table(L *lua.LState, sl []string) *lua.LTable {
	table := L.NewTable()
	for _, element := range sl {
		table.Append(lua.LString(element))
	}
	return table
}

func markdown(text string) string {
	return string(blackfriday.MarkdownCommon([]byte(text)))
}

func runLua(w http.ResponseWriter, req *http.Request, filename string, userstate *permissions.UserState, luapool *lStatePool) {
	L := luapool.Get()
	defer luapool.Put(L)

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
	// Register two aliases
	L.SetGlobal("content", setContentType)
	L.SetGlobal("SetContentType", setContentType)
	// And a third one for backwards compatibility
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
			if i != top {
				fmt.Fprint(w, markdown("\t"))
			}
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

	exportUserstate(w, req, L, userstate)
	if err := L.DoFile(filename); err != nil {
		log.Println(err)
	}
}
