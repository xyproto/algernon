package main

import (
	"html/template"
	"net/http"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/xyproto/datablock"
	"github.com/xyproto/pinterface"
)

// Functions for concurrent use by rendering.go and handlers.go

// Read in a Lua file and return a template.FuncMap (or an error)
func lua2funcMap(w http.ResponseWriter, req *http.Request, filename, luafilename, ext string, perm pinterface.IPermissions, luapool *lStatePool, cache *datablock.FileCache, pongomutex *sync.RWMutex, errChan chan error, funcMapChan chan template.FuncMap) {

	// Make functions from the given Lua data available
	funcs := make(template.FuncMap)

	// Try reading data.lua, if possible
	luablock, err := cache.Read(luafilename, shouldCache(ext))
	if err != nil {
		// Could not find and/or read data.lua
		luablock = datablock.EmptyDataBlock

		// This is not an error tha needs to be given to the user
	}

	// luablock can be empty if there was an error or if the file was empty
	if luablock.HasData() {
		// There was Lua code available. Now make the functions and
		// variables available for the template.
		funcs, err = luaFunctionMap(w, req, luablock.MustData(), luafilename, perm, luapool, cache, pongomutex)
		if err != nil {
			funcMapChan <- funcs
			errChan <- err
			return
		}
		if debugMode && verboseMode {
			s := "These functions from " + luafilename
			s += " are useable for " + filename + ": "
			// Create a comma separated list of the available functions
			for key := range funcs {
				s += key + ", "
			}
			// Remove the final comma
			if strings.HasSuffix(s, ", ") {
				s = s[:len(s)-2]
			}
			// Output the message
			log.Info(s)
		}
	}
	funcMapChan <- funcs
	errChan <- err
}
