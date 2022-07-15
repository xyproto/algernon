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

-- Return the number of nanoseconds from 1970 ("Unix time")
global unixnano: function(): number

-- Convert Markdown to HTML
global markdown: function(data: string): string

-- Set the Content-Type for a page.
global content: function(ct: string)

-- Return the requested HTTP method (GET, POST etc).
global method: function(): string

-- Return the requested URL path.
global urlpath: function(): string

-- Return the HTTP header in the request, for a given key, or an empty string.
global header: function(key: string): string

-- Set an HTTP header given a key and a value.
global setheader: function(key: string, val: string)

global print: function(...: string)

-- Return the HTTP headers, as a table.
global headers: function(): {string:string}

-- Return the HTTP body in the request (will only read the body once, since it's streamed).
global body: function(): string

-- Set a HTTP status code (like 200 or 404). Must be used before other functions that writes to the client!
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

global MSSQL: function(sql: string, conn: string, args: {string:any} | nil): {{string:any}}

global POST: function(url: string, params: {string:string}, headers: {string:string}): string

global GET: function(url: string, params: {string:string}, headers: {string:string}): string

global json: function(any): string
