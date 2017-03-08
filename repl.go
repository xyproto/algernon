package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/chzyer/readline"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/xyproto/pinterface"
	"github.com/xyproto/term"
	"github.com/yuin/gopher-lua"
)

const (
	generalHelpText = `Available functions:

Data structures

// Get or create database-backed Set (takes a name, returns a set object)
Set(string) -> userdata
// Add an element to the set
set:add(string)
// Remove an element from the set
set:del(string)
// Check if a set contains a value.
// Returns true only if the value exists and there were no errors.
set:has(string) -> bool
// Get all members of the set
set:getall() -> table
// Remove the set itself. Returns true if successful.
set:remove() -> bool
// Clear the set. Returns true if successful.
set:clear() -> bool

// Get or create a database-backed List (takes a name, returns a list object)
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

// Get or create a database-backed HashMap
// (takes a name, returns a hash map object)
HashMap(string) -> userdata
// For a given element id (for instance a user id), set a key.
// Returns true if successful.
hash:set(string, string, string) -> bool
// For a given element id (for instance a user id), and a key, return a value.
hash:get(string, string) -> string
// For a given element id (for instance a user id), and a key,
// check if the key exists in the hash map.
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

// Get or create a database-backed KeyValue collection
// (takes a name, returns a key/value object)
KeyValue(string) -> userdata
// Set a key and value. Returns true if successful.
kv:set(string, string) -> bool
// Takes a key, returns a value. May return an empty string.
kv:get(string) -> string
// Takes a key, returns the value+1.
// Creates a key/value and returns "1" if it did not already exist.
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
// Direct the logging to the given filename. If the filename is an empty
// string, direct logging to stderr. Returns true if successful.
LogTo(string) -> bool

Output

// Log the given strings as info. Takes a variable number of strings.
log(...)
// Log the given strings as a warning. Takes a variable number of strings.
warn(...)
// Log the given strings as an error. Takes a variable number of strings.
err(...)
// Output text. Takes a variable number of strings.
print(...)
// Output rendered HTML given Markdown. Takes a variable number of strings.
mprint(...)
// Output rendered HTML given Amber. Takes a variable number of strings.
aprint(...)
// Output rendered CSS given GCSS. Takes a variable number of strings.
gprint(...)
// Output rendered JavaScript given JSX. Takes a variable number of strings.
jprint(...)
// Output (mainly) rendered HTML given Pongo2. Takes a variable number of strings.
poprint(...)

Cache

CacheInfo() -> string // Return information about the file cache.
ClearCache() // Clear the file cache.

JSON

// Use, or create, a JSON document/file.
JFile(filename) -> userdata
// Retrieve a string, given a valid JSON path. May return an empty string.
jfile:getstring(string) -> string
// Retrieve a JSON node, given a valid JSON path. May return nil.
jfile:getnode(string) -> userdata
// Retrieve a value, given a valid JSON path. May return nil.
jfile:get(string) -> value
// Change an entry given a JSON path and a value. Returns true if successful.
jfile:set(string, string) -> bool
// Given a JSON path (optional) and JSON data, add it to a JSON list.
// Returns true if successful.
jfile:add([string, ]string) -> bool
// Removes a key in a map in a JSON document. Returns true if successful.
jfile:delkey(string) -> bool
// Convert a Lua table with strings or ints to JSON.
// Takes an optional number of spaces to indent the JSON data.
JSON(table[, number]) -> string
// Create a JSON document node.
JNode() -> userdata
// Add JSON data to a node. The first argument is an optional JSON path.
// The second argument is a JSON data string. Returns true on success.
// "x" is the default JSON path.
jnode:add([string, ]string) ->
// Given a JSON path, retrieves a JSON node.
jnode:get(string) -> userdata
// Given a JSON path, retrieves a JSON string.
jnode:getstring(string) -> string
// Given a JSON path and a JSON string, set the value.
jnode:set(string, string)
// Given a JSON path, remove a key from a map.
jnode:delkey(string) -> bool
// Return the JSON data, nicely formatted.
jnode:pretty() -> string
// Return the JSON data, as a compact string.
jnode:compact() -> string
// Sends JSON data to the given URL. Returns the HTTP status code as a string.
// The content type is set to "application/json; charset=utf-8".
// The second argument is an optional authentication token that is used for the
// Authorization header field. Uses HTTP POST.
jnode:POST(string[, string]) -> string
// Sends JSON data to the given URL. Returns the HTTP status code as a string.
// The content type is set to "application/json; charset=utf-8".
// The second argument is an optional authentication token that is used for the
// Authorization header field. Uses HTTP PUT.
jnode:PUT(string[, string]) -> string
// Alias for jnode:POST
jnode:send(string[, string]) -> string
// Fetches JSON over HTTP given an URL that starts with http or https.
// The JSON data is placed in the JNode. Returns the HTTP status code as a string.
jnode:GET(string) -> string
// Alias for jnode:GET
jnode:receive(string) -> string

Plugins

// Load a plugin given the path to an executable. Returns true if successful.
// Will return the plugin help text if called on the Lua prompt.
Plugin(string) -> bool
// Returns the Lua code as returned by the Lua.Code function in the plugin,
// given a plugin path. May return an empty string.
PluginCode(string) -> string
// Takes a plugin path, function name and arguments. Returns an empty string
// if the function call fails, or the results as a JSON string if successful.
CallPlugin(string, string, ...) -> string

Code libraries

// Create or use a code library object. Takes an optional data structure name.
CodeLib([string]) -> userdata
// Given a namespace and Lua code, add the given code to the namespace.
// Returns true if successful.
codelib:add(string, string) -> bool
// Given a namespace and Lua code, set the given code as the only code
// in the namespace. Returns true if successful.
codelib:set(string, string) -> bool
// Given a namespace, return Lua code, or an empty string.
codelib:get(string) -> string
// Import (eval) code from the given namespace into the current Lua state.
// Returns true if successful.
codelib:import(string) -> bool
// Completely clear the code library. Returns true if successful.
codelib:clear() -> bool

Various

// Return a string with various server information
ServerInfo() -> string
// Return the version string for the server
version() -> string
// Tries to extract and print the contents of the given Lua values
pprint(...)
// Sleep the given number of seconds (can be a float)
sleep(number)
// Return the number of nanoseconds from 1970 ("Unix time")
unixnano() -> number
// Convert Markdown to HTML
markdown(string) -> string

Extra

// Takes a Python filename, executes the script with the "python" binary in the Path.
// Returns the output as a Lua table, where each line is an entry.
py(string) -> table
// Takes one or more system commands (possibly separated by ";") and runs them.
// Returns the output lines as a table.
run(string) -> table
// Lists the keys and values of a Lua table. Returns a string.
// Lists the contents of the global namespace "_G" if no arguments are given.
dir([table]) -> string
`
	usageMessage = `
Type "webhelp" for an overview of functions that are available when
handling requests. Or "confighelp" for an overview of functions that are
available when configuring an Algernon application.
`
	webHelpText = `Available functions:

Handling users and permissions

// Check if the current user has "user" rights
UserRights() -> bool
// Check if the given username exists
HasUser(string) -> bool
// Get the value from the given boolean field
// Takes a username and field name
BooleanField(string, string) -> bool
// Save a value as a boolean field
// Takes a username, field name and boolean value
SetBooleanField(string, string, bool)
// Check if a given username is confirmed
IsConfirmed(string) -> bool
// Check if a given username is logged in
IsLoggedIn(string) -> bool
// Check if the current user has "admin rights"
AdminRights() -> bool
// Check if a given username is an admin
IsAdmin(string) -> bool
// Get the username stored in a cookie, or an empty string
UsernameCookie() -> string
// Store the username in a cookie, returns true if successful
SetUsernameCookie(string) -> bool
// Clear the login cookie
ClearCookie()
// Get a table containing all usernames
AllUsernames() -> table
// Get the email for a given username, or an empty string
Email(string) -> string
// Get the password hash for a given username, or an empty string
PasswordHash(string) -> string
// Get all unconfirmed usernames
AllUnconfirmedUsernames() -> table
// Get a confirmation code that can be given to a user,
// or an empty string. Takes a username.
ConfirmationCode(string) -> string
// Add a user to the list of unconfirmed users.
// Takes a username and a confirmation code.
AddUnconfirmed(string, string)
// Remove a user from the list of unconfirmed users. Takes a username.
RemoveUnconfirmed(string)
// Mark a user as confirmed. Takes a username.
MarkConfirmed(string)
// Removes a user. Takes a username.
RemoveUser(string)
// Make a user an admin. Takes a username.
SetAdminStatus(string)
// Make an admin user a regular user. Takes a username.
RemoveAdminStatus(string)
// Add a user. Takes a username, password and email.
AddUser(string, string, string)
// Set a user as logged in on the server (not cookie). Takes a username.
SetLoggedIn(string)
// Set a user as logged out on the server (not cookie). Takes a username.
SetLoggedOut(string)
// Log in a user, both on the server and with a cookie. Takes a username.
Login(string)
// Log in a user, both on the server and with a cookie.
// Takes a username. Returns true if the cookie was set successfully.
CookieLogin(string) -> bool
// Log out a user, on the server (which is enough). Takes a username.
Logout(string)
// Get the current username, from the cookie
Username() -> string
// Get the current cookie timeout. Takes a username.
CookieTimeout(string) -> number
// Set the current cookie timeout. Takes a timeout, in seconds.
SetCookieTimeout(number)
// Get the current password hashing algorithm (bcrypt, bcrypt+ or sha256)
PasswordAlgo() -> string
// Set the current password hashing algorithm (bcrypt, bcrypt+ or sha256)
// Takes a string
SetPasswordAlgo(string)
// Hash the password
// Takes a username and password (username can be used for salting)
HashPassword(string, string) -> string
// Change the password for a user, given a username and a new password
SetPassword(string, string)
// Check if a given username and password is correct
// Takes a username and password
CorrectPassword(string, string) -> bool
// Checks if a confirmation code is already in use
// Takes a confirmation code
AlreadyHasConfirmationCode(string) -> bool
// Find a username based on a given confirmation code,
// or returns an empty string. Takes a confirmation code
FindUserByConfirmationCode(string) -> string
// Mark a user as confirmed
// Takes a username
Confirm(string)
// Mark a user as confirmed, returns true if successful
// Takes a confirmation code
ConfirmUserByConfirmationCode(string) -> bool
// Set the minimum confirmation code length
// Takes the minimum number of characters
SetMinimumConfirmationCodeLength(number)
// Generates a unique confirmation code, or an empty string
GenerateUniqueConfirmationCode() -> string

File uploads

// Creates a file upload object. Takes a form ID (from a POST request) as the
// first parameter. Takes an optional maximum upload size (in MiB) as the
// second parameter. Returns nil and an error string on failure, or userdata
// and an empty string on success.
UploadedFile(string[, number]) -> userdata, string
// Return the uploaded filename, as specified by the client
uploadedfile:filename() -> string
// Return the size of the data that has been received
uploadedfile:size() -> number
// Return the mime type of the uploaded file, as specified by the client
uploadedfile:mimetype() -> string
// Save the uploaded data locally. Takes an optional filename.
uploadedfile:save([string]) -> bool
// Save the uploaded data as the client-provided filename, in the specified
// directory. Takes a relative or absolute path. Returns true on success.
uploadedfile:savein(string)  -> bool

Handling requests

// Set the Content-Type for a page.
content(string)
// Return the requested HTTP method (GET, POST etc).
method() -> string
// Output text to the browser/client. Takes a variable number of strings.
print(...)
// Return the requested URL path.
urlpath() -> string
// Return the HTTP header in the request, for a given key, or an empty string.
header(string) -> string
// Set an HTTP header given a key and a value.
setheader(string, string)
// Return the HTTP headers, as a table.
headers() -> table
// Return the HTTP body in the request
// (will only read the body once, since it's streamed).
body() -> string
// Set a HTTP status code (like 200 or 404).
// Must be used before other functions that writes to the client!
status(number)
// Set a HTTP status code and output a message (optional).
error(number[, string])
// Return the directory where the script is running. If a filename (optional)
// is given, then the path to where the script is running, joined with a path
// separator and the given filename, is returned.
scriptdir([string]) -> string
// Return the directory where the server is running. If a filename (optional)
// is given, then the path to where the server is running, joined with a path
// separator and the given filename, is returned.
serverdir([string]) -> string
// Serve a file that exists in the same directory as the script.
serve(string)
// Return a table with keys and values as given in a posted form, or as given
// in the URL ("/some/page?x=7" makes "x" with the value "7" available).
formdata() -> table
// Redirect to an absolute or relative URL. Also takes a HTTP status code.
redirect(string[, string])
// Permanently redirect to an absolute or relative URL. Uses status code 302.
permanent_redirect(string)
// Transmit what has been outputted so far, to the client.
flush()
`
	configHelpText = `Available functions:

Only available when used in serverconf.lua

// Set the default address for the server on the form [host][:port].
SetAddr(string)
// Reset the URL prefixes and make everything *public*.
ClearPermissions()
// Add an URL prefix that will have *admin* rights.
AddAdminPrefix(string)
// Add an URL prefix that will have *user* rights.
AddUserPrefix(string)
// Provide a lua function that will be used as the permission denied handler.
DenyHandler(function)
// Provide a lua function that will be run once,
// when the server is ready to start serving.
OnReady(function)
// Use a Lua file for setting up HTTP handlers instead of using the directory structure.
ServerFile(string) -> bool
`
	exitMessage = "bye"
)

