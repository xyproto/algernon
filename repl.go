package main

import (
	"fmt"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/xyproto/pinterface"
	"github.com/xyproto/term"
	"github.com/yuin/gopher-lua"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	helpText = `Available functions:

Data structures

// Get or create Redis-backed Set (takes a name, returns a set object)
Set(string) -> userdata
// Add an element to the set
set:add(string)
// Remove an element from the set
set:del(string)
// Check if a set contains a value. Returns true only if the value exists and there were no errors.
set:has(string) -> bool
// Get all members of the set
set:getall() -> table
// Remove the set itself. Returns true if successful.
set:remove() -> bool
// Clear the set. Returns true if successful.
set:clear() -> bool

// Get or create a Redis-backed List (takes a name, returns a list object)
List(string) -> userdata
// Add an element to the list
list:add(string)
// Get all members of the list
list:getall() -> table
// Get the last element of the list. The returned value can be empty
list:getlast() -> string
// Get the N last elements of the list
list:getlastn(number) -> table
// Remove the list itself. Returns true if successful.
list:remove() -> bool
// Clear the list. Returns true if successful.
list:clear() -> bool

// Get or create a Redis-backed HashMap (takes a name, returns a hash map object)
HashMap(string) -> userdata
// For a given element id (for instance a user id), set a key. Returns true if successful.
hash:set(string, string, string) -> bool
// For a given element id (for instance a user id), and a key, return a value.
hash:get(string, string) -> string
// For a given element id (for instance a user id), and a key, check if the key exists in the hash map.
hash:has(string, string) -> bool
// For a given element id (for instance a user id), check if it exists.
hash:exists(string) -> bool
// Get all keys of the hash map
hash:getall() -> table
// Remove a key for an entry in a hash map. Returns true if successful
hash:delkey(string, string) -> bool
// Remove an element (for instance a user). Returns true if successful
hash:del(string) -> bool
// Remove the hash map itself. Returns true if successful.
hash:remove() -> bool
// Clear the hash map. Returns true if successful.
hash:clear() -> bool

// Get or create a Redis-backed KeyValue collection (takes a name, returns a key/value object)
KeyValue(string) -> userdata
// Set a key and value. Returns true if successful.
kv:set(string, string) -> bool
// Takes a key, returns a value. Returns an empty string if the function fails.
kv:get(string) -> string
// Takes a key, returns the value+1. Creates a key/value and returns "1" if it did not already exist.
kv:inc(string) -> string
// Remove a key. Returns true if successful.
kv:del(string) -> bool
// Remove the KeyValue itself. Returns true if successful.
kv:remove() -> bool
// Clear the KeyValue. Returns true if successful.
kv:clear() -> bool

Live server configuration

// Reset the URL prefixes and make everything *public*.
ClearPermissions()
// Add an URL prefix that will have *admin* rights.
AddAdminPrefix(string)
// Add an URL prefix that will have *user* rights.
AddUserPrefix(string)
// Provide a lua function that will be used as the permission denied handler.
DenyHandler(function)
// Log to the given filename. If the filename is an empty string, log to stderr. Returns true if successful.
LogTo(string) -> bool

Output

log(...) // Log the given strings as info. Takes a variable number of strings.
warn(...) // Log the given strings as a warning. Takes a variable number of strings.
error(...) // Log the given strings as an error. Takes a variable number of strings.
print(...) // Output text. Takes a variable number of strings.

Cache

CacheStats() -> string // Return stats about the file cache
ClearCache() // Clear the file cache

JSON

JFile(filename) -> userdata // Use, or create, a JSON document/file.
jfile:get(string) -> string // Retrieve a string, given a valid JSON path.
jfile:set(string, string) -> bool // Changes an entrie given a JSON path and a value.
jfile:add([string, ]string) -> bool // Given a JSON path (optional) and JSON data, add it to a JSON list.
jfile:delkey(string) -> bool // Removes a key in a map in a JSON document.
ToJSON(table) -> string // Convert a Lua table with strings or ints to JSON.


Plugins

Plugin(string) -> bool // Load a plugin given the path to an executable. Returns true if successful. Will return the plugin help text if called on the Lua prompt.
PluginCode(string) -> string // Returns the Lua code as returned by the Lua.Code function in the plugin, given a plugin path. May return an empty string.
CallPlugin(string, string, ...) -> string // Takes a plugin path, function name and arguments. Returns an empty string if the function call fails, or the results as a JSON string if successful.

Code libraries

CodeLib() -> userdata // Creates a code library object.
codelib:add(string, string) -> bool // Given a namespace and Lua code, add the given code to the namespace. Returns true if successful.
codelib:set(string, string) -> bool // Given a namespace and Lua code, set the given code as the only code in the namespace. Returns true if successful.
codelib:get(string) -> string // Given a namespace, return Lua code or an empty string.
codelib:import(string) -> bool // Import (eval) code from the given namespace into the current Lua state. Returns true on success.
codelib:clear() -> bool // Completely clear the code library. Returns true of successful.

Various

// Return a string with various server information
ServerInfo() -> string
// Return the version string for the server
version() -> string
// Tries to extract and print the contents of a Lua value
pprint(value)
// Sleep the given number of seconds (can be a float)
sleep(number)
`
	exitMessage = "bye!"
)

