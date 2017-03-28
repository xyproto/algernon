package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/xyproto/jpath"
	"github.com/yuin/gluamapper"
	"github.com/yuin/gopher-lua"
)

var (
	errToMap = errors.New("Could not represent Lua structure table as a map")
)

// Output more informative information than the memory location.
// Attempt to extract and print the values of the given lua.LValue.
// Does not add a newline at the end.
func pprintToWriter(w io.Writer, value lua.LValue) {
	switch v := value.(type) {
	case *lua.LTable:
		m, isAnArray, err := table2mapinterface(v)
		if err != nil {
			//log.Info("try: for k,v in pairs(t) do pprint(k,v) end")
			// Could not convert to a map
			fmt.Fprint(w, v)
			return
		}
		if isAnArray {
			// A map which is really an array (arrays in Lua are maps)
			var buf bytes.Buffer
			buf.WriteString("{")
			// Order the map
			length := len(m)
			for i := 1; i <= length; i++ {
				val := m[float64(i)] // gluamapper uses float64 for all numbers
				buf.WriteString(fmt.Sprintf("%#v", val))
				if i != length {
					// Output a comma for every element except the last one
					buf.WriteString(", ")
				}
			}
			buf.WriteString("}")
			buf.WriteTo(w)
			return
		}
		// A go map
		fmt.Fprint(w, fmt.Sprintf("%#v", m)[29:])
	case *lua.LFunction:
		if v.Proto != nil {
			// Extended information about the function
			fmt.Fprint(w, v.Proto)
		} else {
			fmt.Fprint(w, v)
		}
	case *lua.LUserData:
		if jfile, ok := v.Value.(*jpath.JFile); ok {
			fmt.Fprintln(w, v)
			fmt.Fprintf(w, "filename: %s\n", jfile.GetFilename())
			if data, err := jfile.JSON(); err == nil { // success
				fmt.Fprintf(w, "JSON data:\n%s", string(data))
			}
		} else {
			fmt.Fprint(w, v)
		}
	default:
		fmt.Fprint(w, v)
	}
}

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
// If several different types are found, the returned bool is true.
// TODO: Look into that gopher-lua module that are made for converting to and from Lua tables. It will be cleaner.
func table2map(luaTable *lua.LTable, preferInt bool) (interface{}, bool) {

	mapSS, mapSI, mapIS, mapII := table2maps(luaTable)

	lss := len(mapSS)
	lsi := len(mapSI)
	lis := len(mapIS)
	lii := len(mapII)

	total := lss + lsi + lis + lii

	// Return the first map that has values
	if !preferInt {
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
	} else {
		if lii > 0 {
			//log.Println(key, "INT -> INT map")
			return interface{}(mapII), lii < total
		} else if lis > 0 {
			//log.Println(key, "INT -> STRING map")
			return interface{}(mapIS), lis < total
		} else if lsi > 0 {
			//log.Println(key, "STRING -> INT map")
			return interface{}(mapSI), lsi < total
		} else if lss > 0 {
			//log.Println(key, "STRING -> STRING map")
			return interface{}(mapSS), lss < total
		}
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

// Convert a Lua table to a map by using gluamapper.
// If the map really is an array (all the keys are indices), return true.
func table2mapinterface(luaTable *lua.LTable) (retmap map[interface{}]interface{}, isArray bool, err error) {
	var (
		m         = make(map[interface{}]interface{})
		opt       = gluamapper.Option{}
		indices   []uint64
		i, length uint64
	)
	// Catch a problem that may occur when converting the map value with gluamapper.ToGoValue
	defer func() {
		if r := recover(); r != nil {
			retmap = m
			err = errToMap // Could not represent Lua structure table as a map
			return
		}
	}()
	luaTable.ForEach(func(tkey, tvalue lua.LValue) {
		if i, isNum := tkey.(lua.LNumber); isNum {
			indices = append(indices, uint64(i))
		}
		// If tkey or tvalue is an LTable, give up
		m[gluamapper.ToGoValue(tkey, opt)] = gluamapper.ToGoValue(tvalue, opt)
		length++
	})
	// Report back as a map, not an array, if there are no elements
	if length == 0 {
		return m, false, nil
	}
	// Loop through every index that must be present in an array
	isAnArray := true
	for i = 1; i <= length; i++ {
		// The map must have this index in order to be an array
		hasIt := false
		for _, val := range indices {
			if val == i {
				hasIt = true
				break
			}
		}
		if !hasIt {
			isAnArray = false
			break
		}
	}
	return m, isAnArray, nil
}

// Return a *lua.LState object that contains several exposed functions
func (ac *algernonConfig) exportCommonFunctions(w http.ResponseWriter, req *http.Request, filename string, L *lua.LState, flushFunc func(), httpStatus *FutureStatus) {

	// Make basic functions, like print, available to the Lua script.
	// Only exports functions that can relate to HTTP responses or requests.
	ac.exportBasicWeb(w, req, L, filename, flushFunc, httpStatus)

	// Make other basic functions available
	exportBasicSystemFunctions(L)

	// Functions for rendering markdown or amber
	ac.exportRenderFunctions(w, req, L)

	// If there is a database backend
	if ac.perm != nil {

		// Retrieve the userstate
		userstate := ac.perm.UserState()

		// Functions for serving files in the same directory as a script
		ac.exportServeFile(w, req, L, filename)

		// Make the functions related to userstate available to the Lua script
		exportUserstate(w, req, L, userstate)

		// Simpleredis data structures
		exportList(L, userstate)
		exportSet(L, userstate)
		exportHash(L, userstate)
		exportKeyValue(L, userstate)

		// For saving and loading Lua functions
		exportCodeLibrary(L, userstate)

	}

	// For handling JSON data
	exportJSONFunctions(L)
	ac.exportJFile(L, filepath.Dir(filename))
	exportJNode(L)

	// Extras
	exportExtras(L)

	// pprint
	//exportREPL(L)

	// Plugins
	ac.exportPluginFunctions(L, nil)

	// Cache
	ac.exportCacheFunctions(L)

	// File uploads
	exportUploadedFile(L, w, req, filepath.Dir(filename))
}

// Run a Lua file as a HTTP handler. Also has access to the userstate and permissions.
// Returns an error if there was a problem with running the lua script, otherwise nil.
// Also returns a header map
func (ac *algernonConfig) runLua(w http.ResponseWriter, req *http.Request, filename string, flushFunc func(), fust *FutureStatus) error {

	// Retrieve a Lua state
	L := ac.luapool.Get()
	defer ac.luapool.Put(L)

	// Warn if the connection is closed before the script has finished.
	// Requires that the requestWriter has CloseNotify.
	if ac.verboseMode {

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
	ac.exportCommonFunctions(w, req, filename, L, flushFunc, fust)

	// Run the script and return the error value.
	// Logging and/or HTTP response is handled elsewhere.
	return L.DoFile(filename)
}

// Run a Lua file as a configuration script. Also has access to the userstate and permissions.
// Returns an error if there was a problem with running the lua script, otherwise nil.
// perm can be nil, but then several Lua functions will not be exposed
func (ac *algernonConfig) runConfiguration(filename string, mux *http.ServeMux, singleFileMode bool) error {

	// Retrieve a Lua state
	L := ac.luapool.Get()

	// Basic system functions, like log()
	exportBasicSystemFunctions(L)

	// If there is a database backend
	if ac.perm != nil {

		// Retrieve the userstate
		userstate := ac.perm.UserState()

		// Server configuration functions
		ac.exportServerConfigFunctions(L, filename)

		// Simpleredis data structures (could be used for storing server stats)
		exportList(L, userstate)
		exportSet(L, userstate)
		exportHash(L, userstate)
		exportKeyValue(L, userstate)

		// For saving and loading Lua functions
		exportCodeLibrary(L, userstate)
	}

	// For handling JSON data
	exportJSONFunctions(L)
	ac.exportJFile(L, filepath.Dir(filename))
	exportJNode(L)

	// Extras
	exportExtras(L)

	// Plugins
	ac.exportPluginFunctions(L, nil)

	// Cache
	ac.exportCacheFunctions(L)

	if singleFileMode {
		// Lua HTTP handlers
		ac.exportLuaHandlerFunctions(L, filename, mux, false, nil, ac.defaultTheme)
	}

	// Run the script
	if err := L.DoFile(filename); err != nil {
		// Close the Lua state
		L.Close()

		// Logging and/or HTTP response is handled elsewhere
		return err
	}

	// Only put the Lua state back if there were no errors
	ac.luapool.Put(L)

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
func (ac *algernonConfig) luaFunctionMap(w http.ResponseWriter, req *http.Request, luadata []byte, filename string) (template.FuncMap, error) {
	ac.pongomutex.Lock()
	defer ac.pongomutex.Unlock()

	// Retrieve a Lua state
	L := ac.luapool.Get()
	defer ac.luapool.Put(L)

	// Prepare an empty map of functions (and variables)
	funcs := make(template.FuncMap)

	// Give no filename (an empty string will be handled correctly by the function).
	ac.exportCommonFunctions(w, req, filename, L, nil, nil)

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
			mapinterface, _ := table2map(luaTable, false)
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
					L2 := ac.luapool.New()
					defer L2.Close()

					// Set up a new Lua state with the current http.ResponseWriter and *http.Request
					ac.exportCommonFunctions(w, req, filename, L2, nil, nil)

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
							if ac.debugMode && ac.verboseMode {
								log.Info(infostring(functionName, args) + " -> (map)")
							}
						} else if lv.Type() == lua.LTString {
							// lv is a Lua String
							retstr := L2.ToString(1)
							retval = retstr
							if ac.debugMode && ac.verboseMode {
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
