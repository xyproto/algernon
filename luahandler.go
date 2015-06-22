package main

import (
	"github.com/didip/tollbooth"
	"net/http"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/xyproto/pinterface"
	"github.com/yuin/gopher-lua"
)

// Make functions related to handling HTTP requests available to Lua scripts
func exportLuaHandlerFunctions(L *lua.LState, filename string, perm pinterface.IPermissions, luapool *lStatePool, cache *FileCache, mux *http.ServeMux) {

	L.SetGlobal("handle", L.NewFunction(func(L *lua.LState) int {
		handlePath := L.ToString(1)
		handleFunc := L.ToFunction(2)

		wrappedHandleFunc := func(w http.ResponseWriter, req *http.Request) {
			// Set up a new Lua state with the current http.ResponseWriter and *http.Request
			exportCommonFunctions(w, req, filename, perm, L, luapool, nil, cache)

			// Then run the given Lua function
			L.Push(handleFunc)
			if err := L.PCall(0, lua.MultRet, nil); err != nil {
				// Non-fatal error
				log.Error("Handler for "+handlePath+" failed:", err)
			}
		}

		// Handle requests differently depending on if rate limiting is enabled or not
		if disableRateLimiting {
			mux.HandleFunc(handlePath, wrappedHandleFunc)
		} else {
			limiter := tollbooth.NewLimiter(limitRequests, time.Second)
			limiter.MessageContentType = "text/html; charset=utf-8"
			limiter.Message = easyPage("Rate-limit exceeded", "<div style='color:red'>You have reached the maximum request limit.</div>")
			mux.Handle(handlePath, tollbooth.LimitFuncHandler(limiter, wrappedHandleFunc))
		}

		return 0 // number of results
	}))

	L.SetGlobal("servedir", L.NewFunction(func(L *lua.LState) int {
		handlePath := L.ToString(1) // serve as (ie. "/")
		rootdir := L.ToString(2) // filesystem directory (ie. "./public")
		rootdir = filepath.Join(filepath.Dir(filename), rootdir)
		handler := http.FileServer(http.Dir(rootdir))

		// Handle requests differently depending on if rate limiting is enabled or not
		if disableRateLimiting {
			mux.Handle(handlePath, handler)
		} else {
			limiter := tollbooth.NewLimiter(limitRequests, time.Second)
			limiter.MessageContentType = "text/html; charset=utf-8"
			limiter.Message = easyPage("Rate-limit exceeded", "<div style='color:red'>You have reached the maximum request limit.</div>")
			mux.Handle(handlePath, tollbooth.LimitHandler(limiter, handler))
		}

		return 0 // number of results
	}))

}
