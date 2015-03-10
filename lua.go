package main

import (
	"fmt"
	"log"
	"net/http"

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

func runLua(w http.ResponseWriter, req *http.Request, filename string, userstate *permissions.UserState) {
	L := lua.NewState()
	defer L.Close()
	L.SetGlobal("print", L.NewFunction(func(L *lua.LState) int {
		a := L.ToString(1)
		b := L.ToString(2)
		fmt.Fprintln(w, a, b)
		return 0 // number of results
	}))
	L.SetGlobal("setContentType", L.NewFunction(func(L *lua.LState) int {
		lv := L.ToString(1)
		w.Header().Add("Content-Type", lv)
		return 0 // number of results
	}))
	exportUserstate(w, req, L, userstate)
	if err := L.DoFile(filename); err != nil {
		log.Println(err)
	}
}
