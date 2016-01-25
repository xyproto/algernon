package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/rpc/jsonrpc"
	"strings"

	"github.com/natefinch/pie"
)

// LuaPlugin represents a plugin for Algernon (for Lua)
type LuaPlugin struct{}

const namespace = "Lua"

// --- Plugin functionality ---

func add3(a, b int) int {
	// Functionality not otherwise available in Lua goes here
	return a + b + 3
}

// --- Lua wrapper code ($0 is replaced with the plugin path) ---

const luacode = `
function add3(a, b)
  return CallPlugin("$0", "Add3", a, b)
end
`

// --- Lua help text (will be syntax highlighted) ---

const luahelp = `
add3(number, number) -> number // Adds two numbers and then the number 3
`

// --- Plugin wrapper functions ---

// Add3 is exposed to Algernon
func (LuaPlugin) Add3(jsonargs []byte, response *[]byte) (err error) {
	var args []int
	err = json.Unmarshal(jsonargs, &args)
	if err != nil || len(args) < 2 {
		// Could not unmarshal the given arguments, or too few arguments
		return errors.New("add3 requires two integer arguments")
	}
	result := add3(args[0], args[1])
	*response, err = json.Marshal(result)
	return
}

// --- Plugin functions that must be present ---

// Code is called once when the Plugin function is used in Algernon
func (LuaPlugin) Code(pluginPath string, response *string) error {
	*response = strings.Replace(luacode, "$0", pluginPath, -1)
	return nil
}

// Help is called once when the help function is used in Algernon
func (LuaPlugin) Help(_ string, response *string) error {
	*response = luahelp
	return nil
}

// Called once when the Plugin or CallPlugin function is used in Algernon
func main() {
	log.SetPrefix("[plugin log] ")
	p := pie.NewProvider()
	if err := p.RegisterName(namespace, LuaPlugin{}); err != nil {
		log.Fatalf("Failed to register plugin: %s", err)
	}
	p.ServeCodec(jsonrpc.NewServerCodec)
}
