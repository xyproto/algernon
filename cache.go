package main

import (
	"github.com/xyproto/datablock"
	"github.com/yuin/gopher-lua"
	"net/http"
)

// Helper function for sending file data (that might be cached) to a HTTP client
func dataToClient(w http.ResponseWriter, req *http.Request, filename string, data []byte) {
	datablock.NewDataBlock(data, true).ToClient(w, req, filename, clientCanGzip(req), gzipThreshold)
}

// Export functions related to the cache. cache can be nil.
func exportCacheFunctions(L *lua.LState, cache *datablock.FileCache) {

	const disabledMessage = "Caching is disabled"
	const clearedMessage = "Cache cleared"

	luaCacheStatsFunc := L.NewFunction(func(L *lua.LState) int {
		if cache == nil {
			L.Push(lua.LString(disabledMessage))
			return 1 // number of results
		}
		info := cache.Stats()
		// Return the string, but drop the final newline
		L.Push(lua.LString(info[:len(info)-1]))
		return 1 // number of results
	})

	// Return information about the cache use
	L.SetGlobal("CacheInfo", luaCacheStatsFunc)
	L.SetGlobal("CacheStats", luaCacheStatsFunc) // undocumented alias

	// Clear the cache
	L.SetGlobal("ClearCache", L.NewFunction(func(L *lua.LState) int {
		if cache == nil {
			L.Push(lua.LString(disabledMessage))
			return 1 // number of results
		}
		cache.Clear()
		L.Push(lua.LString(clearedMessage))
		return 1 // number of results
	}))

	// Try to load a file into the file cache, if it isn't already there
	L.SetGlobal("preload", L.NewFunction(func(L *lua.LState) int {
		filename := L.ToString(1)
		if cache == nil {
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		// Don't read from disk if already in cache, hence "true"
		if _, err := cache.Read(filename, true); err != nil {
			L.Push(lua.LBool(false))
			return 1 // number of results
		}
		L.Push(lua.LBool(true)) // success
		return 1                // number of results
	}))

}
