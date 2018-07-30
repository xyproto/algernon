package engine

import (
	"net/http"
	"path/filepath"
	"sync"

	"github.com/didip/tollbooth"
	log "github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/themes"
	"github.com/yuin/gopher-lua"
)

// LoadLuaHandlerFunctions makes functions related to handling HTTP requests
// available to Lua scripts
func (ac *Config) LoadLuaHandlerFunctions(L *lua.LState, filename string, mux *http.ServeMux, addDomain bool, httpStatus *FutureStatus, theme string) {

	luahandlermutex := &sync.RWMutex{}

	L.SetGlobal("handle", L.NewFunction(func(L *lua.LState) int {

		handlePath := L.ToString(1)
		handleFunc := L.ToFunction(2)

		// TODO: Set up a channel and function for retrieving a lua "handleFunc" and running it,
		//       using the common luapool as needed

		wrappedHandleFunc := func(w http.ResponseWriter, req *http.Request) {

			// Set up a new Lua state with the current http.ResponseWriter and *http.Request
			luahandlermutex.Lock()
			ac.LoadCommonFunctions(w, req, filename, L, nil, httpStatus)
			luahandlermutex.Unlock()

			// Then run the given Lua function
			L.Push(handleFunc)
			if err := L.PCall(0, lua.MultRet, nil); err != nil {
				// Non-fatal error
				log.Error("Handler for "+handlePath+" failed:", err)
			}

			// Then exit after the first request, if specified
			if ac.quitAfterFirstRequest {
				go ac.quitSoon("Quit after first request", defaultSoonDuration)
			}
		}

		// Handle requests differently depending on if rate limiting is enabled or not
		if ac.disableRateLimiting {
			mux.HandleFunc(handlePath, wrappedHandleFunc)
		} else {
			limiter := tollbooth.NewLimiter(float64(ac.limitRequests), nil)
			limiter.SetMessage(themes.MessagePage("Rate-limit exceeded", "<div style='color:red'>You have reached the maximum request limit.</div>", theme))
			limiter.SetMessageContentType("text/html;charset=utf-8")
			mux.Handle(handlePath, tollbooth.LimitFuncHandler(limiter, wrappedHandleFunc))
		}

		return 0 // number of results
	}))

	L.SetGlobal("servedir", L.NewFunction(func(L *lua.LState) int {
		handlePath := L.ToString(1) // serve as (ie. "/")
		rootdir := L.ToString(2)    // filesystem directory (ie. "./public")
		rootdir = filepath.Join(filepath.Dir(filename), rootdir)

		ac.RegisterHandlers(mux, handlePath, rootdir, addDomain)

		return 0 // number of results
	}))

}
