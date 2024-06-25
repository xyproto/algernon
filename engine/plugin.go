package engine

import (
	"encoding/json"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"path/filepath"
	"strings"

	"github.com/natefinch/pie"
	"github.com/xyproto/algernon/platformdep"
	lua "github.com/xyproto/gopher-lua"
	"github.com/xyproto/textoutput"
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

// LoadPluginFunctions takes a Lua state and a TextOutput
// (the TextOutput struct should be nil if not in a REPL)
func (ac *Config) LoadPluginFunctions(L *lua.LState, o *textoutput.TextOutput) {
	// Expose the functionality of a given plugin (executable file).
	// If on Windows, ".exe" (platformdep.ExeExt) is added to the path.
	// Returns true of successful.
	L.SetGlobal("Plugin", L.NewFunction(func(L *lua.LState) int {
		path := L.ToString(1)
		givenPath := path
		path += platformdep.ExeExt
		if !ac.fs.Exists(path) {
			path = filepath.Join(ac.serverDirOrFilename, path)
		}

		// Keep the plugin running in the background?
		keepRunning := false
		if L.GetTop() >= 2 {
			keepRunning = L.ToBool(2)
		}

		// Connect with the Plugin
		client, err := pie.StartProviderCodec(jsonrpc.NewClientCodec, os.Stderr, path)
		if err != nil {
			if o != nil {
				o.Err("[Plugin] Could not run plugin!")
				o.Err("Error: " + err.Error())
			}
			L.Push(lua.LBool(false)) // Fail
			return 1                 // number of results
		}

		if !keepRunning {
			// Close the client once this function has completed
			defer client.Close()
		}

		p := &luaPlugin{client}

		// Retrieve the Lua code
		luacode, err := p.LuaCode(givenPath)
		if err != nil {
			if o != nil {
				o.Err("[Plugin] Could not call the LuaCode function!")
				o.Err("Error: " + err.Error())
			}
			L.Push(lua.LBool(false)) // Fail
			return 1                 // number of results
		}

		// Retrieve the help text
		luahelp, err := p.LuaHelp()
		if err != nil {
			if o != nil {
				o.Err("[Plugin] Could not call the LuaHelp function!")
				o.Err("Error: " + err.Error())
			}
			L.Push(lua.LBool(false)) // Fail
			return 1                 // number of results
		}

		// Run luacode on the current LuaState
		luacode = strings.TrimSpace(luacode)
		if L.DoString(luacode) != nil {
			if o != nil {
				o.Err("[Plugin] Error in Lua code provided by plugin!")
				o.Err("Error: " + err.Error())
			}
			L.Push(lua.LBool(false)) // Fail
			return 1                 // number of results
		}

		// If in a REPL, output the Plugin help text
		if o != nil {
			luahelp = strings.TrimSpace(luahelp)
			// Add syntax highlighting and output the text
			o.Println(highlight(o, luahelp))
		}

		L.Push(lua.LBool(true)) // Success
		return 1                // number of results
	}))

	// Retrieve the code from the Lua.Code function of the plugin
	L.SetGlobal("PluginCode", L.NewFunction(func(L *lua.LState) int {
		path := L.ToString(1)
		givenPath := path
		path += platformdep.ExeExt
		if !ac.fs.Exists(path) {
			path = filepath.Join(ac.serverDirOrFilename, path)
		}

		// Keep the plugin running in the background?
		keepRunning := false
		if L.GetTop() >= 2 {
			keepRunning = L.ToBool(2)
		}

		// Connect with the Plugin
		client, err := pie.StartProviderCodec(jsonrpc.NewClientCodec, os.Stderr, path)
		if err != nil {
			if o != nil {
				o.Err("[PluginCode] Could not run plugin!")
				o.Err("Error: " + err.Error())
			}
			L.Push(lua.LString("")) // Fail
			return 1                // number of results
		}
		if !keepRunning {
			// Close the client once this function has completed
			defer client.Close()
		}

		p := &luaPlugin{client}

		// Retrieve the Lua code
		luacode, err := p.LuaCode(givenPath)
		if err != nil {
			if o != nil {
				o.Err("[PluginCode] Could not call the LuaCode function!")
				o.Err("Error: " + err.Error())
			}
			L.Push(lua.LString("")) // Fail
			return 1                // number of results
		}

		L.Push(lua.LString(luacode))
		return 1 // number of results
	}))

	// Call a function exposed by a plugin (executable file)
	// Returns either nil (fail) or a string (success)
	L.SetGlobal("CallPlugin", L.NewFunction(func(L *lua.LState) int {
		if L.GetTop() < 2 {
			if o != nil {
				o.Err("[CallPlugin] Needs at least 2 arguments")
			}
			L.Push(lua.LString("")) // Fail
			return 1                // number of results
		}

		path := L.ToString(1)
		path += platformdep.ExeExt
		if !ac.fs.Exists(path) {
			path = filepath.Join(ac.serverDirOrFilename, path)
		}

		fn := L.ToString(2)

		var args []lua.LValue
		if L.GetTop() > 2 {
			for i := 3; i <= L.GetTop(); i++ {
				args = append(args, L.Get(i))
			}
		}

		// Connect with the Plugin
		logto := os.Stderr
		if o != nil {
			logto = os.Stdout
		}

		// Keep the plugin running in the background?
		keepRunning := false

		client, err := pie.StartProviderCodec(jsonrpc.NewClientCodec, logto, path)
		if err != nil {
			if o != nil {
				o.Err("[CallPlugin] Could not run plugin!")
				o.Err("Error: " + err.Error())
			}
			L.Push(lua.LString("")) // Fail
			return 1                // number of results
		}

		if !keepRunning {
			// Close the client once this function has completed
			defer client.Close()
		}

		jsonargs, err := json.Marshal(args)
		if err != nil {
			if o != nil {
				o.Err("[CallPlugin] Error when marshalling arguments to JSON")
				o.Err("Error: " + err.Error())
			}
			L.Push(lua.LString("")) // Fail
			return 1                // number of results
		}

		// Attempt to call the given function name
		var jsonreply []byte
		if err := client.Call(namespace+"."+fn, jsonargs, &jsonreply); err != nil {
			if o != nil {
				o.Err("[CallPlugin] Error when calling function!")
				o.Err("Function: " + namespace + "." + fn)
				o.Err("JSON Arguments: " + string(jsonargs))
				o.Err("Error: " + err.Error())
			}
			L.Push(lua.LString("")) // Fail
			return 1                // number of results
		}

		L.Push(lua.LString(jsonreply)) // Resulting string
		return 1                       // number of results
	}))
}
