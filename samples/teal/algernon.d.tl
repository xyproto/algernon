-- Return the version string for the server.
global version: function(): string

-- Sleep the given number of seconds (can be a float).
global sleep: function(sec: number)

-- Log the given strings as information. Takes a variable number of strings.
global log: function(...: string)

-- Log the given strings as a warning. Takes a variable number of strings.
global warn: function(...: string)

-- Log the given strings as an error. Takes a variable number of strings.
global err: function(...: string)

-- Return the number of nanoseconds from 1970 ("Unix time").
global unixnano: function(): number

-- Convert Markdown to HTML.
global markdown: function(data: string): string

-- Sanitize HTML.
global sanhtml: function(data: string): string

-- Return the directory where the REPL or script is running. If a filename (optional) is given, then the path to where the script is running, joined with a path separator and the given filename, is returned.
global scriptdir: function(filename: string | nil): string

-- Read a glob pattern (e.g., "*.md") in the current script directory, or the given directory (optional). The contents of all found files are returned as a table.
global readglob: function(pattern: string, directory: string | nil): {string}

-- Return the directory where the server is running. If a filename (optional) is given, then the path to where the server is running, joined with a path separator and the given filename, is returned.
global serverdir: function(filename: string | nil): string

-- Set the Content-Type for a page.
global content: function(ct: string)

-- Return the requested HTTP method (GET, POST, etc.).
global method: function(): string

-- Return the requested URL path.
global urlpath: function(): string

-- Return the HTTP header in the request, for a given key, or an empty string.
global header: function(key: string): string

-- Set an HTTP header given a key and a value.
global setheader: function(key: string, val: string)

-- Return the HTTP headers, as a table.
global headers: function(): {string:string}

