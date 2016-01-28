package main

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/xyproto/pinterface"
	"github.com/yuin/gopher-lua"
)

// Retrieve all the arguments given to a lua function
// and gather the strings in a buffer.
func arguments2buffer(L *lua.LState, addNewline bool) bytes.Buffer {
	var buf bytes.Buffer
	top := L.GetTop()
	// Add all the string arguments to the buffer
	for i := 1; i <= top; i++ {
		buf.WriteString(L.Get(i).String())
		if i != top {
			buf.WriteString(" ")
		}
	}
	if addNewline {
		buf.WriteString("\n")
	}
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

// Convert a map[string]string to a lua table
func map2table(L *lua.LState, m map[string]string) *lua.LTable {
	table := L.NewTable()
	for key, value := range m {
		L.RawSet(table, lua.LString(key), lua.LString(value))
	}
	return table
}

// Convert a Lua table to one of the following types, depending on the content:
// map[string]string, map[string]int, map[int]string, map[int]int
// If no suitable keys and values are found, a nil interface is returned.
// If several different types are found, multiple is set to true.
func table2map(luaTable *lua.LTable) (interface{}, bool) {

	mapSS, mapSI, mapIS, mapII := table2maps(luaTable)

	lss := len(mapSS)
	lsi := len(mapSI)
	lis := len(mapIS)
	lii := len(mapII)

	total := lss + lsi + lis + lii

	// Return the first map that has values
	if lss > 0 {
		//log.Println(key, "STRING -> STRING map")
		return interface{}(mapSS), lss < total
	} else if lsi > 0 {
		//log.Println(key, "STRING -> INT map")
		return interface{}(mapSI), lsi < total
	} else if lis > 0 {
		//log.Println(key, "INT -> STRING map")
		return interface{}(mapIS), lis < total
	} else if lii > 0 {
		//log.Println(key, "INT -> INT map")
		return interface{}(mapII), lii < total
	}

	return nil, false
}

// Convert a Lua table to all of the following types, depending on the content:
// map[string]string, map[string]int, map[int]string, map[int]int
func table2maps(luaTable *lua.LTable) (map[string]string, map[string]int, map[int]string, map[int]int) {

	// Initialize possible maps we want to convert to
	mapSS := make(map[string]string)
	mapSI := make(map[string]int)
	mapIS := make(map[int]string)
	mapII := make(map[int]int)

	var skey, svalue lua.LString
	var ikey, ivalue lua.LNumber
	var hasSkey, hasIkey, hasSvalue, hasIvalue bool

	luaTable.ForEach(func(tkey, tvalue lua.LValue) {

		// Convert the keys and values to strings or ints
		skey, hasSkey = tkey.(lua.LString)
		ikey, hasIkey = tkey.(lua.LNumber)
		svalue, hasSvalue = tvalue.(lua.LString)
		ivalue, hasIvalue = tvalue.(lua.LNumber)

		// Store the right keys and values in the right maps
		if hasSkey && hasSvalue {
			mapSS[skey.String()] = svalue.String()
		} else if hasSkey && hasIvalue {
			mapSI[skey.String()] = int(ivalue)
		} else if hasIkey && hasSvalue {
			mapIS[int(ikey)] = svalue.String()
		} else if hasIkey && hasIvalue {
			mapII[int(ikey)] = int(ivalue)
		}
	})

	return mapSS, mapSI, mapIS, mapII
}

// Convert a Lua table to a map[string]interface{}
func table2interfacemap(luaTable *lua.LTable) map[string]interface{} {

	// Initialize possible maps we want to convert to
	everything := make(map[string]interface{})

	var skey, svalue lua.LString
	var nkey, nvalue lua.LNumber
	var hasSkey, hasSvalue, hasNkey, hasNvalue bool

	luaTable.ForEach(func(tkey, tvalue lua.LValue) {

		// Convert the keys and values to strings or ints
		skey, hasSkey = tkey.(lua.LString)
		nkey, hasNkey = tkey.(lua.LNumber)
		svalue, hasSvalue = tvalue.(lua.LString)
		nvalue, hasNvalue = tvalue.(lua.LNumber)

		// Store the right keys and values in the right maps
		if hasSkey && hasSvalue {
			everything[skey.String()] = svalue.String()
		} else if hasSkey && hasNvalue {
			floatVal := float64(nvalue)
			intVal := int(nvalue)
			// Use the int value if it's the same as the float representation
			if floatVal == float64(intVal) {
				everything[skey.String()] = intVal
			} else {
				everything[skey.String()] = floatVal
			}
		} else if hasNkey && hasSvalue {
			floatKey := float64(nkey)
			intKey := int(nkey)
			// Use the int key if it's the same as the float representation
			if floatKey == float64(intKey) {
				everything[fmt.Sprintf("%d", intKey)] = svalue.String()
			} else {
				everything[fmt.Sprintf("%f", floatKey)] = svalue.String()
			}
		} else if hasNkey && hasNvalue {
			var sk, sv string
			floatKey := float64(nkey)
			intKey := int(nkey)
			floatVal := float64(nvalue)
			intVal := int(nvalue)
			// Use the int key if it's the same as the float representation
			if floatKey == float64(intKey) {
				sk = fmt.Sprintf("%d", intKey)
			} else {
				sk = fmt.Sprintf("%f", floatKey)
			}
			// Use the int value if it's the same as the float representation
			if floatVal == float64(intVal) {
				sv = fmt.Sprintf("%d", intVal)
			} else {
				sv = fmt.Sprintf("%f", floatVal)
			}
			everything[sk] = sv
		} else {
			log.Warn("table2interfacemap: Unsupported type for map key. Value:", tvalue)
		}
	})

	return everything
}

// Return a *lua.LState object that contains several exposed functions
func exportCommonFunctions(w http.ResponseWriter, req *http.Request, filename string, perm pinterface.IPermissions, L *lua.LState, luapool *lStatePool, flushFunc func(), cache *FileCache) {

	// Retrieve the userstate
	userstate := perm.UserState()

	// Make basic functions, like print, available to the Lua script.
	// Only exports functions that can relate to HTTP responses or requests.
	exportBasicWeb(w, req, L, filename, flushFunc)

	// Functions for serving files in the same directory as a script
	exportServeFile(w, req, L, filename, perm, luapool, cache)

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

	// For handling JSON data
	exportJSONFunctions(L)
	exportJFile(L, filepath.Dir(filename))
	exportJNode(L)

	// For saving and loading Lua functions
	exportCodeLibrary(L, userstate)

	// pprint
	//exportREPL(L)

	// Plugins
	exportPluginFunctions(L, nil)

	// Cache
	exportCacheFunctions(L, cache)

	// File uploads
	exportUploadedFile(L, w, req, filepath.Dir(filename))
}

// Run a Lua file as a HTTP handler. Also has access to the userstate and permissions.
// Returns an error if there was a problem with running the lua script, otherwise nil.
func runLua(w http.ResponseWriter, req *http.Request, filename string, perm pinterface.IPermissions, luapool *lStatePool, flushFunc func(), cache *FileCache) error {

	// Retrieve a Lua state
	L := luapool.Get()
	defer luapool.Put(L)

	// Warn if the connection is closed before the script has finished.
	// Requires that the requestWriter has CloseNotify.
	if verboseMode {

		done := make(chan bool)

		// Stop the background goroutine when this function returns
		// There must be a receiver for the done channel,
		// or else this will hang everything!
		defer func() {
			_, ok := w.(http.CloseNotifier)
			if ok {
				// Only do this is the select case below is sure to be running!
				done <- true
			}
		}()

		// Set up a background notifier
		go func() {
			wCloseNotify, ok := w.(http.CloseNotifier)
			if !ok {
				//log.Error("ResponseWriter has no CloseNotify()!")
				return
			}
			for {
				select {
				case <-wCloseNotify.CloseNotify():
					// Client is done
					log.Warn("Connection to client closed")
				case <-done:
					// We are done
					return
				}
			}
		}() // Call the goroutine
	}

	// Export functions to the Lua state
	// Flush can be an uninitialized channel, it is handled in the function.
	exportCommonFunctions(w, req, filename, perm, L, luapool, flushFunc, cache)

	// Run the script
	if err := L.DoFile(filename); err != nil {
		// Logging and/or HTTP response is handled elsewhere
		return err
	}

	return nil
}

// Run a Lua string as a HTTP handler. Also has access to the userstate and permissions.
// Returns an error if there was a problem with running the lua script, otherwise nil.
func runLuaString(w http.ResponseWriter, req *http.Request, script string, perm pinterface.IPermissions, luapool *lStatePool, cache *FileCache) error {

	// Retrieve a Lua state
	L := luapool.Get()

	// Give no filename (an empty string will be handled correctly by the function).
	// Nil is the channel for sending flush requests. nil is checked for in the function.
	exportCommonFunctions(w, req, "", perm, L, luapool, nil, cache)

	// Run the script
	if err := L.DoString(script); err != nil {
		// Close the Lua state
		L.Close()

		// Logging and/or HTTP response is handled elsewhere
		return err
	}

	// Only put the Lua state back if there were no errors
	luapool.Put(L)

	return nil
}

// Run a Lua file as a configuration script. Also has access to the userstate and permissions.
// Returns an error if there was a problem with running the lua script, otherwise nil.
func runConfiguration(filename string, perm pinterface.IPermissions, luapool *lStatePool, cache *FileCache, mux *http.ServeMux, singleFileMode bool) error {

	// Retrieve a Lua state
	L := luapool.Get()

	// Retrieve the userstate
	userstate := perm.UserState()

	// Server configuration functions
	exportServerConfigFunctions(L, perm, filename, luapool)

	// Other basic system functions, like log()
	exportBasicSystemFunctions(L)

	// Simpleredis data structures (could be used for storing server stats)
	exportList(L, userstate)
	exportSet(L, userstate)
	exportHash(L, userstate)
	exportKeyValue(L, userstate)

	// For handling JSON data
	exportJSONFunctions(L)
	exportJFile(L, filepath.Dir(filename))
	exportJNode(L)

	// For saving and loading Lua functions
	exportCodeLibrary(L, userstate)

	// Plugins
	exportPluginFunctions(L, nil)

	// Cache
	exportCacheFunctions(L, cache)

	if singleFileMode {
		// Lua HTTP handlers
		exportLuaHandlerFunctions(L, filename, perm, luapool, cache, mux, false)
	}

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

/*
 * Return the functions available in the given Lua code as
 * functions in a map that can be used by templates.
 *
 * Note that the lua functions must only accept and return strings
 * and that only the first returned value will be accessible.
 * The Lua functions may take an optional number of arguments.
 */
func luaFunctionMap(w http.ResponseWriter, req *http.Request, luadata []byte, filename string, perm pinterface.IPermissions, luapool *lStatePool, cache *FileCache) (template.FuncMap, error) {

	// Retrieve a Lua state
	L := luapool.Get()
	defer luapool.Put(L)

	// Prepare an empty map of functions (and variables)
	funcs := make(template.FuncMap)

	// Give no filename (an empty string will be handled correctly by the function).
	exportCommonFunctions(w, req, filename, perm, L, luapool, nil, cache)

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
			mapinterface, _ := table2map(luaTable)
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
				funcs[functionName] = func(args ...string) (interface{}, error) {

					// Create a brand new Lua state
					L2 := luapool.New()
					defer L2.Close()

					// Set up a new Lua state with the current http.ResponseWriter and *http.Request
					exportCommonFunctions(w, req, filename, perm, L2, luapool, nil, cache)

					// Push the Lua function to run
					L2.Push(luaFunc)

					// Push the given arguments
					for _, arg := range args {
						L2.Push(lua.LString(arg))
					}

					// Run the Lua function
					if err := L2.PCall(len(args), lua.MultRet, nil); err != nil {
						// If calling the function did not work out, return the infostring and error
						return infostring(functionName, args), err
					}

					// Empty return value if no values were returned
					var retval interface{}

					// Return the first of the returned arguments, as a string
					if L2.GetTop() >= 1 {
						lv := L2.Get(-1)
						if tbl, ok := lv.(*lua.LTable); ok {
							// lv was a Lua Table
							retval = table2interfacemap(tbl)
							if debugMode && verboseMode {
								log.Info(infostring(functionName, args) + " -> (map)")
							}
						} else if lv.Type() == lua.LTString {
							// lv is a Lua String
							retstr := L2.ToString(1)
							retval = retstr
							if debugMode && verboseMode {
								log.Info(infostring(functionName, args) + " -> \"" + retstr + "\"")
							}
						} else {
							retval = ""
							log.Warn("The return type of " + infostring(functionName, args) + " can't be converted")
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
