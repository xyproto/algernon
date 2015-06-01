package main

import (
	"github.com/xyproto/pinterface"
	"github.com/yuin/gopher-lua"
	"net/http"
	"path/filepath"
)

// Expose functions for serving other files to Lua
func exportServeFile(w http.ResponseWriter, req *http.Request, L *lua.LState, filename string, perm pinterface.IPermissions, luapool *lStatePool) {

	// Serve a file in the scriptdir
	L.SetGlobal("serve", L.NewFunction(func(L *lua.LState) int {
		scriptdir := filepath.Dir(filename)
		serveFilename := filepath.Join(scriptdir, L.ToString(1))
		if exists(serveFilename) {
			filePage(w, req, serveFilename, perm, luapool)
		}
		return 0
	}))

}