// Attempt to output a more informative text than the memory location
func pprint(value lua.LValue) {
	switch v := value.(type) {
	case *lua.LTable:
		mapinterface, multiple := table2map(v)
		if multiple {
			fmt.Println(v)
		}
		// TODO: Use reflect
		switch m := mapinterface.(type) {
		case map[string]string:
			fmt.Printf("%#v\n", map[string]string(m))
		case map[string]int:
			fmt.Printf("%#v\n", map[string]int(m))
		case map[int]string:
			fmt.Printf("%#v\n", map[int]string(m))
		case map[int]int:
			fmt.Printf("%#v\n", map[int]int(m))
		default:
			fmt.Println(v)
		}
	case *lua.LFunction:
		if v.Proto != nil {
			// Extended information about the function
			fmt.Println(v.Proto)
		} else {
			fmt.Println(v)
		}
	default:
		fmt.Println(v)
	}
}

// Export Lua functions related to the REPL
func exportREPL(L *lua.LState) {

	// Attempt to return a more informative text than the memory location
	L.SetGlobal("pprint", L.NewFunction(func(L *lua.LState) int {
		pprint(L.Get(1))
		return 0 // number of results
	}))

}

// Split the given line in three parts, and color the parts
func colorSplit(line, sep string, colorFunc1, colorFuncSep, colorFunc2 func(string) string, reverse bool) (string, string) {
	if strings.Contains(line, sep) {
		fields := strings.SplitN(line, sep, 2)
		s1 := ""
		if colorFunc1 != nil {
			s1 += colorFunc1(fields[0])
		} else {
			s1 += fields[0]
		}
		s2 := ""
		if colorFunc2 != nil {
			s2 += colorFuncSep(sep) + colorFunc2(fields[1])
		} else {
			s2 += sep + fields[1]
		}
		return s1, s2
	}
	if reverse {
		return "", line
	}
	return line, ""
}

// Syntax highlight the given line
func highlight(o *term.TextOutput, line string) string {
	unprocessed := line
	unprocessed, comment := colorSplit(unprocessed, "//", nil, o.DarkGray, o.DarkGray, false)
	module, unprocessed := colorSplit(unprocessed, ":", o.LightGreen, o.DarkRed, nil, true)
	function := ""
	if unprocessed != "" {
		// Green function names
		if strings.Contains(unprocessed, "(") {
			fields := strings.SplitN(unprocessed, "(", 2)
			function = o.LightGreen(fields[0])
			unprocessed = "(" + fields[1]
		}
	}
	unprocessed, typed := colorSplit(unprocessed, "->", nil, o.LightBlue, o.DarkRed, false)
	unprocessed = strings.Replace(unprocessed, "string", o.LightBlue("string"), -1)
	unprocessed = strings.Replace(unprocessed, "number", o.LightYellow("number"), -1)
	unprocessed = strings.Replace(unprocessed, "function", o.LightCyan("function"), -1)
	return module + function + unprocessed + typed + comment
}

func mustSaveHistory(o *term.TextOutput, historyFilename string) {
	fmt.Printf(o.LightBlue("Saving history to %s... "), historyFilename)
	if err := saveHistory(historyFilename); err != nil {
		fmt.Println(o.DarkRed("failed: " + err.Error()))
	} else {
		fmt.Println(o.LightGreen("ok"))
	}
}

