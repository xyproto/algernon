package main

import (
	"bytes"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/xyproto/permissions2"
	"github.com/yuin/gopher-lua"
)

// Used when rendering templates that get values from Lua functions
// TODO: Try using html/template.FuncMap instead
type LuaDefinedGoFunctions map[string]interface{}

// Retrieve all the arguments given to a lua function
// and gather the strings in a buffer.
func arguments2buffer(L *lua.LState) bytes.Buffer {
	var buf bytes.Buffer
	top := L.GetTop()
	// Add all the string arguments to the buffer
	for i := 1; i <= top; i++ {
		buf.WriteString(L.Get(i).String())
		if i != top {
			buf.WriteString(" ")
		}
	}
	buf.WriteString("\n")
	return buf
}

// Convert a string slice to a lua table
func strings2table(L *lua.LState, sl []string) *lua.LTable {
	table := L.NewTable()
	for _, element := range sl {
		table.Append(lua.LString(element))
	}
	return table
}

// Return a *lua.LState object that contains several exposed functions
func exportCommonFunctions(w http.ResponseWriter, req *http.Request, filename string, perm *permissions.Permissions, L *lua.LState) {

	// Retrieve the userstate
	userstate := perm.UserState()

	// Make basic functions, like print, available to the Lua script.
	// Only exports functions that can relate to HTTP responses or requests.
	exportBasicWeb(w, req, L, filename)

	// Make other basic functions available
	exportBasicSystemFunctions(L)

	// Functions for rendering markdown or amber
	exportRenderFunctions(w, req, L)

	// Make the functions related to userstate available to the Lua script
	exportUserstate(w, req, L, userstate)

	// Simpleredis data structures
	exportList(L, userstate)
	exportSet(L, userstate)
	exportHash(L, userstate)
	exportKeyValue(L, userstate)
}

// Run a Lua file as a HTTP handler. Also has access to the userstate and permissions.
// Returns an error if there was a problem with running the lua script, otherwise nil.
func runLua(w http.ResponseWriter, req *http.Request, filename string, perm *permissions.Permissions, luapool *lStatePool) error {
	// Retrieve a Lua state
	L := luapool.Get()
	defer luapool.Put(L)

	// Export functions to the Lua state
	exportCommonFunctions(w, req, filename, perm, L)

	// Run the script
	if err := L.DoFile(filename); err != nil {
		// Logging and/or HTTP response is handled elsewhere
		return err
	}

	return nil
}

// Run a Lua string as a HTTP handler. Also has access to the userstate and permissions.
// Returns an error if there was a problem with running the lua script, otherwise nil.
func runLuaString(w http.ResponseWriter, req *http.Request, script string, perm *permissions.Permissions, luapool *lStatePool) error {

	// Retrieve a Lua state
	L := luapool.Get()

	// Give no filename (an empty string will be handled correctly by the function).
	exportCommonFunctions(w, req, "", perm, L)

	// Run the script
	if err := L.DoString(script); err != nil {
		// Close the Lua state
		L.Close()

		// Logging and/or HTTP response is handled elsewhere
		return err
	}

	// TODO Figure out if the Lua state should rather be put back in either case
	// Only put the Lua state back if there were no errors
	luapool.Put(L)

	return nil
}

// Run a Lua file as a configuration script. Also has access to the userstate and permissions.
// Returns an error if there was a problem with running the lua script, otherwise nil.
func runConfiguration(filename string, perm *permissions.Permissions, luapool *lStatePool) error {

	// Retrieve a Lua state
	L := luapool.Get()

	// Retrieve the userstate
	userstate := perm.UserState()

	// Server configuration functions
	exportServerConfigFunctions(L, perm, filename)

	// Other basic system functions, like log()
	exportBasicSystemFunctions(L)

	// Simpleredis data structures (could be used for storing server stats)
	exportList(L, userstate)
	exportSet(L, userstate)
	exportHash(L, userstate)
	exportKeyValue(L, userstate)

	// Run the script
	if err := L.DoFile(filename); err != nil {
		// Close the Lua state
		L.Close()

		// Logging and/or HTTP response is handled elsewhere
		return err
	}

	// Only put the Lua state back if there were no errors
	luapool.Put(L)

	return nil
}

// Return the functions available in the given Lua data as
// functions in a map that can be used for templates
//
// NOTE that all functions must:
// * return nothing or
// * return a string or
// * return a string and an error
//
func luaFunctionMap(w http.ResponseWriter, req *http.Request, luadata []byte, filename string, perm *permissions.Permissions, L *lua.LState) (LuaDefinedGoFunctions, error) {

	// Prepare an empty map of functions
	funcs := make(LuaDefinedGoFunctions)

	// Give no filename (an empty string will be handled correctly by the function).
	exportCommonFunctions(w, req, filename, perm, L)

	// TODO: 1. Run the Lua script
	// TODO: 2. Find the functions from the L object
	// TODO: 3. Make the functions available as Go functions
	// TODO: 4. Put the Go functions in the map, and return it

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
		// Check if the current value is a function
		if luaFunc, ok := value.(*lua.LFunction); ok {
			//fmt.Println("FUNCTION", key)
			//fmt.Println("IsG", lfunc.IsG)
			// Only export the functions defined in the given Lua code,
			// not all the global functions. IsG is true if the function is global.
			if !luaFunc.IsG {
				// Register the function, with a variable number of string arguments
				funcs[key.String()] = func(args ...string) (string, error) {

					// Debug information
					infostring := ""
					if DEBUG_MODE {
						// Build up a string on the form "functionname(arg1, arg2, arg3)"
						infostring = key.String() + "("
						if len(args) > 0 {
							infostring += "\"" + strings.Join(args, "\", \"") + "\""
						}
						infostring += ")"
					}

					// TODO: Must create a new Lua state here!! See also serverconf.go for exportCommonFunctions.

					// Push the Lua function to run
					L.Push(luaFunc)

					// Push the given arguments
					for _, arg := range args {
						L.Push(lua.LString(arg))
					}

					// Run the Lua function
					if err := L.PCall(len(args), lua.MultRet, nil); err != nil {
						// If calling the function did not work out, return the infostring and error
						return infostring, err
					}

					// Empty return value if no values were returned
					retval := ""

					// Return the first of the returned arguments, as a string
					if L.GetTop() >= 1 {
						retval = L.ToString(1)
					}

					if DEBUG_MODE {
						log.Info(infostring + " -> \"" + retval + "\"")
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
