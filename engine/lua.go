package engine

import (
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/cachemode"
	"github.com/xyproto/algernon/lua/codelib"
	"github.com/xyproto/algernon/lua/convert"
	"github.com/xyproto/algernon/lua/datastruct"
	"github.com/xyproto/algernon/lua/httpclient"
	"github.com/xyproto/algernon/lua/jnode"
	"github.com/xyproto/algernon/lua/mssql"
	"github.com/xyproto/algernon/lua/ollama"
	"github.com/xyproto/algernon/lua/onthefly"
	"github.com/xyproto/algernon/lua/pquery"
	"github.com/xyproto/algernon/lua/pure"
	"github.com/xyproto/algernon/lua/upload"
	"github.com/xyproto/algernon/lua/users"
	"github.com/xyproto/algernon/utils"
	"github.com/xyproto/gluamapper"
	lua "github.com/xyproto/gopher-lua"
)

// LoadCommonFunctions adds most of the available Lua functions in algernon to
// the given Lua state struct
func (ac *Config) LoadCommonFunctions(w http.ResponseWriter, req *http.Request, filename string, L *lua.LState, flushFunc func(), httpStatus *FutureStatus) {
	// Make basic functions, like print, available to the Lua script.
	// Only exports functions that can relate to HTTP responses or requests.
	ac.LoadBasicWeb(w, req, L, filename, flushFunc, httpStatus)

	// Make other basic functions available
	ac.LoadBasicSystemFunctions(L)

	// Functions for rendering markdown or amber
	ac.LoadRenderFunctions(w, req, L)

	// If there is a database backend
	if ac.perm != nil {

		// Retrieve the userstate
		userstate := ac.perm.UserState()

		// Set the cookie secret, if set
		if ac.cookieSecret != "" {
			userstate.SetCookieSecret(ac.cookieSecret)
		}

		// Functions for serving files in the same directory as a script
		ac.LoadServeFile(w, req, L, filename)

		// Functions mainly for adding admin prefixes and configuring permissions
		ac.LoadServerConfigFunctions(L, filename)

		// Make the functions related to userstate available to the Lua script
		users.Load(w, req, L, userstate)

		creator := userstate.Creator()

		// Simpleredis data structures
		datastruct.LoadList(L, creator)
		datastruct.LoadSet(L, creator)
		datastruct.LoadHash(L, creator)
		datastruct.LoadKeyValue(L, creator)

		// For saving and loading Lua functions
		codelib.Load(L, creator)

		// For executing PostgreSQL queries
		pquery.Load(L)

		// For executing MSSQL queries
		mssql.Load(L)

	}

	// For handling JSON data
	jnode.LoadJSONFunctions(L)
	ac.LoadJFile(L, filepath.Dir(filename))
	jnode.Load(L)

	// Extras
	pure.Load(L)

	// pprint
	// exportREPL(L)

	// Plugins
	ac.LoadPluginFunctions(L, nil)

	// Cache
	ac.LoadCacheFunctions(L)

	// Pages and Tags
	onthefly.Load(L)

	// Ollama / LLM support
	ollama.Load(L)

	// File uploads
	upload.Load(L, w, req, filepath.Dir(filename))

	// HTTP Client
	httpclient.Load(L, ac.serverHeaderName)
}