-- Return the HTTP body in the request (will only read the body once, since it's streamed).
global body: function(): string

-- Set an HTTP status code (like 200 or 404). Must be used before other functions that write to the client!
global status: function(code: number)

-- Serve a file that exists in the same directory as the script. Takes a filename.
global serve: function(filename: string)

-- Serve a Pongo2 template file, with an optional table with template key/values.
global serve2: function(filename: string, args: {string:any} | nil)

-- Return the rendered contents of a file that exists in the same directory as the script. Takes a filename.
global render: function(filename: string): string

-- Return a table with keys and values as given in a posted form, or as given in the URL.
global formdata: function(): {string:string}

-- Return a table with keys and values as given in the request URL, or in the given URL (`/some/page?x=7` makes the key `x` with the value `7` available).
global urldata: function(url: string | nil): {string:string}

-- Redirect to an absolute or relative URL. May take an HTTP status code that will be used when redirecting.
global redirect: function(path: string, code: number | nil)

-- Permanent redirect to an absolute or relative URL. Uses status code 302.
global permanent_redirect: function(path: string)

-- Send "Connection: close" as a header to the client, flush the body, and also stop Lua functions from writing more data to the HTTP body.
global close: function()

-- Transmit what has been outputted so far, to the client.
global flush: function()

-- Return a table containing the parsed JSON document.
global json: function(data: any): string

-- Query a MSSQL database with SQL, a connection string, and a parameter table.
global MSSQL: function(sql: string, conn: string, args: {string:any} | nil): {{string:any}}

-- Send an HTTP POST request.
global POST: function(url: string, params: {string:string}, headers: {string:string}): string

-- Send an HTTP GET request.
global GET: function(url: string, params: {string:string}, headers: {string:string}): string

-- Return the username stored in a cookie, or an empty string.
global UsernameCookie: function(): string

-- Store the username in a cookie. Returns true on success.
global SetUsernameCookie: function(username: string): boolean

-- Clear the login cookie.
global ClearCookie: function()

-- Check if the current user has "user" rights.
global UserRights: function(): boolean

-- Check if the given username exists (does not look at the list of unconfirmed users).
global HasUser: function(username: string): boolean

-- Check if the given username exists in the list of unconfirmed users.
global HasUnconfirmedUser: function(username: string): boolean

-- Get the value from the given boolean field. Takes a username and field name.
global BooleanField: function(username: string, field: string): boolean

-- Save a value as a boolean field. Takes a username, field name, and boolean value.
global SetBooleanField: function(username: string, field: string, value: boolean)

-- Check if a given username is confirmed.
global IsConfirmed: function(username: string): boolean

-- Check if a given username is logged in.
global IsLoggedIn: function(username: string): boolean

-- Check if the current user has "admin" rights.
global AdminRights: function(): boolean

-- Check if a given username is an admin.
global IsAdmin: function(username: string): boolean

-- Get a table containing all usernames.
global AllUsernames: function(): {string}

-- Get the email for a given username, or an empty string.
global Email: function(username: string): string

-- Get the password hash for a given username, or an empty string.
global PasswordHash: function(username: string): string

-- Get all unconfirmed usernames.
global AllUnconfirmedUsernames: function(): {string}

-- Get the existing confirmation code for a given user, or an empty string. Takes a username.
global ConfirmationCode: function(username: string): string

-- Add a user to the list of unconfirmed users. Takes a username and a confirmation code.
global AddUnconfirmed: function(username: string, code: string)

-- Remove a user from the list of unconfirmed users. Takes a username.
global RemoveUnconfirmed: function(username: string)

-- Mark a user as confirmed. Takes a username.
global MarkConfirmed: function(username: string)

-- Removes a user. Takes a username.
global RemoveUser: function(username: string)

-- Make a user an admin. Takes a username.
global SetAdminStatus: function(username: string)

-- Make an admin user a regular user. Takes a username.
global RemoveAdminStatus: function(username: string)

-- Add a user. Takes a username, password, and email.
global AddUser: function(username: string, password: string, email: string)

-- Set a user as logged in on the server (not cookie). Takes a username.
global SetLoggedIn: function(username: string)

-- Set a user as logged out on the server (not cookie). Takes a username.
global SetLoggedOut: function(username: string)

-- Log in a user, both on the server and with a cookie. Takes a username.
global Login: function(username: string)

-- Log out a user, on the server (which is enough). Takes a username.
global Logout: function(username: string)

-- Get the current username, from the cookie.
global Username: function(): string

-- Get the current cookie timeout. Takes a username.
global CookieTimeout: function(username: string): number

-- Set the current cookie timeout. Takes a timeout number, measured in seconds.
global SetCookieTimeout: function(timeout: number)

-- Get the current server-wide cookie secret. This is used when setting and getting browser cookies when users log in.
global CookieSecret: function(): string

-- Set the current server-side cookie secret. This is used when setting and getting browser cookies when users log in. Using the same secret makes browser cookies usable across server restarts.
global SetCookieSecret: function(secret: string)

-- Get the current password hashing algorithm (bcrypt, bcrypt+, or sha256).
global PasswordAlgo: function(): string

-- Set the current password hashing algorithm (bcrypt, bcrypt+, or sha256). 'bcrypt+' accepts bcrypt or sha256 for old passwords, but will only use bcrypt for new passwords.
global SetPasswordAlgo: function(algo: string)

-- Hash the password. Takes a username and password (username can be used for salting sha256).
global HashPassword: function(username: string, password: string): string

-- Change the password for a user, given a username and a new password.
global SetPassword: function(username: string, password: string)

-- Check if a given username and password is correct. Takes a username and password.
global CorrectPassword: function(username: string, password: string): boolean

-- Check if a confirmation code is already in use. Takes a confirmation code.
global AlreadyHasConfirmationCode: function(code: string): boolean

-- Find a username based on a given confirmation code, or return an empty string. Takes a confirmation code.
global FindUserByConfirmationCode: function(code: string): string

-- Mark a user as confirmed. Takes a username.
global Confirm: function(username: string)

-- Mark a user as confirmed, returns true on success. Takes a confirmation code.
global ConfirmUserByConfirmationCode: function(code: string): boolean

-- Set the minimum confirmation code length. Takes the minimum number of characters.
global SetMinimumConfirmationCodeLength: function(length: number)

-- Generates a unique confirmation code, or an empty string.
global GenerateUniqueConfirmationCode: function(): string

-- Pretty print. Outputs the values in, or a description of, the given Lua value(s).
global pprint: function(...: any)

-- Takes a Python filename, executes the script with the `python` binary in the Path. Returns the output as a Lua table, where each line is an entry.
global py: function(filename: string): {string}

-- Takes one or more system commands (possibly separated by `;`) and runs them. Returns the output lines as a table.
global run: function(command: string): {string}

-- Lists the keys and values of a Lua table. Returns a string. Lists the contents of the global namespace `_G` if no arguments are given.
global dir: function(tbl: table | nil): string
