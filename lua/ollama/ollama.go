// Package ollama provides Lua functions for communicting with a local Ollama server
package ollama

import (
	log "github.com/sirupsen/logrus"
	lua "github.com/xyproto/gopher-lua"
	"github.com/xyproto/ollamaclient"
)

const (
	// Class is an identifier for the OllamaClient class in Lua
	Class = "OllamaClient"

	defaultModel = "tinyllama"
)

// Get the first argument, "self", and cast it from userdata to a library (which is really a hash map).
func checkOllamaClient(L *lua.LState) *ollamaclient.Config {
	ud := L.CheckUserData(1)
	if ollama, ok := ud.Value.(*ollamaclient.Config); ok {
		return ollama
	}
	L.ArgError(1, "ollamaclient.Config expected")
	return nil
}

// ollamaPullIfNeeded will download the active model, if it's missing
// it takes an optional "verbose" argument that is bool.
func ollamaPullIfNeeded(L *lua.LState) int {
	oc := checkOllamaClient(L) // arg 1
	//top := L.GetTop()
	//if top == 2 {
	//oc.Verbose = bool(L.ToBool(2))
	//}
	//log.Printf("OLLAMA VERBOSE is %s\n", oc.Verbose)
	oc.Verbose = true
	err := oc.PullIfNeeded()
	oc.Verbose = false
	if err != nil {
		log.Error(err)
		L.Push(lua.LString(err.Error()))
		return 1 // number of results
	}
	return 0 // number of results
}

func ollamaGetOutput(L *lua.LState) int {
	oc := checkOllamaClient(L) // arg 1
	prompt := "Write a haiku about the poet Algernon"
	top := L.GetTop()
	if top == 2 {
		prompt = L.ToString(2)
	}
	output, err := oc.GetOutput(prompt)
	if err != nil {
		log.Error(err)
		L.Push(lua.LString(err.Error()))
		return 1 // number of results
	}
	L.Push(lua.LString(output))
	return 1 // number of results
}

func constructOllamaClient(L *lua.LState) (*lua.LUserData, error) {
	// Create a new OllamaClient
	var oc *ollamaclient.Config

	top := L.GetTop()
	if top == 1 {
		// Optional
		modelAndOptionalTag := L.ToString(1)
		oc = ollamaclient.NewWithModel(modelAndOptionalTag)
	} else {
		oc = ollamaclient.NewWithModel(defaultModel)
	}
	// Create a new userdata struct
	ud := L.NewUserData()
	ud.Value = oc
	L.SetMetatable(ud, L.GetTypeMetatable(Class))
	return ud, nil
}

// The hash map methods that are to be registered
var ollamaMethods = map[string]lua.LGFunction{
	"pull": ollamaPullIfNeeded,
	"say":  ollamaGetOutput,
}

func askOllama(L *lua.LState) int {
	oc := ollamaclient.New()
	oc.Verbose = false
	err := oc.PullIfNeeded()
	if err != nil {
		log.Error(err)
		L.Push(lua.LString(err.Error()))
		return 1 // number of results
	}
	prompt := "Write a haiku about the poet Algernon"
	top := L.GetTop()
	if top == 1 {
		prompt = L.ToString(1)
	}
	output, err := oc.GetOutput(prompt)
	if err != nil {
		log.Error(err)
		L.Push(lua.LString(err.Error()))
		return 1 // number of results
	}
	L.Push(lua.LString(output))
	return 1 // number of results
}

// Load makes functions related Ollama clients available to the given Lua state
func Load(L *lua.LState) {
	// Register the OllamaClient class and the methods that belongs with it.
	mt := L.NewTypeMetatable(Class)
	mt.RawSetH(lua.LString("__index"), mt)
	L.SetFuncs(mt, ollamaMethods)

	// The constructor for new Libraries takes only an optional id
	L.SetGlobal("OllamaClient", L.NewFunction(func(L *lua.LState) int {
		// Construct a new OllamaClient
		userdata, err := constructOllamaClient(L)
		if err != nil {
			log.Error(err)
			L.Push(lua.LString(err.Error()))
			return 1 // Number of returned values
		}

		// Return the Lua OllamaClient object
		L.Push(userdata)
		return 1 // number of results
	}))

	L.SetGlobal("ollama", L.NewFunction(askOllama))
}