// RunLua uses a Lua file as the HTTP handler. Also has access to the userstate
// and permissions. Returns an error if there was a problem with running the lua
// script, otherwise nil.
func (ac *Config) RunLua(w http.ResponseWriter, req *http.Request, filename string, flushFunc func(), fust *FutureStatus) error {
	// Retrieve a Lua state
	L := ac.luapool.Borrow()
	defer ac.luapool.Return(L)

	// Warn if the connection is closed before the script has finished.
	if ac.verboseMode {

		done := make(chan bool)

		// Stop the background goroutine when this function returns
		// There must be a receiver for the done channel,
		// or else this will hang everything!
		defer func() {
			done <- true
		}()

		// Set up a background notifier
		go func() {
			ctx := req.Context()
			for {
				select {
				case <-ctx.Done():
					// Client is done
					logrus.Warn("Connection to client closed")
				case <-done:
					// We are done
					return
				}
			}
		}() // Call the goroutine
	}

	// Export functions to the Lua state
	// Flush can be an uninitialized channel, it is handled in the function.
	ac.LoadCommonFunctions(w, req, filename, L, flushFunc, fust)

	// Run the script and return the error value.
	// Logging and/or HTTP response is handled elsewhere.
	if filepath.Ext(filename) == ".tl" {
		return L.DoString(`
            local fname = [[` + filename + `]]
            local do_cache = ` + strconv.FormatBool(ac.cacheMode == cachemode.Production) + `

            if  do_cache and tl.cache[fname] then
            	tl.cache[fname]()
                return
            end

        	local result, err = tl.process(fname)
            if err ~= nil then
            	throw('Teal failed to process file: '..err)
            end

            if #result.syntax_errors > 0 then
            	local err = result.syntax_errors[1]
                throw(err.filename..':'..err.y..': Teal processing error: '..err.msg, 0)
            end

            local code, gen_error = tl.pretty_print_ast(result.ast, "5.1")
            if gen_error ~= nil then
            	throw('Teal failed to generate Lua: '..err)
            end

            local chunk = load(code)
            if do_cache then
            	tl.cache[fname] = chunk
            end

            chunk()
        `)
	}
	return L.DoFile(filename)
}

// RunConfiguration runs a Lua file as a configuration script. Also has access
// to the userstate and permissions. Returns an error if there was a problem
// with running the lua script, otherwise nil. perm can be nil, but then several
// Lua functions will not be exposed.
//
// The idea is not to change the Lua struct or the luapool, but to set the
// configuration variables with the given Lua configuration script.
//
// luaHandler is a flag that lets Lua functions like "handle" and "servedir" be available or not.
func (ac *Config) RunConfiguration(filename string, mux *http.ServeMux, withHandlerFunctions bool) error {
	// Retrieve a Lua state
	L := ac.luapool.Borrow()

	// Basic system functions, like log()
	ac.LoadBasicSystemFunctions(L)

	// If there is a database backend
	if ac.perm != nil {

		// Retrieve the userstate
		userstate := ac.perm.UserState()

		// Server configuration functions
		ac.LoadServerConfigFunctions(L, filename)

		creator := userstate.Creator()

		// Simpleredis data structures (could be used for storing server stats)
		datastruct.LoadList(L, creator)
		datastruct.LoadSet(L, creator)
		datastruct.LoadHash(L, creator)
		datastruct.LoadKeyValue(L, creator)

		// For saving and loading Lua functions
		codelib.Load(L, creator)

		// For executing PostgreSQL queries
		pquery.Load(L)

		// For executing MSSQL queries
		mssql.Load(L)
	}

	// For handling JSON data
	jnode.LoadJSONFunctions(L)
	ac.LoadJFile(L, filepath.Dir(filename))
	jnode.Load(L)

	// Extras
	pure.Load(L)

	// Plugins
	ac.LoadPluginFunctions(L, nil)

	// Cache
	ac.LoadCacheFunctions(L)

	// Pages and Tags
	onthefly.Load(L)

	// Ollama / LLM support
	ollama.Load(L)

	// HTTP Client
	httpclient.Load(L, ac.serverHeaderName)

	if withHandlerFunctions {
		// Lua HTTP handlers
		ac.LoadLuaHandlerFunctions(L, filename, mux, false, nil, ac.defaultTheme, true)
	}

	// Run the script
	if err := L.DoFile(filename); err != nil {
		// Close the Lua state
		L.Close()

		// Logging and/or HTTP response is handled elsewhere
		return err
	}

	// Only put the Lua state back if there were no errors
	ac.luapool.Return(L)

	// Populate a pool of Lua states dedicated to serving handle() requests,
	// so that concurrent requests can execute on different states in parallel.
	if withHandlerFunctions {
		if err := ac.buildHandlerPool(filename, mux); err != nil {
			return err
		}
	}

	return nil
}

