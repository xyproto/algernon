// Package profile provides Lua functions for profiling
package profile

import (
	"net/http"
	_ "net/http/pprof"

	log "github.com/sirupsen/logrus"
	"github.com/yuin/gopher-lua"
)

// Load makes helper functions for profilaing available
func Load(L *lua.LState) {

	// Lua function for converting a table to JSON (string or int)
	enableProf := L.NewFunction(func(L *lua.LState) int {
		log.Info("Listening to port 6060. Try: go tool pprof http://localhost:6060/debug/pprof/profile")
		go http.ListenAndServe("0.0.0.0:6060", nil)
		return 0 // number of results
	})

	// Expose functions
	L.SetGlobal("debugProfile", enableProf)
}
