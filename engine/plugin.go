package engine

import (
	"encoding/json"
	"io"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"path/filepath"
	"strings"

	"github.com/natefinch/pie"
	"github.com/xyproto/algernon/platformdep"
	lua "github.com/xyproto/gopher-lua"
	"github.com/xyproto/vt"
)

type luaPlugin struct {
	client *rpc.Client
}

const namespace = "Lua"

func (lp *luaPlugin) LuaCode(pluginPath string) (luacode string, err error) {
	return luacode, lp.client.Call(namespace+".Code", pluginPath, &luacode)
}

func (lp *luaPlugin) LuaHelp() (luahelp string, err error) {
	return luahelp, lp.client.Call(namespace+".Help", "", &luahelp)
}

// startPlugin returns an rpc.Client for the plugin at path.
// The persistent cache is always checked first, so CallPlugin automatically
// reuses any client stored by an earlier Plugin(..., true) call.
// If keepRunning is true and no cached client exists, the new client is stored
// for reuse across requests. The second return value is true when the caller
// owns the client and must close it.
func (ac *Config) startPlugin(path string, keepRunning bool, logto io.Writer) (*rpc.Client, bool, error) {
	// Check the cache. Reading a nil map is safe in Go (returns zero value).
	ac.pluginClientsMu.Lock()
	c, ok := ac.pluginClients[path]
	ac.pluginClientsMu.Unlock()
	if ok {
		return c, false, nil
	}

	c, err := pie.StartProviderCodec(jsonrpc.NewClientCodec, logto, path)
	if err != nil {
		return nil, false, err
	}

	if keepRunning {
		ac.pluginClientsMu.Lock()
		if existing, ok := ac.pluginClients[path]; ok {
			// A concurrent goroutine already stored a client for this path.
			ac.pluginClientsMu.Unlock()
			c.Close()
			return existing, false, nil
		}
		if ac.pluginClients == nil {
			ac.pluginClients = make(map[string]*rpc.Client)
		}
		ac.pluginClients[path] = c
		ac.pluginClientsMu.Unlock()
		return c, false, nil
	}

	return c, true, nil
}

// closePluginClients closes all cached plugin clients. Called at shutdown.
func (ac *Config) closePluginClients() {
	ac.pluginClientsMu.Lock()
	defer ac.pluginClientsMu.Unlock()
	for _, c := range ac.pluginClients {
		c.Close()
	}
	ac.pluginClients = nil
}

// LoadPluginFunctions takes a Lua state and a TextOutput
// (the TextOutput struct should be nil if not in a REPL)
func (ac *Config) LoadPluginFunctions(L *lua.LState, o *vt.TextOutput) {
	// Expose the functionality of a given plugin (executable file).
	// If on Windows, ".exe" (platformdep.ExeExt) is added to the path.
	// Returns true if successful.
	L.SetGlobal("Plugin", L.NewFunction(func(L *lua.LState) int {
		path := L.ToString(1)
		givenPath := path
		path += platformdep.ExeExt
		if !ac.fs.Exists(path) {
			path = filepath.Join(ac.serverDirOrFilename, path)
		}

		keepRunning := L.GetTop() >= 2 && L.ToBool(2)

		client, owned, err := ac.startPlugin(path, keepRunning, os.Stderr)
		if err != nil {
			if o != nil {
				o.Err("[Plugin] Could not run plugin!")
				o.Err("Error: " + err.Error())
			}
			L.Push(lua.LBool(false))
			return 1
		}
		if owned {
			defer client.Close()
		}

		p := &luaPlugin{client}

		luacode, err := p.LuaCode(givenPath)
		if err != nil {
			if o != nil {
				o.Err("[Plugin] Could not call the LuaCode function!")
				o.Err("Error: " + err.Error())
			}
			L.Push(lua.LBool(false))
			return 1
		}

		luahelp, err := p.LuaHelp()
		if err != nil {
			if o != nil {
				o.Err("[Plugin] Could not call the LuaHelp function!")
				o.Err("Error: " + err.Error())
			}
			L.Push(lua.LBool(false))
			return 1
		}

		luacode = strings.TrimSpace(luacode)
		if err := L.DoString(luacode); err != nil {
			if o != nil {
				o.Err("[Plugin] Error in Lua code provided by plugin!")
				o.Err("Error: " + err.Error())
			}
			L.Push(lua.LBool(false))
			return 1
		}

		if o != nil {
			o.Println(highlight(strings.TrimSpace(luahelp)))
		}

		L.Push(lua.LBool(true))
		return 1
	}))

	// Retrieve the code from the Lua.Code function of the plugin
	L.SetGlobal("PluginCode", L.NewFunction(func(L *lua.LState) int {
		path := L.ToString(1)
		givenPath := path
		path += platformdep.ExeExt
		if !ac.fs.Exists(path) {
			path = filepath.Join(ac.serverDirOrFilename, path)
		}

		keepRunning := L.GetTop() >= 2 && L.ToBool(2)

		client, owned, err := ac.startPlugin(path, keepRunning, os.Stderr)
		if err != nil {
			if o != nil {
				o.Err("[PluginCode] Could not run plugin!")
				o.Err("Error: " + err.Error())
			}
			L.Push(lua.LString(""))
			return 1
		}
		if owned {
			defer client.Close()
		}

		p := &luaPlugin{client}

		luacode, err := p.LuaCode(givenPath)
		if err != nil {
			if o != nil {
				o.Err("[PluginCode] Could not call the LuaCode function!")
				o.Err("Error: " + err.Error())
			}
			L.Push(lua.LString(""))
			return 1
		}

		L.Push(lua.LString(luacode))
		return 1
	}))

	// Call a function exposed by a plugin (executable file).
	// Returns either an empty string (fail) or the JSON reply as a string (success).
	// Uses a persistent client if one was registered by Plugin(..., true).
	L.SetGlobal("CallPlugin", L.NewFunction(func(L *lua.LState) int {
		if L.GetTop() < 2 {
			if o != nil {
				o.Err("[CallPlugin] Needs at least 2 arguments")
			}
			L.Push(lua.LString(""))
			return 1
		}

		path := L.ToString(1)
		path += platformdep.ExeExt
		if !ac.fs.Exists(path) {
			path = filepath.Join(ac.serverDirOrFilename, path)
		}

		fn := L.ToString(2)

		var args []lua.LValue
		for i := 3; i <= L.GetTop(); i++ {
			args = append(args, L.Get(i))
		}

		logto := os.Stderr
		if o != nil {
			logto = os.Stdout
		}

		client, owned, err := ac.startPlugin(path, false, logto)
		if err != nil {
			if o != nil {
				o.Err("[CallPlugin] Could not run plugin!")
				o.Err("Error: " + err.Error())
			}
			L.Push(lua.LString(""))
			return 1
		}
		if owned {
			defer client.Close()
		}

		jsonargs, err := json.Marshal(args)
		if err != nil {
			if o != nil {
				o.Err("[CallPlugin] Error when marshalling arguments to JSON")
				o.Err("Error: " + err.Error())
			}
			L.Push(lua.LString(""))
			return 1
		}

		var jsonreply []byte
		if err := client.Call(namespace+"."+fn, jsonargs, &jsonreply); err != nil {
			if o != nil {
				o.Err("[CallPlugin] Error when calling function!")
				o.Err("Function: " + namespace + "." + fn)
				o.Err("JSON Arguments: " + string(jsonargs))
				o.Err("Error: " + err.Error())
			}
			L.Push(lua.LString(""))
			return 1
		}

		L.Push(lua.LString(jsonreply))
		return 1
	}))
}
