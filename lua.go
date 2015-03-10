package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/xyproto/permissions2"
	"github.com/yuin/gopher-lua"
)

// Functions that are available from Lua
func Double(L *lua.LState) int {
	lv := L.ToInt(1)            /* get argument */
	L.Push(lua.LNumber(lv * 2)) /* push result */
	return 1                    /* number of results */
}

func exportFunctions(L *lua.LState) {
	L.SetGlobal("double", L.NewFunction(Double))
}

func runLua(w http.ResponseWriter, req *http.Request, filename string, userstate *permissions.UserState) {
	L := lua.NewState()
	defer L.Close()
	exportFunctions(L)
	L.SetGlobal("print", L.NewFunction(func(L *lua.LState) int {
		lv := L.ToString(1)
		fmt.Fprintln(w, lv)
		return 0 // number of results
	}))
	L.SetGlobal("setContentType", L.NewFunction(func(L *lua.LState) int {
		lv := L.ToString(1)
		w.Header().Add("Content-Type", lv)
		return 0 // number of results
	}))
	L.SetGlobal("getUsername", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(userstate.Username(req)))
		return 1 // number of results
	}))
	if err := L.DoFile(filename); err != nil {
		log.Println(err)
	}
}