// Export Lua functions related to the REPL
func exportREPL(L *lua.LState) {

	// Attempt to return a more informative text than the memory location.
	// Can take several arguments, just like print().
	L.SetGlobal("pprint", L.NewFunction(func(L *lua.LState) int {
		var buf bytes.Buffer
		top := L.GetTop()
		for i := 1; i <= top; i++ {
			pprintToWriter(&buf, L.Get(i))
			if i != top {
				buf.WriteString("\t")
			}
		}

		// Output the combined text
		fmt.Println(buf.String())

		return 0 // number of results
	}))

	// Get the current directory since this is probably in the REPL
	L.SetGlobal("scriptdir", L.NewFunction(func(L *lua.LState) int {
		scriptpath, err := os.Getwd()
		if err != nil {
			log.Error(err)
			L.Push(lua.LString("."))
			return 1 // number of results
		}
		top := L.GetTop()
		if top == 1 {
			// Also include a separator and a filename
			fn := L.ToString(1)
			scriptpath = filepath.Join(scriptpath, fn)
		}
		// Now have the correct absolute scriptpath
		L.Push(lua.LString(scriptpath))
		return 1 // number of results
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

// Output syntax highlighted help text, with an additional usage message
func outputHelp(o *term.TextOutput, helpText string) {
	for _, line := range strings.Split(helpText, "\n") {
		o.Println(highlight(o, line))
	}
	o.Println(usageMessage)
}

// Take all functions mentioned in the given help text string and add them to the readline completer
func addFunctionsFromHelptextToCompleter(helpText string, completer *readline.PrefixCompleter) {
	for _, line := range strings.Split(helpText, "\n") {
		if !strings.HasPrefix(line, "//") && strings.Contains(line, "(") {
			parts := strings.Split(line, "(")
			if strings.Contains(line, "()") {
				completer.Children = append(completer.Children, &readline.PrefixCompleter{Name: []rune(parts[0] + "()")})
			} else {
				completer.Children = append(completer.Children, &readline.PrefixCompleter{Name: []rune(parts[0] + "(")})
			}
		}
	}
}

// REPL provides a "Read Eval Print" loop for interacting with Lua.
// A variety of functions are exposed to the Lua state.
func REPL(perm pinterface.IPermissions, luapool *lStatePool, cache *FileCache, pongomutex *sync.RWMutex, ready, done chan bool) error {
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

	// Retrieve a Lua state
	L := luapool.Get()
	// Don't re-use the Lua state
	defer L.Close()

	// Server configuration functions
	exportServerConfigFunctions(L, perm, "", luapool, pongomutex)

	// Other basic system functions, like log()
	exportBasicSystemFunctions(L)

	// If there is a database backend
	if perm != nil {

		// Retrieve the userstate
		userstate := perm.UserState()

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
	exportJFile(L, serverDir)
	exportJNode(L)

	// Extras
	exportExtras(L)

	// Pretty printing
	exportREPL(L)

	// Colors and input
	enableColors := runtime.GOOS != "windows"
	o := term.NewTextOutput(enableColors, true)

	// Plugin functionality
	exportPluginFunctions(L, o)

	// Cache
	exportCacheFunctions(L, cache)

	// Getting ready
	o.Println(o.LightBlue(versionString))

	<-ready // Wait for the server to be ready

	// Tell the user that the server is ready
	o.Println(o.LightGreen("Ready"))

	// Start the read, eval, print loop
	var (
		line   string
		prompt = o.LightGreen("lua> ")
		EOF    bool
		//EOFcount int
	)

	// TODO: Automatically generate a list of all words that should be completed
	//       based on the documentation or repl help text. Then add each word
	//       to the completer.
	completer := readline.NewPrefixCompleter(
		&readline.PrefixCompleter{Name: []rune("help")},
		&readline.PrefixCompleter{Name: []rune("webhelp")},
		&readline.PrefixCompleter{Name: []rune("bye")},
		&readline.PrefixCompleter{Name: []rune("quit")},
		&readline.PrefixCompleter{Name: []rune("exit")},
		&readline.PrefixCompleter{Name: []rune("zalgo")},
	)

	addFunctionsFromHelptextToCompleter(generalHelpText, completer)

	l, err := readline.NewEx(&readline.Config{
		Prompt:            prompt,
		HistoryFile:       historyFilename,
		AutoComplete:      completer,
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
	})
	if err != nil {
		log.Error("Could not initiate github.com/chzyer/readline: " + err.Error())
	}

	// To be run at server shutdown
	atShutdown(func() {
		// Verbose mode has different log output at shutdown
		if !verboseMode {
			o.Println(o.LightBlue(exitMessage))
		}
	})
	for {
		// Retrieve user input
		EOF = false
		if line, err = l.Readline(); err != nil {
			if err == io.EOF {
				if debugMode {
					o.Println(o.LightPurple(err.Error()))
				}
				EOF = true
			} else if err == readline.ErrInterrupt {
				log.Warn("Interrupted")
				done <- true
				return nil
			} else {
				log.Error("Error reading line(" + err.Error() + ").")
				continue
			}
		} else {
			//addHistory(line)
			// Save the REPL history at every line.
			// This proved to be safer than only saving the history at shutdown
			// (due to how ctrl-c was handled on some systems)
			// and it's hard to imagine performance issues with this.
			//saveHistory(historyFilename)
		}

		if EOF {
			done <- true
			return nil
			//switch EOFcount {
			//case 0:
			//	o.Err("Press ctrl-d again to exit.")
			//	EOFcount++
			//	continue
			//default:
			//	done <- true
			//	return nil
			//}
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch line {
		case "help":
			outputHelp(o, generalHelpText)
			continue
		case "webhelp":
			outputHelp(o, webHelpText)
			continue
		case "confighelp":
			outputHelp(o, configHelpText)
			continue
		case "quit", "exit", "shutdown", "halt":
			done <- true
			return nil
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
