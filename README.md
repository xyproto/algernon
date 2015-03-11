# Algernon

HTTP/2 web server that can serve markdown and dynamic lua scripts.

[http2check](https://github.com/xyproto/http2check) can be used to confirm that the server is in fact serving HTTP/2.


Technologies
------------

Written in [Go](https://golang.org). Uses [Redis](https://redis.io) as the database backend, [permissions2](https://github.com/xyproto/permissions2) for handling users and permissions, [gopher-lua](https://github.com/yuin/gopher-lua) for interpreting and running Lua, [http2](https://github.com/bradfitz/http2) for serving HTTP/2 and [blackfriday](https://github.com/russross/blackfriday) for Markdown rendering.


Screenshot
----------

<img src="https://raw.github.com/xyproto/algernon/master/img/screenshot.png">

Screenshot of the "prettify" example. Served from a single lua script.


Design decisions
----------------

* HTTP/2 over SSL/TLS (https) is used by default, if a certificate and key is given.
* If not, unecrypted HTTP is used.
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


LUA functions for handling requests
-----------------------------------

* `content(string)` sets the Content-Type for a page.
* `method()` returns the requested HTTP method (GET, POST etc).
* `print(...)` can be used for outputting data to the browser. Takes a variable number of strings.
* `mprint(...)` can be used for outputting markdown to the browser. The given text is converted from markdown to html. Takes a variable number of strings.
* `urlpath()` returns the current URL path.
* `header(string)` returns the HTTP header in the request, for a given key, or an empty string.
* `body()` returns the HTTP body in the request (will only read the body once, since it's streamed).
* `version()` returns the version string for the server.
* `status(number)` sets a HTTP status code (like 200 or 404). Must come before any printing.
* `error(string, number)` sets a HTTP status code and outputs a message.
* `scriptdir(...)` returns the directory where the script is running. If a filename is given, then the path to where the script is running, joined with a path separator and the given filename, is returned.
* `serverdir(...)` returns the directory where the server is running. If a filename is given, then the path to where the server is running, joined with a path separator and the given filename, is returned.


LUA functions for handling users and permissions
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

// Get a confirmation code that can be given to a user, or an empty string
// Takes a username
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

// Generates and returns a unique confirmation code, or an empty string
GenerateUniqueConfirmationCode() -> string
~~~

General information
-------------------

* Version: 0.41
* License: MIT
* Alexander F Rødseth