// loadServerConfigNoopFunctions binds the server configuration functions as
// no-ops. Used while populating the handler pool: the user's script will call
// functions like SetAddr or AddReverseProxy during setup, but those effects
// have already been applied on the main state and must not run again per
// pool state.
func (ac *Config) loadServerConfigNoopFunctions(L *lua.LState) {
	noop := L.NewFunction(func(_ *lua.LState) int {
		return 0
	})
	noopTrue := L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LBool(true))
		return 1
	})
	cookieSecret := L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(ac.cookieSecret))
		return 1
	})
	serverInfo := L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(ac.Info()))
		return 1
	})
	for _, name := range []string{
		"SetAddr", "SetCookieSecret", "ClearPermissions",
		"AddUserPrefix", "AddAdminPrefix", "AddReverseProxy",
		"DenyHandler", "OnReady",
	} {
		L.SetGlobal(name, noop)
	}
	for _, name := range []string{"LogTo", "ServerFile", "ServerDir"} {
		L.SetGlobal(name, noopTrue)
	}
	L.SetGlobal("CookieSecret", cookieSecret)
	L.SetGlobal("ServerInfo", serverInfo)
}

// loadPoolStateFunctions binds every function the user's script needs to run
// inside a pool state. Request-scoped functions (LoadCommonFunctions) are
// rebound on every request, so here we only cover what the setup phase uses.
func (ac *Config) loadPoolStateFunctions(L *lua.LState, filename string, mux *http.ServeMux) {
	ac.LoadBasicSystemFunctions(L)
	ac.loadServerConfigNoopFunctions(L)

	if ac.perm != nil {
		userstate := ac.perm.UserState()
		creator := userstate.Creator()
		datastruct.LoadList(L, creator)
		datastruct.LoadSet(L, creator)
		datastruct.LoadHash(L, creator)
		datastruct.LoadKeyValue(L, creator)
		codelib.Load(L, creator)
		pquery.Load(L)
		mssql.Load(L)
	}

	jnode.LoadJSONFunctions(L)
	ac.LoadJFile(L, filepath.Dir(filename))
	jnode.Load(L)
	pure.Load(L)
	ac.LoadPluginFunctions(L, nil)
	ac.LoadCacheFunctions(L)
	onthefly.Load(L)
	ollama.Load(L)
	httpclient.Load(L, ac.serverHeaderName)

	ac.LoadLuaHandlerFunctions(L, filename, mux, false, nil, ac.defaultTheme, false)
}

// buildHandlerPool creates ac.handlerPoolSize fresh Lua states, runs the
// script in each, and enqueues them in ac.handlerPool. Every state ends up
// with its own copy of each handle() function stored in its Lua registry.
func (ac *Config) buildHandlerPool(filename string, mux *http.ServeMux) error {
	size := ac.handlerPoolSize
	if size < 1 {
		size = 1
	}
	pool := newHandlerPool(size)
	for i := 0; i < size; i++ {
		L := lua.NewState()
		ac.loadPoolStateFunctions(L, filename, mux)
		if err := L.DoFile(filename); err != nil {
			L.Close()
			if len(pool.states) == 0 {
				// Couldn't build any usable pool state
				return err
			}
			logrus.Errorf("handler pool state %d failed to initialise: %v", i, err)
			continue
		}
		pool.Add(L)
	}
	ac.handlerPool = pool
	AtShutdown(func() {
		pool.Close()
	})
	return nil
}

