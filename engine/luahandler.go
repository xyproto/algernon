package engine

import (
	"net/http"
	"path/filepath"

	"github.com/didip/tollbooth"
	"github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/themes"
	lua "github.com/xyproto/gopher-lua"
)

// handleRegistryPrefix keys a handler function in a Lua state's registry
const handleRegistryPrefix = "algernon:handle:"

// LoadLuaHandlerFunctions makes functions related to handling HTTP requests
// available to Lua scripts.
//
// When registerRoutes is true, handle() registers the route on the mux. That
// wrapped handler borrows a state from ac.handlerPool at request time, looks
// up the handler function from the state's registry by path, and runs it.
// When registerRoutes is false, handle() only stores the function in the
// state's registry; this is the mode used while populating the pool.
func (ac *Config) LoadLuaHandlerFunctions(L *lua.LState, filename string, mux *http.ServeMux, addDomain bool, httpStatus *FutureStatus, theme string, registerRoutes bool) {
	L.SetGlobal("handle", L.NewFunction(func(L *lua.LState) int {
		handlePath := L.ToString(1)
		handleFunc := L.ToFunction(2)

		// Store the function in this state's registry, keyed by path,
		// so the request-time wrapper can fetch it from whichever pool
		// state it happens to borrow.
		L.G.Registry.RawSetString(handleRegistryPrefix+handlePath, handleFunc)

		if !registerRoutes {
			return 0 // number of results
		}

		wrappedHandleFunc := func(w http.ResponseWriter, req *http.Request) {
			if ac.handlerPool == nil {
				logrus.Error("Handler for " + handlePath + " called before the handler pool was built")
				return
			}
			poolL := ac.handlerPool.Get()
			defer ac.handlerPool.Put(poolL)

			fn := poolL.G.Registry.RawGetString(handleRegistryPrefix + handlePath)
			handlerFn, ok := fn.(*lua.LFunction)
			if !ok {
				logrus.Error("Handler for " + handlePath + " is missing from the pool state")
				return
			}

			ac.LoadCommonFunctions(w, req, filename, poolL, nil, httpStatus)
			poolL.Push(handlerFn)
			if err := poolL.PCall(0, lua.MultRet, nil); err != nil {
				// Non-fatal error
				logrus.Error("Handler for "+handlePath+" failed:", err)
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
			limiter.SetMessageContentType(htmlUTF8)
			mux.Handle(handlePath, tollbooth.LimitFuncHandler(limiter, wrappedHandleFunc))
		}

		return 0 // number of results
	}))

	L.SetGlobal("servedir", L.NewFunction(func(L *lua.LState) int {
		// servedir only has an effect during the first pass; subsequent
		// passes (pool build) must not re-register mux routes.
		if !registerRoutes {
			return 0 // number of results
		}
		handlePath := L.ToString(1) // serve as (ie. "/")
		rootdir := L.ToString(2)    // filesystem directory (ie. "./public")
		if handlePath == "" || rootdir == "" {
			logrus.Errorf("servedir needs an URL path to serve, ie. %q and a directory releative to %q, ie. %q", "/", filepath.Dir(filename), "./public")
			return 0
		}
		rootdir = filepath.Join(filepath.Dir(filename), rootdir)
		ac.RegisterHandlers(mux, handlePath, rootdir, addDomain)
		return 0 // number of results
	}))
}
