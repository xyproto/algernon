package main

import (
	"github.com/xyproto/pinterface"
	"github.com/yuin/gopher-lua"
	"net/http"
	"path/filepath"
)

// Expose functions for serving other files to Lua (and caching)
func exportServeFile(w http.ResponseWriter, req *http.Request, L *lua.LState, filename string, perm pinterface.IPermissions, luapool *lStatePool, cache *FileCache) {

	// Serve a file in the scriptdir
	L.SetGlobal("serve", L.NewFunction(func(L *lua.LState) int {
		scriptdir := filepath.Dir(filename)
		serveFilename := filepath.Join(scriptdir, L.ToString(1))
		if exists(serveFilename) {
			filePage(w, req, serveFilename, perm, luapool, cache)
		}
		return 0
	}))

}
