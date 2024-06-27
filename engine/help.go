package engine

import (
	"fmt"
	"os"
)

const generalHelpText = `Available functions:

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
// Return all list elements (expected to be JSON strings) as a JSON list
list:json() -> string

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
// Add a reverse proxy given a path prefix and an endpoint URL
AddReverseProxy(string, string)

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
// Output rendered JavaScript given JSX for HyperApp. Takes a variable number of strings.
hprint(...)
// Output rendered JavaScript given JSX for React. Takes a variable number of strings.
jprint(...)
// Output a Pongo2 template and key/value table as rendered HTML. Use "{{ key }}" to insert a key.
poprint(string[, table])
// Output a simple HTML page with a message, title and theme.
msgpage(string[, string][, string])

Cache

CacheInfo() -> string // Return information about the file cache.
ClearCache() // Clear the file cache.
preload(string) -> bool // Load a file into the cache, returns true on success.

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
json(table[, number]) -> string
// Create a JSON document node.
JNode() -> userdata
// Add JSON data to a node. The first argument is an optional JSON path.
// The second argument is a JSON data string. Returns true on success.
// "x" is the default JSON path.
jnode:add([string, ]string) -> bool
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
// The content type is set to "application/json;charset=utf-8".
// The second argument is an optional authentication token that is used for the
// Authorization header field. Uses HTTP POST.
jnode:POST(string[, string]) -> string
// Sends JSON data to the given URL. Returns the HTTP status code as a string.
// The content type is set to "application/json;charset=utf-8".
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
// Convert from a simple Lua table to a JSON string
JSON(table) -> string

HTTP Requests

// Create a new HTTP Client object
HTTPClient() -> userdata
// Select Accept-Language (ie. "en-us")
hc:SetLanguage(string)
// Set the request timeout (in milliseconds)
hc:SetTimeout(number)
// Set a cookie (name and value)
hc:SetCookie(string, string)
// Set the user agent (ie. "curl")
hc:SetUserAgent(string)
// Perform a HTTP GET request. First comes the URL, then an optional table with
// URL paramets, then an optional table with HTTP headers.
hc:Get(string, [table], [table]) -> string
// Perform a HTTP POST request. It's the same arguments as for hc:Get, except
// the fourth optional argument is the POST body.
hc:Post(string, [table], [table], [string]) -> string
// Like hc:Get, except the first argument is the HTTP method (like "PUT")
hc:Do(string, string, [table], [table]) -> string
// Shorthand for HTTPClient():Get(). Retrieve an URL, with optional tables for
// URL parameters and HTTP headers.
GET(string, [table], [table]) -> string
// Shorthand for HTTPClient():Post(). Post to an URL, with optional tables for
// URL parameters and HTTP headers, followed by a string for the body.
POST(string, [table], [table], [string]) -> string
// Shorthand for HTTPClient():Do(). Like Get, but the first argument is the
// method, like ie. "PUT".
DO(string, string, [table], [table]) -> string

Plugins

// Load a plugin given the path to an executable. Returns true if successful.
// Will return the plugin help text if called on the Lua prompt. Pass true as
// the last argument to keep it running.
Plugin(string, [bool]) -> bool
// Returns the Lua code as returned by the Lua.Code function in the plugin,
// given a plugin path. Pass true as the last argument to keep it running.
// May return an empty string.
PluginCode(string, [bool]) -> string
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

AI

// Connect to an Ollama server. Takes an optional model:tag string and an optional host address.
OllamaClient([string], [string]) -> userdata
// List all models that are downloaded and ready.
oc:list()
// Check if the given model name is downloaded and ready.
oc:has(string)
// Get or set the current model, but don't pull anything.
oc:model([string])
// Download the required model, if needed. This can take a while the first time if the model is large.
oc:pull()
// Pass a prompt to Ollama and return the reproducible generated output. Can also take a model name.
oc:ask([string], [string]) -> string
// Pass a prompt to Ollama and generate output that will differ every time. Can also take a model name.
oc:creative([string], [string]) -> string
// Get the size of the given model name as a human-friendly string.
oc:size(string) -> string
// Get the size of the given model name, in bytes.
oc:bytesize(string) -> number
// Convenience function for passing a prompt and optional model name to the local Ollama server.
// The default prompt generates a poem and the default model is "tinyllama".
ollama([string], [string]) -> string
// Convenience function for Base64-encoding the given file.
base64EncodeFile(string) -> string
// Describe the given base64-encoded image using Ollama (and the "llava-llama3" model, by default).
describeImage(string, [string]) -> string
// Given two embeddings (tables of floats, representing text or data, as returned by Ollama), return how similar they are.
// The optional string is the algorithm for measuring the distance: "euclidean", "manhattan", "chebyshev" or "hamming".
embeddedDistance(table, table, [string]) -> number

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
// Sanitize HTML
sanhtml(string) -> string
// Query a PostgreSQL database with a query and a connection string.
// Default connection string: "host=localhost port=5432 user=postgres dbname=test sslmode=disable"
PQ([string], [string]) -> table
// Query a MSSQL database with a query and a connection string.
// Default connection string: "server=localhost;user=user;password=password,port=1433"
MSSQL([string], [string]) -> table

REPL-only

// Output the current working directory
cwd | pwd
// Output the current file or directory that is being served
serverdir | serverfile
// Exit Algernon
exit | halt | quit | shutdown

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

const usageMessage = `
Type "webhelp" for an overview of functions that are available when
handling requests. Or "confighelp" for an overview of functions that are
available when configuring an Algernon application.
`

const webHelpText = `Available functions:

