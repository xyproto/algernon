package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/xyproto/pinterface"
	"github.com/yuin/gopher-lua"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
)

// Expose functions for serving other files to Lua
func exportServeFile(w http.ResponseWriter, req *http.Request, L *lua.LState, filename string, perm pinterface.IPermissions, luapool *lStatePool, cache *FileCache, pongomutex *sync.RWMutex) {

	// Serve a file in the scriptdir
	L.SetGlobal("serve", L.NewFunction(func(L *lua.LState) int {
		scriptdir := filepath.Dir(filename)
		serveFilename := filepath.Join(scriptdir, L.ToString(1))
		dataFilename := filepath.Join(scriptdir, defaultLuaDataFilename)
		if L.GetTop() >= 2 {
			// Optional argument for using a different file than "data.lua"
			dataFilename = filepath.Join(scriptdir, L.ToString(2))
		}
		if !fs.exists(serveFilename) {
			log.Error("Could not serve " + serveFilename + ". File not found.")
			return 0 // Number of results
		}
		if fs.isDir(serveFilename) {
			log.Error("Could not serve " + serveFilename + ". Not a file.")
			return 0 // Number of results
		}
		filePage(w, req, serveFilename, dataFilename, perm, luapool, cache, pongomutex)
		return 0 // Number of results
	}))

	// Get the rendered contents of a file in the scriptdir. Discards HTTP headers.
	L.SetGlobal("render", L.NewFunction(func(L *lua.LState) int {
		scriptdir := filepath.Dir(filename)
		serveFilename := filepath.Join(scriptdir, L.ToString(1))
		dataFilename := filepath.Join(scriptdir, defaultLuaDataFilename)
		if L.GetTop() >= 2 {
			// Optional argument for using a different file than "data.lua"
			dataFilename = filepath.Join(scriptdir, L.ToString(2))
		}
		if !fs.exists(serveFilename) {
			log.Error("Could not render " + serveFilename + ". File not found.")
			return 0 // Number of results
		}
		if fs.isDir(serveFilename) {
			log.Error("Could not render " + serveFilename + ". Not a file.")
			return 0 // Number of results
		}

		// Render the filename to a httptest.Recorder
		recorder := httptest.NewRecorder()
		filePage(recorder, req, serveFilename, dataFilename, perm, luapool, cache, pongomutex)

		// Return the recorder as a string
		L.Push(lua.LString(recorderToString(recorder)))
		return 1 // Number of results
	}))

}
