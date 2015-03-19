# Algernon

HTTP/2 web server that can serve Markdown and Lua scripts, with built-in support for users and permissions.

[http2check](https://github.com/xyproto/http2check) can be used to confirm that the server is in fact serving [HTTP/2](https://tools.ietf.org/html/draft-ietf-httpbis-http2-16).


Technologies
------------

Written in [Go](https://golang.org). Uses [Redis](https://redis.io) as the database backend, [permissions2](https://github.com/xyproto/permissions2) for handling users and permissions, [gopher-lua](https://github.com/yuin/gopher-lua) for interpreting and running Lua, [http2](https://github.com/bradfitz/http2) for serving HTTP/2 and [blackfriday](https://github.com/russross/blackfriday) for Markdown rendering.


Design decisions
----------------

* HTTP/2 over SSL/TLS (https) is used by default, if a certificate and key is given.
* If not, unencrypted HTTP is used.
* /data and /repos have user permissions, /admin has admin permissions and / is public.
* The following filenames are special, in prioritized order:
    * index.lua is interpreted as a handler function for the current directory
    * index.md is rendered as html
    * index.html is outputted as it is, with the correct Content-Type
    * index.txt is outputted as it is, with the correct Content-Type
* Other files are given a mimetype based on the extension.
* Directories without an index file are shown as a directory listing, where the design is hardcoded.
* Redis is used for the database backend.
* UTF-8 is used whenever possible.

Features
--------

* Supports HTTP/2.
* Works on OS X and Linux.
* Algernon is compiled to native. It's reasonably fast.
* The [Lua interpreter](https://github.com/yuin/gopher-lua) is compiled into the executable.
* The use of Lua allows for short development cycles, where code is interpreted when the page is refreshed.


Screenshots
-----------

<img src="https://raw.github.com/xyproto/algernon/master/img/algernon_redis.png">

*Screenshot of `algernon` and `redis` running in a terminal emulator.*

--

<img src="https://raw.github.com/xyproto/algernon/master/img/prettify.png">

*Screenshot of the <strong>prettify</strong> example. Served from a single Lua script.*



Getting started
---------------

##### Install Algernon

* Install [go](https://golang.org), set `$GOPATH` and add `$GOPATH/bin` to the PATH (optional).
* `go get github.com/xyproto/algernon`

##### Enable HTTP/2 in the browser

* Chrome: go to `chrome://flags/#enable-spdy4`, enable, save and restart the browser.
* Firefox: go to `about:config`, set `network.http.spdy.enabled.http2draft` to `true`. You might need the nightly version of Firefox.

##### Run the example

* Make sure Redis is running. On OS X, you can install `redis` with homebrew and start `redis-server`. On Linux, you can install `redis` and run `systemctl start redis-server`, depending on your distro.
* Run the "bob" example: `./runexample.sh`
* Visit `https://localhost:3000/`.

##### Create your own Algernon application

* `mkdir mypage`
* `cd mypage`
* `echo 'print("Hello, Algernon")' >> index.lua` (or use your favorite editor)
* Create a certificate just for testing:
 * `openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 3000 -nodes`
 * Just press return at all the prompts, but enter `localhost` at *Common Name*.
* Start `algernon`.
* Visit `https://localhost:3000/`.
* If you have not imported the certificates into the browser, or used certificates that are signed by trusted certificate authorities, perform the necessary clicks to confirm that you wish to visit this page.
* You can now edit and save the `index.lua` file and all you have to do is reload the browser page to see the new result (or error message, if the script had a problem).


Lua functions for handling requests
-----------------------------------

* `content(string)` sets the Content-Type for a page.
* `method()` returns the requested HTTP method (GET, POST etc).
* `print(...)` can be used for outputting data to the browser/client. Takes a variable number of strings.
* `mprint(...)` can be used for outputting markdown to the browser/client. The given text is converted from markdown to html. Takes a variable number of strings.
* `urlpath()` returns the requested URL path.
* `header(string)` returns the HTTP header in the request, for a given key, or an empty string.
* `body()` returns the HTTP body in the request (will only read the body once, since it's streamed).
* `version()` returns the version string for the server.
* `status(number)` sets a HTTP status code (like 200 or 404). Must come before other output.
* `error(string, number)` sets a HTTP status code and outputs a message.
* `scriptdir(...)` returns the directory where the script is running. If a filename is given, then the path to where the script is running, joined with a path separator and the given filename, is returned.
* `serverdir(...)` returns the directory where the server is running. If a filename is given, then the path to where the server is running, joined with a path separator and the given filename, is returned.


Lua functions for persistent data structures
--------------------------------------------

##### Set

~~~
// A Redis-backed Set (takes a name, returns an object)
Set(string) -> userdata

// Add an element to the set
set:add(string)

// Remove an element from the set
set:del(string)

// Check if a set contains a value
// Returns true only if the value exists and there were no errors.
set:has(string) -> bool

// Get all members of the set
set:getall() -> table

// Remove the set itself. Returns true if it worked out.
set:remove() -> bool
~~~

##### List

~~~
// A Redis-backed List (takes a name, returns an object)
List(string) -> userdata

// Add an element to the list
list:add(string)

// Get all members of the list
list::getall() -> table

// Get the last element of the list
// The returned value can be empty
list::getlast() -> string

// Get the N last elements of the list
list::getlastn(number) -> table

// Remove the list itself. Returns true if it worked out.
list:remove() -> bool
~~~

##### HashMap

~~~
// A Redis-backed HashMap (takes a name, returns an object)
HashMap(string) -> userdata

// For a given element id (for instance a user id), set a key
// (for instance "password") and a value.
// Returns true if it worked out.
hash:set(string, string, string) -> bool

// For a given element id (for instance a user id), and a key
// (for instance "password"), return a value.
// Returns a value only if they key was found and if there were no errors.
hash:get(string, string) -> string

// For a given element id (for instance a user id), and a key
// (for instance "password"), check if it exists in the hash map.
// Returns true only if it exists and there were no errors.
hash:has(string, string) -> bool

// For a given element id (for instance a user id), check if it exists.
// Returns true only if it exists and there were no errors.
hash:exists(string) -> bool

// Get all keys of the hash map
hash::getall() -> table

// Remove a key for an entry in a hash map
// (for instance the email field for a user)
// Returns true if it worked out
hash:delkey(string, string) -> bool

// Remove an element (for instance a user)
// Returns true if it worked out
hash:del(string) -> bool

// Remove the hash map itself. Returns true if it worked out.
hash:remove() -> bool
~~~

##### KeyValue

~~~
// A Redis-backed KeyValue collection (takes a name, returns an object)
KeyValue(string) -> userdata

// Set a key and value. Returns true if it worked out.
kv:set(string, string) -> bool

// Takes a key, returns a value. May return an empty string.
kv:get(string) -> string

// Remove a key. Returns true if it worked out.
kv:del(string) -> bool

// Remove the KeyValue itself. Returns true if it worked out.
kv:remove() -> bool
~~~


Lua functions for handling users and permissions
------------------------------------------------

~~~
// Check if the current user has "user" rights
UserRights() -> bool

// Check if the given username exists
HasUser(string) -> bool

// Get the value from the given boolean field
// Takes a username and fieldname
BooleanField(string, string) -> bool

// Save a value as a boolean field
// Takes a username, fieldname and boolean value
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

// Add a user to the list of unconfirmed users
// Takes a username and a confirmation code
AddUnconfirmed(string, string)

// Remove a user from the list of unconfirmed users
// Takes a username
RemoveUnconfirmed(string)

// Mark a user as confirmed
// Takes a username
MarkConfirmed(string)

// Removes a user
// Takes a username
RemoveUser(string)

// Make a user an admin
// Takes a username
SetAdminStatus(string)

// Make an admin user a regular user
// Takes a username
RemoveAdminStatus(string)

// Add a user
// Takes a username, password and email
AddUser(string, string, string)

// Set a user as logged in on the server (not cookie)
// Takes a username
SetLoggedIn(string)

// Set a user as logged out on the server (not cookie)
// Takes a username
SetLoggedOut(string)

// Log in a user, both on the server and with a cookie
// Takes a username
Login(string)

// Logs out a user, on the server (which is enough)
// Takes a username
Logout(string)

// Get the current username, from the cookie
Username() -> string

// Get the current cookie timeout
// Takes a username
CookieTimeout(string) -> number

// Set the current cookie timeout
// Takes a timeout number, measured in seconds
SetCookieTimeout(number)

// Get the current password hashing algorithm (bcrypt, bcrypt+ or sha256)
PasswordAlgo() -> string

// Set the current password hashing algorithm (bcrypt, bcrypt+ or sha256)
// Takes a string
SetPasswordAlgo(string)

// Hash the password
// Takes a username and password (username can be used for salting)
HashPassword(string, string) -> string

// Check if a given username and password is correct
// Takes a username and password
CorrectPassword(string) -> bool

// Checks if a confirmation code is already in use
// Takes a confirmation code
AlreadyHasConfirmationCode(string) -> bool

// Find a username based on a given confirmation code,
// or returns an empty string. Takes a confirmation code
FindUserByConfirmationCode(string) -> string

// Mark a user as confirmed
// Takes a username
Confirm(string)

// Mark a user as confirmed, returns true if it worked out
// Takes a confirmation code
ConfirmUserByConfirmationCode(string) -> bool

// Set the minimum confirmation code length
// Takes the minimum number of characters
SetMinimumConfirmationCodeLength(number)

// Generates a unique confirmation code, or an empty string
GenerateUniqueConfirmationCode() -> string
~~~


General information
-------------------

* Version: 0.47
* License: MIT
* Alexander F Rødseth

