package main

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"github.com/yuin/gopher-lua"
)

func exportJSONFunctions(L *lua.LState) {

	// Convert a table to JSON
	L.SetGlobal("JSON", L.NewFunction(func(L *lua.LState) int {
		table := L.ToTable(1)
		mapinterface, multiple := table2map(table)
		if multiple {
			log.Warn("JSON: Ignoring table values with different types")
		}
		b, err := json.Marshal(mapinterface)
		if err != nil {
			log.Error(err)
		}
		L.Push(lua.LString(string(b)))
		return 1 // number of results
	}))

}
