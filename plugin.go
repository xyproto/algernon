package main

import (
	"encoding/json"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/natefinch/pie"
	"github.com/xyproto/term"
	"github.com/yuin/gopher-lua"
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

// Takes a Lua state and a term Output (should be nil if not in a REPL)
func (ac *algernonConfig) exportPluginFunctions(L *lua.LState, o *term.TextOutput) {

	// Expose the functionality of a given plugin (executable file).
	// If on Windows, ".exe" is added to the path.
	// Returns true of successful.
	L.SetGlobal("Plugin", L.NewFunction(func(L *lua.LState) int {
		path := L.ToString(1)
		givenPath := path
		if runtime.GOOS == "windows" {
			path = path + ".exe"
		}
		if !fs.Exists(path) {
			path = filepath.Join(ac.serverDirOrFilename, path)
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
		// May cause a data race
		//defer client.Close()
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
		if runtime.GOOS == "windows" {
			path = path + ".exe"
		}
		if !fs.Exists(path) {
			path = filepath.Join(ac.serverDirOrFilename, path)
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
		// May cause a data race
		//defer client.Close()
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
		if runtime.GOOS == "windows" {
			path = path + ".exe"
		}
		if !fs.Exists(path) {
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
		client, err := pie.StartProviderCodec(jsonrpc.NewClientCodec, logto, path)
		if err != nil {
			if o != nil {
				o.Err("[CallPlugin] Could not run plugin!")
				o.Err("Error: " + err.Error())
			}
			L.Push(lua.LString("")) // Fail
			return 1                // number of results
		}
		// May cause a data race
		//defer client.Close()

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