Handling users and permissions

// Check if the current user has "user" rights
UserRights() -> bool
// Check if the given username exists (does not check unconfirmed users)
HasUser(string) -> bool
// Check if the given username exists in the list of unconfirmed users
HasUnconfirmedUser(string) -> bool
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
// Get the existing confirmation code for a given user,
// or an empty string. Takes a username.
ConfirmationCode(string) -> string
// Add a user to the list of unconfirmed users.
// Takes a username and a confirmation code.
// Remember to also add a user, when registering new users.
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
// Log out a user, on the server (which is enough). Takes a username.
Logout(string)
// Get the current username, from the cookie
Username() -> string
// Get the current cookie timeout. Takes a username.
CookieTimeout(string) -> number
// Set the current cookie timeout. Takes a timeout, in seconds.
SetCookieTimeout(number)
// Get the current server-wide cookie secret, for persistent logins
CookieSecret() -> string
// Set the current server-side cookie secret, for persistent logins
SetCookieSecret(string)
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
// Return the full textual content of the uploaded file
uploadedfile:content() -> string
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
// Read a glob, ie. "*.md" in the current script directory, or the given dir.
// The contents of all found files are reeturned as a table.
readglob(string[, string]) -> table
// Return the directory where the server is running. If a filename (optional)
// is given, then the path to where the server is running, joined with a path
// separator and the given filename, is returned.
serverdir([string]) -> string
// Serve a file that exists in the same directory as the script.
serve(string)
// Serve a Pongo2 template file, with an optional table with key/values.
serve2(string[, table)
// Return the rendered contents of a file that exists in the same directory
// as the script. Takes a filename.
render(string) -> string
// Return a table with keys and values as given in a posted form, or as given
// in the URL ("/some/page?x=7" makes "x" with the value "7" available).
formdata() -> table
// Redirect to an absolute or relative URL. Also takes a HTTP status code.
redirect(string[, number])
// Permanently redirect to an absolute or relative URL. Uses status code 302.
permanent_redirect(string)
// Send "Connection: close" as a header to the client, flush the body and also
// stop Lua functions from writing more data to the HTTP body.
close()
// Transmit what has been outputted so far, to the client.
flush()
`

const configHelpText = `Available functions:

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
// Get the cookie secret from the server configuration.
CookieSecret() -> string
// Set the cookie secret that will be used when setting and getting browser cookies.
SetCookieSecret(string)

`

func generateUsageFunction(ac *Config) func() {
	return func() {
		fmt.Println("\n" + ac.versionString + "\n\n" + ac.description)

		var quicExample, quicUsageOrMessage, quicFinalMessage string

		// Prepare and/or output a message, depending on if QUIC support is compiled in or not
		if quicEnabled {
			quicUsageOrMessage = "\n  -u                           Serve over QUIC / HTTP3."
			quicExample = "\n  Serve the current dir over QUIC, port 7000, no banner:\n    algernon -s -u -n . :7000\n"
		} else {
			quicFinalMessage = "\n\nThis Algernon executable was built without QUIC support."
		}

		// Possible arguments are also, for backward compatibility:
		// server dir, server addr, certificate file, key file, redis addr and redis db index
		// They are not mentioned here, but are possible to use, in that strict order.
		fmt.Println(`
Syntax:
  algernon [flags] [file or directory to serve] [host][:port]

Available flags:
  -a, --autorefresh            Enable event server and auto-refresh feature.
                               Sets cache mode to "images".
  -b, --bolt                   Use "` + ac.defaultBoltFilename + `"
                               for the Bolt database.
  -c, --statcache              Speed up responses by caching os.Stat.
                               Only use if served files will not be removed.
  -d, --debug                  Enable debug mode (show errors in the browser).
  -e, --dev                    Development mode: Enables Debug mode, uses
                               regular HTTP, Bolt and sets cache mode "dev".
  -h, --help                   This help text
  -l, --lua                    Don't serve anything, just present the Lua REPL.
  -m                           View the given Markdown file in the browser.
                               Quits after the file has been served once.
                               ("-m" is equivalent to "-q -o -z").
  -n, --nobanner               Don't display a colorful banner at start.
  -o, --open=EXECUTABLE        Open the served URL in a browser with a standard
                               methodor or with the given (optional) executable.
  -p, --prod                   Serve HTTP/2+HTTPS on port 443. Serve regular
                               HTTP on port 80. Uses /srv/algernon for files.
                               Disables debug mode. Disables auto-refresh.
                               Enables server mode. Sets cache to "prod".
  -q, --quiet                  Don't output anything to stdout or stderr.
  -r, --redirect               Redirect HTTP traffic to HTTPS, if both are enabled.
  -s, --server                 Server mode (disable debug + interactive mode).
  -t, --httponly               Serve regular HTTP.` + quicUsageOrMessage + `
  -v, --version                Application name and version
  -V, --verbose                Slightly more verbose logging.
  -z, --quit                   Quit after the first request has been served.
  --accesslog=FILENAME         Access log filename. Logged in Combined Log Format (CLF).
  --addr=[HOST][:PORT]         Server host and port ("` + ac.defaultWebColonPort + `" is default)
  --boltdb=FILENAME            Use a specific file for the Bolt database
  --cache=MODE                 Sets a cache mode. The default is "on".
                               "on"      - Cache everything.
                               "dev"     - Everything, except Amber,
                                           Lua, GCSS, Markdown and JSX.
                               "prod"    - Everything, except Amber and Lua.
                               "small"   - Like "prod", but only files <= 64KB.
                               "images"  - Only images (png, jpg, gif, svg).
                               "off"     - Disable caching.
  --cachesize=N                Set the total cache size, in bytes.
  --cert=FILENAME              TLS certificate, if using HTTPS.
  --conf=FILENAME              Lua script with additional configuration.
  --clear                      Clear the default URI prefixes that are used
                               when handling permissions.
  --cookiesecret=STRING        Secret that will be used for login cookies.
  --ctrld                      Press ctrl-d twice to exit the REPL.
  --dbindex=INDEX              Redis database index (0 is default).
  --dir=DIRECTORY              Set the server directory
  --domain                     Serve files from the subdirectory with the same
                               name as the requested domain.
  --eventrefresh=DURATION      How often the event server should refresh
                               (the default is "` + ac.defaultEventRefresh + `").
  --eventserver=[HOST][:PORT]  SSE server address (for filesystem changes).
  --http2only                  Serve HTTP/2, without HTTPS.
  --internal=FILENAME          Internal log file (can be a bit verbose).
  --key=FILENAME               TLS key, if using HTTPS.
  --largesize=N                Threshold for not reading static files into memory, in bytes.
  --letsencrypt                Use certificates provided by Let's Encrypt for all served
                               domains and serve over regular HTTPS by using CertMagic.
  --limit=N                    Limit clients to N requests per second
                               (the default is ` + ac.defaultLimitString + `).
  --log=FILENAME               Log to a file instead of to the console.
  --maria=DSN                  Use the given MariaDB or MySQL host/database.
  --mariadb=NAME               Use the given MariaDB or MySQL database name.
  --ncsa=FILENAME              Alternative access log filename. Logged in Common Log Format (NCSA).
  --nocache                    Another way to disable the caching.
  --nodb                       No database backend. (same as --boltdb=` + os.DevNull + `).
  --noheaders                  Don't use the security-related HTTP headers.
  --nolimit                    Disable rate limiting.
  --postgres=DSN               Use the given PostgreSQL host/database.
  --postgresdb=NAME            Use the given PostgreSQL database name.
  --redis=[HOST][:PORT]        Use "` + ac.defaultRedisColonPort + `" for the Redis database.
  --rawcache                   Disable cache compression.
  --servername=STRING          Custom HTTP header value for the Server field.
  --stricter                   Stricter HTTP headers (same origin policy).
  --theme=NAME                 Builtin theme to use for Markdown, error pages,
                               directory listings and HyperApp apps.
                               Possible values are: light, dark, bw, redbox, wing,
                               material, neon or werc.
  --timeout=N                  Timeout when serving files, in seconds.
  --watchdir=DIRECTORY         Enables auto-refresh for only this directory.
  -x, --simple                 Serve as regular HTTP, enable server mode and
                               disable all features that requires a database.

Example usage:

  For auto-refreshing a webpage while developing:
    algernon --dev --httponly --debug --autorefresh --bolt --server . :4000

  Serve /srv/mydomain.com and /srv/otherweb.com over HTTP and HTTPS + HTTP/2:
    algernon -c --domain --server --cachesize 67108864 --prod /srv
` + quicExample + `
  Serve the current directory over HTTP, port 3000. No limits, cache,
  permissions or database connections:
    algernon -x` + quicFinalMessage)
	}
}
