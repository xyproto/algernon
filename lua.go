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

	exportUserstate(w, req, L, userstate)
	if err := L.DoFile(filename); err != nil {
		log.Println(err)
	}
}