// REPL provides a "Read Eveal Print" loop for interacting with Lua.
// A variatey of functions are exposed to the Lua state.
func REPL(perm pinterface.IPermissions, luapool *lStatePool, cache *FileCache) error {
	var (
		historyFilename string
		err             error
	)

	historydir, err := homedir.Dir()
	if err != nil {
		log.Error("Could not find a user directory to store the REPL history.")
		historydir = "."
	}
	if runtime.GOOS == "windows" {
		historyFilename = filepath.Join(historydir, "algernon", "repl.txt")
	} else {
		historyFilename = filepath.Join(historydir, ".algernon_history")
	}

	// Retrieve the userstate
	userstate := perm.UserState()

	// Retrieve a Lua state
	L := luapool.Get()
	// Don't re-use the Lua state
	defer L.Close()

	// Server configuration functions
	exportServerConfigFunctions(L, perm, "", luapool)

	// Other basic system functions, like log()
	exportBasicSystemFunctions(L)

	// Simpleredis data structures
	exportList(L, userstate)
	exportSet(L, userstate)
	exportHash(L, userstate)
	exportKeyValue(L, userstate)

	// For handling JSON data
	exportJSONFunctions(L)
	exportJFile(L)

	// For saving and loading Lua functions
	exportCodeLibrary(L, userstate)

	// Pretty printing
	exportREPL(L)

	// Colors and input
	enableColors := runtime.GOOS != "windows"
	o := term.NewTextOutput(enableColors, true)

	// Plugin functionality
	exportPluginFunctions(L, o)

	// Cache
	exportCacheFunctions(L, cache)

	o.Println(o.LightBlue(versionString))
	o.Println(o.LightGreen("Ready"))

	// Add a newline after the prompt to prepare for logging, if in verbose mode
	go func() {
		if verboseMode {
			// TODO Consider using a channel instead of sleep for outputting the cosmetic newline...
			time.Sleep(200 * time.Millisecond)
			fmt.Println()
		}
	}()

	// Start the read, eval, print loop
	var (
		line     string
		prompt   = o.LightGreen("lua> ")
		EOF      bool
		EOFcount int
	)
	if loadHistory(historyFilename) != nil && exists(historyFilename) {
		log.Error("Could not load REPL history:", historyFilename)
	}
	for {
		// Retrieve user input
		EOF = false
		if line, err = getInput(prompt); err != nil {
			if err.Error() == "EOF" {
				if debugMode {
					o.Println(o.LightPurple(err.Error()))
				}
				EOF = true
			} else {
				log.Error("Error reading line(" + err.Error() + ").")
				continue
			}
		} else {
			addHistory(line)
		}

		if EOF {
			switch EOFcount {
			case 0:
				o.Err("Press Ctrl-d again to quit.")
				EOFcount++
				continue
			default:
				mustSaveHistory(o, historyFilename)
				o.Println(o.LightGreen(exitMessage))
				os.Exit(0)
			}
		}

		line = strings.TrimSpace(line)

		switch line {
		case "help":
			for _, line := range strings.Split(helpText, "\n") {
				o.Println(highlight(o, line))
			}
			continue
		case "quit", "exit", "shutdown":
			mustSaveHistory(o, historyFilename)
			o.Println(o.LightGreen(exitMessage))
			os.Exit(0)
		case "zalgo":
			// Easter egg
			o.ErrExit("Ḫ̷̲̫̰̯̭̀̂̑̈ͅĚ̥̖̩̘̱͔͈͈ͬ̚ ̦̦͖̲̀ͦ͂C̜͓̲̹͐̔ͭ̏Oͭ͛͂̋ͭͬͬ͆͏̺͓̰͚͠ͅM̢͉̼̖͍̊̕Ḛ̭̭͗̉̀̆ͬ̐ͪ̒S͉̪͂͌̄")
		}

		// If the line starts with print, don't touch it
		if strings.HasPrefix(line, "print(") {
			if err = L.DoString(line); err != nil {
				// Output the error message
				o.Err(err.Error())
			}
		} else {
			// Wrap the line in "pprint"
			if err = L.DoString("pprint(" + line + ")"); err != nil {
				// If there was a syntax error, try again without pprint
				if strings.Contains(err.Error(), "syntax error") {
					if err = L.DoString(line); err != nil {
						// Output the error message
						o.Err(err.Error())
					}
					// For other kinds of errors, output the error
				} else {
					// Output the error message
					o.Err(err.Error())
				}
			}
		}
	}
}