// LuaFunctionMap returns the functions available in the given Lua code as
// functions in a map that can be used by templates.
//
// Note that the lua functions must only accept and return strings
// and that only the first returned value will be accessible.
// The Lua functions may take an optional number of arguments.
func (ac *Config) LuaFunctionMap(w http.ResponseWriter, req *http.Request, luadata []byte, filename string) (template.FuncMap, error) {
	ac.pongomutex.Lock()
	defer ac.pongomutex.Unlock()

	// Retrieve a Lua state
	L := ac.luapool.Borrow()
	defer ac.luapool.Return(L)

	// Prepare an empty map of functions (and variables)
	funcs := make(template.FuncMap)

	// Give no filename (an empty string will be handled correctly by the function).
	ac.LoadCommonFunctions(w, req, filename, L, nil, nil)

	// Run the script
	if err := L.DoString(string(luadata)); err != nil {
		// Close the Lua state
		L.Close()

		// Logging and/or HTTP response is handled elsewhere
		return funcs, err
	}

	// Extract the available functions from the Lua state
	globalTable := L.G.Global
	globalTable.ForEach(func(key, value lua.LValue) {
		// Check if the current value is a string variable
		if luaString, ok := value.(lua.LString); ok {
			// Store the variable in the same map as the functions (string -> interface)
			// for ease of use together with templates.
			funcs[key.String()] = luaString.String()
		} else if luaTable, ok := value.(*lua.LTable); ok {

			// Convert the table to a map and save it.
			// Ignore values of a different type.
			mapinterface, _ := convert.Table2map(luaTable, false)
			switch m := mapinterface.(type) {
			case map[string]string:
				funcs[key.String()] = map[string]string(m)
			case map[string]int:
				funcs[key.String()] = map[string]int(m)
			case map[int]string:
				funcs[key.String()] = map[int]string(m)
			case map[int]int:
				funcs[key.String()] = map[int]int(m)
			}

			// Check if the current value is a function
		} else if luaFunc, ok := value.(*lua.LFunction); ok {
			// Only export the functions defined in the given Lua code,
			// not all the global functions. IsG is true if the function is global.
			if !luaFunc.IsG {

				functionName := key.String()

				// Register the function, with a variable number of string arguments
				// Functions returning (string, error) are supported by html.template
				funcs[functionName] = func(args ...string) (any, error) {
					// Create a brand new Lua state
					L2 := ac.luapool.New()
					defer L2.Close()

					// Set up a new Lua state with the current http.ResponseWriter and *http.Request
					ac.LoadCommonFunctions(w, req, filename, L2, nil, nil)

					// Push the Lua function to run
					L2.Push(luaFunc)

					// Push the given arguments
					for _, arg := range args {
						L2.Push(lua.LString(arg))
					}

					// Run the Lua function
					err := L2.PCall(len(args), lua.MultRet, nil)
					if err != nil {
						// If calling the function did not work out, return the infostring and error
						return utils.Infostring(functionName, args), err
					}

					// Empty return value if no values were returned
					var retval any

					// Return the first of the returned arguments, as a string
					if L2.GetTop() >= 1 {
						lv := L2.Get(-1)
						tbl, isTable := lv.(*lua.LTable)
						switch {
						case isTable:
							// lv was a Lua Table
							retval = gluamapper.ToGoValue(tbl, gluamapper.Option{
								NameFunc: func(s string) string {
									return s
								},
							})
							if ac.debugMode && ac.verboseMode {
								logrus.Infof("%s -> (map)", utils.Infostring(functionName, args))
							}
						case lv.Type() == lua.LTString:
							// lv is a Lua String
							retstr := L2.ToString(1)
							retval = retstr
							if ac.debugMode && ac.verboseMode {
								logrus.Infof("%s -> %q", utils.Infostring(functionName, args), retstr)
							}
						case lv.Type() == lua.LTNumber:
							// lv is a Lua Number
							retnum := L2.ToNumber(1)
							retval = retnum
							if ac.debugMode && ac.verboseMode {
								logrus.Infof("%s -> %v", utils.Infostring(functionName, args), retnum)
							}
						default:
							retval = "!! not a table, string or number !!"
							logrus.Warnf("The returned value from %s has an unrecognized type: %v", utils.Infostring(functionName, args), lv.Type())
						}
					}

					// No return value, return an empty string and nil
					return retval, nil
				}
			}
		}
	})

	// Return the map of functions
	return funcs, nil
}
