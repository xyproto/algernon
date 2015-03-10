# Algernon

HTTP/2 web server that can serve dynamic lua scripts.

Technologies
------------

Written in [Go](https://golang.org). Uses [Redis](https://redis.io) as the database backend, [permissions2](https://github.com/xyproto/permissions2) for handling users and permissions, [gopher-lua](https://github.com/yuin/gopher-lua) for interpreting and running Lua, [http2](https://github.com/bradfitz/http2) for serving HTTP/2 and [blackfriday](https://github.com/russross/blackfriday) for Markdown rendering.

Design choices
--------------
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

LUA functions
-------------
* setContentType(string) for setting Content-Type
* print(string, string) for outputting data to the browser. Print always takes two arguments, for now.

~~~
    User/permission functions that are exposed to Lua scripts
    ---------------------------------------------------------

	// Check if the current user has "user rights", returns bool
	// Takes no arguments
	UserRights

	// Check if the given username exists, returns bool
	// Takes a username
	HasUser

	// Get the value from the given boolean field, returns bool
	// Takes a username and fieldname
	BooleanField

	// Save a value as a boolean field, returns nothing
	// Takes a username, fieldname and boolean value
	SetBooleanField

	// Check if a given username is confirmed, returns a bool
	// Takes a username
	IsConfirmed

	// Check if a given username is logged in, returns a bool
	// Takes a username
	IsLoggedIn

	// Check if the current user has "admin rights", returns a bool
	// Takes no arguments.
	AdminRights

	// Check if a given username is an admin, returns a bool
	// Takes a username
	IsAdmin

	// Get the username stored in a cookie, or an empty string
	// Takes no arguments
	UsernameCookie

	// Store the username in a cookie, returns true if successful
	// Takes a username
	SetUsernameCookie

	// Get the username stored in a cookie, or an empty string
	// Takes no arguments
	AllUsernames

	// Get the email for a given username, or an empty string
	// Takes a username
	Email

	// Get the password hash for a given username, or an empty string
	// Takes a username
	PasswordHash

	// Get all unconfirmed usernames
	// Takes no arguments
	AllUnconfirmedUsernames

	// Get a confirmation code that can be given to a user, or an empty string
	// Takes a username
	ConfirmationCode

	// Add a user to the list of unconfirmed users, returns nothing
	// Takes a username and a confirmation code
	AddUnconfirmed

	// Remove a user from the list of unconfirmed users, returns nothing
	// Takes a username
	RemoveUnconfirmed

	// Mark a user as confirmed, returns nothing
	// Takes a username
	MarkConfirmed

	// Removes a user, returns nothing
	// Takes a username
	RemoveUser

	// Make a user an admin, returns nothing
	// Takes a username
	SetAdminStatus

	// Make an admin user a regular user, returns nothing
	// Takes a username
	RemoveAdminStatus

	// Add a user, returns nothing
	// Takes a username, password and email
	AddUser

	// Set a user as logged in on the server (not cookie), returns nothing
	// Takes a username
	SetLoggedIn

	// Set a user as logged out on the server (not cookie), returns nothing
	// Takes a username
	SetLoggedOut

	// Log in a user, both on the server and with a cookie. Returns nothing
	// Takes a username
	Login

	// Logs out a user, on the server (which is enough). Returns nothing
	// Takes a username
	Logout

	// Get the current username, from the cookie
	// Takes nothing
	Username

	// Get the current cookie timeout
	// Takes a username
	CookieTimeout

	// Set the current cookie timeout
	// Takes a timeout number, measured in seconds
	SetCookieTimeout

	// Get the current password hashing algorithm (bcrypt, bcrypt+ or sha256)
	// Takes nothing
	PasswordAlgo

	// Set the current password hashing algorithm (bcrypt, bcrypt+ or sha256)
	// Takes a string
	SetPasswordAlgo

	// Hash the password, returns a string
	// Takes a username and password (username can be used for salting)
	HashPassword

	// Check if a given username and password is correct, returns a bool
	// Takes a username and password
	CorrectPassword

	// Checks if a confirmation code is already in use, returns a bool
	// Takes a confirmation code
	AlreadyHasConfirmationCode

	// Find a username based on a given confirmation code, or returns an empty string
	// Takes a confirmation code
	FindUserByConfirmationCode

	// Mark a user as confirmed, returns nothing
	// Takes a username
	Confirm

	// Mark a user as confirmed, returns true if it worked out
	// Takes a confirmation code
	ConfirmUserByConfirmationCode

	// Set the minimum confirmation code length
	// Takes the minimum number of characters
	SetMinimumConfirmationCodeLength

	// Generates and returns a unique confirmation code, or an empty string
	// Takes no parameters
	ConfirmUserByConfirmationCode
~~~

General information
-------------------

* Version: 0.2
* License: MIT
* Alexander F Rødseth

