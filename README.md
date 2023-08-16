<!--
title: Algernon
description: Web server with built-in support for Lua, Teal, Markdown, Pongo2, Amber, Sass, SCSS, GCSS, JSX, Bolt, PostgreSQL, Redis, MariaDB, MySQL, Tollbooth, Pie, Graceful, Permissions2, users and permissions
keywords: web server, QUIC, lua, teal, markdown, pongo2, application server, http, http2, HTTP/2, go, golang, algernon, JSX, React, BoltDB, Bolt, PostgreSQL, Redis, MariaDB, MySQL, Three.js
theme: material
-->

<!--<a href="https://github.com/xyproto/algernon"><img src="https://algernon.roboticoverlords.org/img/algernon_logo.png" style="margin-left: 2em"></a>-->
![Algernon](img/algernon_logo.png)

![Build](https://github.com/xyproto/algernon/workflows/Build/badge.svg) [![GoDoc](https://godoc.org/github.com/xyproto/algernon?status.svg)](https://godoc.org/github.com/xyproto/algernon) [![License](https://img.shields.io/badge/license-BSD-green.svg?style=flat)](https://raw.githubusercontent.com/xyproto/algernon/main/LICENSE) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/algernon)](https://goreportcard.com/report/github.com/xyproto/algernon) [![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fxyproto%2Falgernon.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fxyproto%2Falgernon?ref=badge_shield) [![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://stand-with-ukraine.pp.ua)

Web server with built-in support for QUIC, HTTP/2, Lua, Teal, Markdown, Pongo2, HyperApp, Amber, Sass(SCSS), GCSS, JSX, BoltDB (built-in, stores the database in a file, like SQLite), Redis, PostgreSQL, MariaDB/MySQL, rate limiting, graceful shutdown, plugins, users and permissions.

All in one small self-contained executable.

Distro Packages
---------------

[![Packaging status](https://repology.org/badge/vertical-allrepos/algernon.svg)](https://repology.org/project/algernon/versions)

Quick installation (development version)
----------------------------------------

Requires Go 1.21 or later.

    go install github.com/xyproto/algernon@latest

Or manually:

    git clone https://github.com/xyproto/algernon
    cd algernon
    go build -mod=vendor
    ./welcome.sh

Releases and pre-built images
-----------------------------

See the [release](https://github.com/xyproto/algernon/releases/latest) page for releases for a variety of platforms and architectures.

The [docker image](https://hub.docker.com/r/xyproto/algernon/tags) is a total of 9MB.

Technologies
------------

Written in [Go](https://golang.org). Uses [Bolt](https://github.com/coreos/bbolt) (built-in), [MySQL](https://github.com/go-sql-driver/mysql), [PostgreSQL](https://www.postgresql.org/) or [Redis](https://redis.io) (recommended) for the database backend, [permissions2](https://github.com/xyproto/permissions2) for handling users and permissions, [gopher-lua](https://github.com/yuin/gopher-lua) for interpreting and running Lua, optional [Teal](https://github.com/teal-language/tl) for type-safe Lua scripting, [http2](https://github.com/bradfitz/http2) for serving HTTP/2, [QUIC](https://github.com/xyproto/quic) for serving over QUIC, [gomarkdown/markdown](https://github.com/gomarkdown/markdown) for Markdown rendering, [amber](https://github.com/eknkc/amber) for Amber templates, [Pongo2](https://github.com/flosch/pongo2) for Pongo2 templates, [Sass](https://github.com/wellington/sass)(SCSS) and [GCSS](https://github.com/yosssi/gcss) for CSS preprocessing. [logrus](https://github.com/Sirupsen/logrus) is used for logging, [goja-babel](github.com/jvatic/goja-babel) for converting from JSX to JavaScript, [tollbooth](https://github.com/didip/tollbooth) for rate limiting, [pie](https://github.com/natefinch/pie) for plugins and [graceful](https://github.com/tylerb/graceful) for graceful shutdowns.

Design decisions
----------------

* HTTP/2 over SSL/TLS (https) is used by default, if a certificate and key is given.
  * If not, regular HTTP is used.
* QUIC ("HTTP over UDP", HTTP/3) can be enabled with a flag.
* /data and /repos have user permissions, /admin has admin permissions and / is public, by default. This is configurable.
* The following filenames are special, in prioritized order:
    * index.lua is Lua code that is interpreted as a handler function for the current directory.
    * index.html is HTML that is outputted with the correct Content-Type.
    * index.md is Markdown code that is rendered as HTML.
    * index.txt is plain text that is outputted with the correct Content-Type.
    * index.pongo2, index.po2 or index.tmpl is Pongo2 code that is rendered as HTML.
    * index.amber is Amber code that is rendered as HTML.
    * index.hyper.js or index.hyper.jsx is JSX+HyperApp code that is rendered as HTML
    * index.tl is Teal code that is interpreted as a handler function for the current directory.
    * data.lua is Lua code, where the functions and variables are made available for Pongo2, Amber and Markdown pages in the same directory.
    * If a single Lua script is given as a command line argument, it will be used as a standalone server. It can be used for setting up handlers or serving files and directories for specific URL prefixes.
    * style.gcss is GCSS code that is used as the style for all Pongo2, Amber and Markdown pages in the same directory.
* The following filename extensions are handled by Algernon:
    * Markdown: .md (rendered as HTML)
    * Pongo2: .po2, .pongo2 or .tpl (rendered as any text, typically HTML)
    * Amber: .amber (rendered as HTML)
    * Sass: .scss (rendered as CSS)
    * GCSS: .gcss (rendered as CSS)
    * JSX: .jsx (rendered as JavaScript/ECMAScript)
    * Lua: .lua (a script that provides its own output and content type)
    * Teal: .tl (same as .lua but with type safety)
    * HyperApp: .hyper.js or .hyper.jsx (rendered as HTML)
* Other files are given a mimetype based on the extension.
* Directories without an index file are shown as a directory listing, where the design is hard coded.
* UTF-8 is used whenever possible.
* The server can be configured by command line flags or with a lua script, but no configuration should be needed for getting started.

Features and limitations
------------------------

* Supports HTTP/2, with or without HTTPS (browsers may require HTTPS when using HTTP/2).
* Also supports QUIC and regular HTTP.
* Can use Lua scripts as handlers for HTTP requests.
* The Algernon executable is compiled to native and is reasonably fast.
* Works on Linux, macOS and 64-bit Windows.
* The [Lua interpreter](https://github.com/yuin/gopher-lua) is compiled into the executable.
* The [Teal typechecker](https://github.com/teal-language/tl) is loaded into the Lua VM.
* Live editing/preview when using the auto-refresh feature.
* The use of Lua allows for short development cycles, where code is interpreted when the page is refreshed (or when the Lua file is modified, if using auto-refresh).
* Self-contained Algernon applications can be zipped into an archive (ending with `.zip` or `.alg`) and be loaded at start.
* Built-in support for [Markdown](https://github.com/gomarkdown/markdown), [Pongo2](https://github.com/flosch/pongo2), [Amber](https://github.com/eknkc/amber), [Sass](https://github.com/wellington/sass)(SCSS), [GCSS](https://github.com/yosssi/gcss) and [JSX](https://github.com/mamaar/risotto).
* Redis is used for the database backend, by default.
* Algernon will fall back to the built-in Bolt database if no Redis server is available.
* The HTML title for a rendered Markdown page can be provided by the first line specifying the title, like this: `title: Title goes here`. This is a subset of MultiMarkdown.
* No file converters needs to run in the background (like for SASS). Files are converted on the fly.
* If `-autorefresh` is enabled, the browser will automatically refresh pages when the source files are changed. Works for Markdown, Lua error pages and Amber (including Sass, GCSS and *data.lua*). This only works on Linux and macOS, for now. If listening for changes on too many files, the OS limit for the number of open files may be reached.
* Includes an interactive REPL.
* If only given a Markdown filename as the first argument, it will be served on port 3000, without using any database, as regular HTTP. Handy for viewing `README.md` files locally.
* Full multi-threading. All available CPUs will be used.
* Supports rate limiting, by using [tollbooth](https://github.com/didip/tollbooth).
* The `help` command is available at the Lua REPL, for a quick overview of the available Lua functions.
* Can load plugins written in any language. Plugins must offer the `Lua.Code` and `Lua.Help` functions and talk JSON-RPC over stderr+stdin. See [pie](https://github.com/natefinch/pie) for more information. Sample plugins for Go and Python are in the `plugins` directory.
* Thread-safe file caching is built-in, with several available cache modes (for only caching images, for example).
* Can read from and save to JSON documents. Supports simple JSON path expressions (like a simple version of XPath, but for JSON).
* If cache compression is enabled, files that are stored in the cache can be sent directly from the cache to the client, without decompressing.
* Files that are sent to the client are compressed with [gzip](https://golang.org/pkg/compress/gzip/#BestSpeed), unless they are under 4096 bytes.
* When using PostgreSQL, the HSTORE key/value type is used (available in PostgreSQL version 9.1 or later).
* No external dependencies, only pure Go.
* Requires Go >= 1.21 or a version of GCC/`gccgo` that supports Go 1.21.
* The Lua implementation used in Algernon (gopherlua) does not support `package.loadlib`.

Q&A
---

Q:

> What is the benefit of using this? In what scenario would this excel? Thanks. -- [mtw@HN](https://news.ycombinator.com/item?id=19583144).

A:

> Good question. I'm not sure if it excels in any scenario. There are specialized web servers that excel at caching or at raw performance. There are dedicated backends for popular front-end toolkits like Vue or React. There are dedicated editors that excel at editing and previewing Markdown, or HTML.
>
> I guess the main benefit is that Algernon covers a lot of ground, with a minimum of configuration, while being powerful enough to have a plugin system and support for programming in Lua. There is an auto-refresh feature that uses Server Sent Events, when editing Markdown or web pages. There is also support for the latest in Web technologies, like HTTP/2, QUIC and TLS 1.3. The caching system is decent. And the use of Go ensures that also smaller platforms like NetBSD and systems like Raspberry Pi are covered. There are no external dependencies, so Algernon can run on any system that Go can support.
>
> The main benefit is that is is versatile, fresh, and covers many platforms and use cases.
>
> For a more specific description of a potential benefit, a more specific use case would be needed.

Utilities
---------
* Comes with the `alg2docker` utility, for creating Docker images from Algernon web applications (`.alg` files).
* [http2check](https://github.com/xyproto/http2check) can be used for checking if a web server is offering [HTTP/2](https://tools.ietf.org/html/rfc7540).

Installation
------------------

##### macOS

* Install [Homebrew](https://brew.sh), if needed.
* `brew install algernon`

##### Arch Linux

* `pacman -S algernon`

##### Any system where Go is available

This method is using the latest commit from the main branch:

`go get -u github.com/xyproto/algernon@main`

If needed, first:

* Set the GOPATH. For example: `export GOPATH=~/go`
* Add $GOPATH/bin to the path. For example: `export PATH=$PATH:$GOPATH/bin`

Overview
--------

Running Algernon:

<img src="https://raw.github.com/xyproto/algernon/main/img/algernon_gopher.png">

Screenshot of an earlier version:

<img src="https://raw.github.com/xyproto/algernon/main/img/algernon_redis_054.png">

---

The idea is that web pages can be written in Markdown, Pongo2, Amber, HTML or JSX (+React), depending on the need, and styled with CSS, Sass(SCSS) or GCSS, while data can be provided by a Lua script that talks to Redis, BoltDB, PostgreSQL or MariaDB/MySQL.

Amber and GCSS is a good combination for static pages, that allows for more clarity and less repetition than HTML and CSS. It˙s also easy to use Lua for providing data for the Amber templates, which helps separate model, controller and view.

Pongo2, Sass and Lua also combines well. Pongo2 is more flexible than Amber.

The auto-refresh feature is supported when using Markdown, Pongo2 or Amber, and is useful to get an instant preview when developing.

The JSX to JavaScript (ECMAscript) transpiler is built-in.

Redis is fast, scalable and offers good [data persistence](https://redis.io/topics/persistence). This should be the preferred backend.

Bolt is a [pure key/value store](https://github.com/coreos/bbolt), written in Go. It makes it easy to run Algernon without having to set up a database host first.
MariaDB/MySQL support is included because of its widespread availability.

PostgreSQL is a solid and fast database that is also supported.

Screenshots
-----------

<img src="https://raw.github.com/xyproto/algernon/main/img/algernon_markdown.png">

*Markdown can easily be styled with Sass or GCSS.*

---

<img src="https://raw.github.com/xyproto/algernon/main/img/algernon_lua_error.png">

*This is how errors in Lua scripts are handled, when Debug mode is enabled.*

---

<img src="https://raw.github.com/xyproto/algernon/main/img/algernon_threejs.png">

*One of the poems of Algernon Charles Swinburne, with three rotating tori in the background.*
*Uses CSS3 for the Gaussian blur and [three.js](https://threejs.org) for the 3D graphics.*

---

<img src="https://raw.github.com/xyproto/algernon/main/img/prettify.png">

*Screenshot of the <strong>prettify</strong> sample. Served from a single Lua script.*

---

<img src="https://raw.github.com/xyproto/algernon/main/img/algernon_react.png">

*JSX transforms are built-in. Using [React](https://facebook.github.io/react/) together with Algernon is easy.*

Samples
-------

The sample collection can be downloaded from the `samples` directory in this repository, or here: [samplepack.zip](https://algernon.roboticoverlords.org/samplepack.zip).


Getting started
---------------

##### Run Algernon in "dev" mode

This enables debug mode, uses the internal Bolt database, uses regular HTTP instead of HTTPS+HTTP/2 and enables caching for all files except: Pongo2, Amber, Lua, Teal, Sass, GCSS, Markdown and JSX.

* `algernon -e`

Then try creating an `index.lua` file with `print("Hello, World!")` and visit the served web page in a browser.

##### Enable HTTP/2 in the browser (for older browsers)

* Chrome: go to `chrome://flags/#enable-spdy4`, enable, save and restart the browser.
* Firefox: go to `about:config`, set `network.http.spdy.enabled.http2draft` to `true`. You might need the nightly version of Firefox.

##### Configure the required ports for local use

* You may need to change the firewall settings for port 3000, if you wish to use the default port for exploring the samples.
* For the auto-refresh feature to work, port 5553 must be available (or another host/port of your choosing, if configured otherwise).

##### Prepare for running the samples

    git clone https://github.com/xyproto/algernon
    make -C algernon

##### Launch the "welcome" page

* Run `./welcome.sh` to start serving the "welcome" sample.
* Visit `http://localhost:3000/`

##### Create your own Algernon application, for regular HTTP

* `mkdir mypage`
* `cd mypage`
* Create a file named `index.lua`, with the following contents:
  `print("Hello, Algernon")`
* Start `algernon --httponly --autorefresh`.
* Visit `http://localhost:3000/`.
* Edit `index.lua` and refresh the browser to see the new result.
* If there were errors, the page will automatically refresh when `index.lua` is changed.
* Markdown, Pongo2 and Amber pages will also refresh automatically, as long as `-autorefresh` is used.

##### Create your own Algernon application, for HTTP/2 + HTTPS

* `mkdir mypage`
* `cd mypage`
* Create a file named `index.lua`, with the following contents:
  `print("Hello, Algernon")`
* Create a self-signed certificate, just for testing:
 * `openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 3000 -nodes`
 * Press return at all the prompts, but enter `localhost` at *Common Name*.
 * For production, store the keys in a directory with as strict permissions as possible, then specify them with the `--cert` and `--key` flags.
* Start `algernon`.
* Visit `https://localhost:3000/`.
* If you have not imported the certificates into the browser, nor used certificates that are signed by trusted certificate authorities, perform the necessary clicks to confirm that you wish to visit this page.
* Edit `index.lua` and refresh the browser to see the result (or a Lua error message, if the script had a problem).


Basic Lua functions
-------------------

~~~c
// Return the version string for the server.
version() -> string

// Sleep the given number of seconds (can be a float).
sleep(number)

// Log the given strings as information. Takes a variable number of strings.
log(...)

// Log the given strings as a warning. Takes a variable number of strings.
warn(...)

// Log the given strings as an error. Takes a variable number of strings.
err(...)

// Return the number of nanoseconds from 1970 ("Unix time")
unixnano() -> number

// Convert Markdown to HTML
markdown(string) -> string

// Return the directory where the REPL or script is running. If a filename (optional) is given, then the path to where the script is running, joined with a path separator and the given filename, is returned.
scriptdir([string]) -> string

// Return the directory where the server is running. If a filename (optional) is given, then the path to where the server is running, joined with a path separator and the given filename, is returned.
serverdir([string]) -> string
~~~


Lua functions for handling requests
-----------------------------------

~~~c
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

// Return the HTTP body in the request (will only read the body once, since it's streamed).
body() -> string

// Set a HTTP status code (like 200 or 404). Must be used before other functions that writes to the client!
status(number)

// Set a HTTP status code and output a message (optional).
error(number[, string])

// Serve a file that exists in the same directory as the script. Takes a filename.
serve(string)

// Serve a Pongo2 template file, with an optional table with template key/values.
serve2(string[, table)

// Return the rendered contents of a file that exists in the same directory as the script. Takes a filename.
render(string) -> string

// Return a table with keys and values as given in a posted form, or as given in the URL.
formdata() -> table

// Return a table with keys and values as given in the request URL, or in the given URL (`/some/page?x=7` makes the key `x` with the value `7` available).
urldata([string]) -> table

// Redirect to an absolute or relative URL. May take an HTTP status code that will be used when redirecting.
redirect(string[, number])

// Permanent redirect to an absolute or relative URL. Uses status code 302.
permanent_redirect(string)

// Send "Connection: close" as a header to the client, flush the body and also
// stop Lua functions from writing more data to the HTTP body.
close()

// Transmit what has been outputted so far, to the client.
flush()
~~~


Lua functions for formatted output
----------------------------------

~~~c
// Output rendered Markdown to the browser/client. The given text is converted from Markdown to HTML. Takes a variable number of strings.
mprint(...)

// Output rendered Amber to the browser/client. The given text is converted from Amber to HTML. Takes a variable number of strings.
aprint(...)

// Output rendered GCSS to the browser/client. The given text is converted from GCSS to CSS. Takes a variable number of strings.
gprint(...)

// Output rendered HyperApp JSX to the browser/client. The given text is converted from JSX to JavaScript. Takes a variable number of strings.
hprint(...)

// Output rendered React JSX to the browser/client. The given text is converted from JSX to JavaScript. Takes a variable number of strings.
jprint(...)

// Output rendered HTML to the browser/client. The given text is converted from Pongo2 to HTML. The first argument is the Pongo2 template and the second argument is a table. The keys in the table can be referred to in the template.
poprint(string[, table])

// Output a simple HTML page with a message, title and theme.
// The title and theme are optional.
msgpage(string[, string][, string])
~~~


Lua functions related to JSON
-----------------------------

Tips:

* Use `JFile(`*filename*`)` to use or store a JSON document in the same directory as the Lua script.
* A JSON path is on the form `x.mapkey.listname[2].mapkey`, where `[`, `]` and `.` have special meaning. It can be used for pinpointing a specific place within a JSON document. It's a bit like a simple version of XPath, but for JSON.
* Use `tostring(userdata)` to fetch the JSON string from the JFile object.

~~~c
// Use, or create, a JSON document/file.
JFile(filename) -> userdata

// Takes a JSON path. Returns a string value, or an empty string.
jfile:getstring(string) -> string

// Takes a JSON path. Returns a JNode or nil.
jfile:getnode(string) -> userdata

// Takes a JSON path. Returns a value or nil.
jfile:get(string) -> value

// Takes a JSON path (optional) and JSON data to be added to the list.
// The JSON path must point to a list, if given, unless the JSON file is empty.
// "x" is the default JSON path. Returns true on success.
jfile:add([string, ]string) -> bool

// Take a JSON path and a string value. Changes the entry. Returns true on success.
jfile:set(string, string) -> bool

// Remove a key in a map. Takes a JSON path, returns true on success.
jfile:delkey(string) -> bool

// Convert a Lua table, where keys are strings and values are strings or numbers, to JSON.
// Takes an optional number of spaces to indent the JSON data.
// (Note that keys in JSON maps are always strings, ref. the JSON standard).
json(table[, number]) -> string

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
// Authorization header field.
jnode:POST(string[, string]) -> string

// Alias for jnode:POST
jnode:send(string[, string]) -> string

// Same as jnode:POST, but sends HTTP PUT instead.
jnode:PUT(string[, string]) -> string

// Fetches JSON over HTTP given an URL that starts with http or https.
// The JSON data is placed in the JNode. Returns the HTTP status code as a string.
jnode:GET(string) -> string

// Alias for jnode:GET
jnode:receive(string) -> string

// Convert from a simple Lua table to a JSON string
JSON(table) -> string
~~~

Lua functions for making HTTP requests
--------------------------------------

Quick example: `GET("http://ix.io/1FTw")`

~~~c
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
// URL parameters, then an optional table with HTTP headers.
hc:Get(string, [table], [table]) -> string

// Perform a HTTP POST request. It's the same arguments as for `Get`, except
// the fourth optional argument is the POST body.
hc:Post(string, [table], [table], [string]) -> string

// Like `Get`, except the first argument is the HTTP method (like "PUT")
hc:Do(string, string, [table], [table]) -> string

// Shorthand for HTTPClient():Get()
GET(string, [table], [table]) -> string

// Shorthand for HTTPClient():Post()
POST(string, [table], [table], [string]) -> string

// Shorthand for HTTPClient():Do()
DO(string, string, [table], [table]) -> string
~~~



Lua functions for plugins
-------------------------

~~~c
// Load a plugin given the path to an executable. Returns true on success. Will return the plugin help text if called on the Lua prompt.
// Pass in true as the second argument to keep it running.
Plugin(string, [bool])

// Returns the Lua code as returned by the Lua.Code function in the plugin, given a plugin path. May return an empty string.
// Pass in true as the second argument to keep it running.
PluginCode(string, [bool]) -> string

// Takes a plugin path, function name and arguments. Returns an empty string if the function call fails, or the results as a JSON string if successful.
CallPlugin(string, string, ...) -> string
~~~


Lua functions for code libraries
--------------------------------

These functions can be used in combination with the plugin functions for storing Lua code returned by plugins when serverconf.lua is loaded, then retrieve the Lua code later, when handling requests. The code is stored in the database.

~~~c
// Create or uses a code library object. Optionally takes a data structure name as the first parameter.
CodeLib([string]) -> userdata

// Given a namespace and Lua code, add the given code to the namespace. Returns true on success.
codelib:add(string, string) -> bool

// Given a namespace and Lua code, set the given code as the only code in the namespace. Returns true on success.
codelib:set(string, string) -> bool

// Given a namespace, return Lua code, or an empty string.
codelib:get(string) -> string

// Import (eval) code from the given namespace into the current Lua state. Returns true on success.
codelib:import(string) -> bool

// Completely clear the code library. Returns true on success.
codelib:clear() -> bool
~~~


Lua functions for file uploads
------------------------------

~~~c
// Creates a file upload object. Takes a form ID (from a POST request) as the first parameter.
// Takes an optional maximum upload size (in MiB) as the second parameter.
// Returns nil and an error string on failure, or userdata and an empty string on success.
UploadedFile(string[, number]) -> userdata, string

// Return the uploaded filename, as specified by the client
uploadedfile:filename() -> string

// Return the size of the data that has been received
uploadedfile:size() -> number

// Return the mime type of the uploaded file, as specified by the client
uploadedfile:mimetype() -> string

// Return the full textual content of the uploaded file
uploadedfile:content() -> string

// Save the uploaded data locally. Takes an optional filename. Returns true on success.
uploadedfile:save([string]) -> bool

// Save the uploaded data as the client-provided filename, in the specified directory.
// Takes a relative or absolute path. Returns true on success.
uploadedfile:savein(string)  -> bool
~~~


Lua functions for the file cache
--------------------------------

~~~c
// Return information about the file cache.
CacheInfo() -> string

// Clear the file cache.
ClearCache()

// Load a file into the cache, returns true on success.
preload(string) -> bool
~~~

Lua functions for data structures
---------------------------------

##### Set

~~~c
// Get or create a database-backed Set (takes a name, returns a set object)
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

// Remove the set itself. Returns true on success.
set:remove() -> bool

// Clear the set
set:clear() -> bool
~~~

##### List

~~~c
// Get or create a database-backed List (takes a name, returns a list object)
List(string) -> userdata

// Add an element to the list
list:add(string)

// Get all members of the list
list:getall() -> table

// Get the last element of the list
// The returned value can be empty
list:getlast() -> string

// Get the N last elements of the list
list:getlastn(number) -> table

// Remove the list itself. Returns true on success.
list:remove() -> bool

// Clear the list. Returns true on success.
list:clear() -> bool

// Return all list elements (expected to be JSON strings) as a JSON list
list:json() -> string
~~~

##### HashMap

~~~c
// Get or create a database-backed HashMap (takes a name, returns a hash map object)
HashMap(string) -> userdata

// For a given element id (for instance a user id), set a key
// (for instance "password") and a value.
// Returns true on success.
hash:set(string, string, string) -> bool

// For a given element id (for instance a user id), and a key
// (for instance "password"), return a value.
// Returns a value only if they key was found and if there were no errors.
hash:get(string, string) -> string

// For a given element id (for instance a user id), and a key
// (for instance "password"), check if the key exists in the hash map.
// Returns true only if it exists and there were no errors.
hash:has(string, string) -> bool

// For a given element id (for instance a user id), check if it exists.
// Returns true only if it exists and there were no errors.
hash:exists(string) -> bool

// Get all keys of the hash map
hash:getall() -> table

// Remove a key for an entry in a hash map
// (for instance the email field for a user)
// Returns true on success
hash:delkey(string, string) -> bool

// Remove an element (for instance a user)
// Returns true on success
hash:del(string) -> bool

// Remove the hash map itself. Returns true on success.
hash:remove() -> bool

// Clear the hash map. Returns true on success.
hash:clear() -> bool
~~~

##### KeyValue

~~~c
// Get or create a database-backed KeyValue collection (takes a name, returns a key/value object)
KeyValue(string) -> userdata

// Set a key and value. Returns true on success.
kv:set(string, string) -> bool

// Takes a key, returns a value.
// Returns an empty string if the function fails.
kv:get(string) -> string

// Takes a key, returns the value+1.
// Creates a key/value and returns "1" if it did not already exist.
// Returns an empty string if the function fails.
kv:inc(string) -> string

// Remove a key. Returns true on success.
kv:del(string) -> bool

// Remove the KeyValue itself. Returns true on success.
kv:remove() -> bool

// Clear the KeyValue. Returns true on success.
kv:clear() -> bool
~~~

Lua functions for external databases
------------------------------------

~~~c
// Query a PostgreSQL database with a SQL query and a connection string
PQ([string], [string]) -> table
~~~

The default connection string is `host=localhost port=5432 user=postgres dbname=test sslmode=disable` and the default SQL query is `SELECT version()`. Database connections are re-used if they still answer to `.Ping()`, for the same connection string.

~~~c
// Query a MSSQL database with SQL, a connection string, and a parameter table
MSSQL([string], [string], [table]) -> table
~~~

- The default connection string is `server=localhost;user=sa;password=Password123,port=1433` and the default SQL query is `"SELECT @@VERSION`. Database connections are re-used if they still answer to `.Ping()`, for the same connection string.
- If the param table is numerically indexed, positional placeholders are expected: `MSSQL("SELECT * FROM users WHERE first = @p1 AND last = @p2", conn, {"John", "Smith"})`
- If the param table is keyed with strings, named placeholders are expected: `MSSQL("SELECT * FROM users WHERE first = @first AND last = @last", conn, {first = "John", last = "Smith"})`


Lua functions for handling users and permissions
------------------------------------------------

~~~c
// Check if the current user has "user" rights
UserRights() -> bool

// Check if the given username exists (does not look at the list of unconfirmed users)
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

// Store the username in a cookie, returns true on success
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

// Add a user to the list of unconfirmed users
// Takes a username and a confirmation code
// Remember to also add a user, when registering new users.
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

// Log out a user, on the server (which is enough)
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

// Get the current server-wide cookie secret. This is used when setting
// and getting browser cookies when users log in.
CookieSecret() -> string

// Set the current server-side cookie secret. This is used when setting
// and getting browser cookies when users log in. Using the same secret
// makes browser cookies usable across server restarts.
SetCookieSecret(string)

// Get the current password hashing algorithm (bcrypt, bcrypt+ or sha256)
PasswordAlgo() -> string

// Set the current password hashing algorithm (bcrypt, bcrypt+ or sha256)
// ‘bcrypt+‘ accepts bcrypt or sha256 for old passwords, but will only use
// bcrypt for new passwords.
SetPasswordAlgo(string)

// Hash the password
// Takes a username and password (username can be used for salting sha256)
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

// Mark a user as confirmed, returns true on success
// Takes a confirmation code
ConfirmUserByConfirmationCode(string) -> bool

// Set the minimum confirmation code length
// Takes the minimum number of characters
SetMinimumConfirmationCodeLength(number)

// Generates a unique confirmation code, or an empty string
GenerateUniqueConfirmationCode() -> string
~~~


Lua functions that are available for server configuration files
---------------------------------------------------------------

~~~c
// Set the default address for the server on the form [host][:port].
// May be useful in Algernon application bundles (.alg or .zip files).
SetAddr(string)

// Reset the URL prefixes and make everything *public*.
ClearPermissions()

// Add an URL prefix that will have *admin* rights.
AddAdminPrefix(string)

// Add a reverse proxy given a path prefix and an endpoint URL
// For example: "/api" and "http://localhost:8080"
AddReverseProxy(string, string)

// Add an URL prefix that will have *user* rights.
AddUserPrefix(string)

// Provide a lua function that will be used as the permission denied handler.
DenyHandler(function)

// Return a string with various server information.
ServerInfo() -> string

// Direct the logging to the given filename. If the filename is an empty
// string, direct logging to stderr. Returns true on success.
LogTo(string) -> bool

// Returns the version string for the server.
version() -> string

// Logs the given strings as INFO. Takes a variable number of strings.
log(...)

// Logs the given strings as WARN. Takes a variable number of strings.
warn(...)

// Logs the given string as ERROR. Takes a variable number of strings.
err(...)

// Provide a lua function that will be run once, when the server is ready to start serving.
OnReady(function)

// Use a Lua file for setting up HTTP handlers instead of using the directory structure.
ServerFile(string) -> bool

// Serve files from this directory.
ServerDir(string) -> bool

// Get the cookie secret from the server configuration.
CookieSecret() -> string

// Set the cookie secret that will be used when setting and getting browser cookies.
SetCookieSecret(string)
~~~

Functions that are only available for Lua server files
------------------------------------------------------

This function is only available when a Lua script is used instead of a server directory, or from Lua files that are specified with the `ServerFile` function in the server configuration.

~~~c
// Given an URL path prefix (like "/") and a Lua function, set up an HTTP handler.
// The given Lua function should take no arguments, but can use all the Lua functions for handling requests, like `content` and `print`.
handle(string, function)

// Given an URL prefix (like "/") and a directory, serve the files and directories.
servedir(string, string)
~~~

Commands that are only available in the REPL
--------------------------------------------

* `help` displays a syntax highlighted overview of most functions.
* `webhelp` displays a syntax highlighted overview of functions related to handling requests.
* `confighelp` displays a syntax highlighted overview of functions related to server configuration.

Extra Lua functions
-------------------

~~~c
// Pretty print. Outputs the values in, or a description of, the given Lua value(s).
pprint(...)

// Takes a Python filename, executes the script with the `python` binary in the Path.
// Returns the output as a Lua table, where each line is an entry.
py(string) -> table

// Takes one or more system commands (possibly separated by `;`) and runs them.
// Returns the output lines as a table.
run(string) -> table

// Lists the keys and values of a Lua table. Returns a string.
// Lists the contents of the global namespace `_G` if no arguments are given.
dir([table]) -> string
~~~

Markdown
--------

Algernon can be used as a quick Markdown viewer with the `-m` flag.

Try `algernon -m README.md` to view `README.md` in the browser, serving the file once on a port >3000.

In addition to the regular Markdown syntax, Algernon supports setting the page title and syntax highlight style with a header comment like this at the top of a Markdown file:

    <!--
    title: Page title
    theme: dark
    code_style: lovelace
    replace_with_theme: default_theme
    -->

Code is highlighted with [highlight.js](https://highlightjs.org/) and [several styles](https://highlightjs.org/static/demo/) are available.

The string that follows `replace_with_theme` will be used for replacing the current theme string (like `dark`) with the given string. This makes it possible to use one image (like `logo_default_theme.png`) for one theme and another image (`logo_dark.png`) for the dark theme.

The theme can be `light`, `dark`, `redbox`, `bw`, `github`, `wing`, `material`, `neon`, `default`, `werc` or a path to a CSS file. Or `style.gcss` can exist in the same directory.

An overview of available syntax highlighting styles can be found at the [Chroma Style Gallery](https://xyproto.github.io/splash/docs/).


HTTPS certificates with Let's Encrypt and Algernon
--------------------------------------------------

Follow the guide at [certbot.eff.org](https://certbot.eff.org/) for the "None of the above" web server, then start `algernon` with `--cert=/etc/letsencrypt/live/mydomain.space/cert.pem --key=/etc/letsencrypt/live/mydomain.space/privkey.pem` where `mydomain.space` is replaced with your own domain name.

First make Algernon serve a directory for the domain, like `/srv/mydomain.space`, then use that as the webroot when configuring `certbot` with the `certbot certonly` command.

Remember to set up a cron-job or something similar to run `certbot renew` every once in a while (every 12 hours is suggested by [certbot.eff.org](https://certbot.eff.org/)). Also remember to restart the algernon service after updating the certificates. A way to refresh the certificates without restarting Algernon will be implemented in the future.


Releases
--------

* [Arch Linux package](https://aur.archlinux.org/packages/algernon) in the AUR.
* [Windows executable](https://github.com/xyproto/algernon/releases/tag/v1.0-win8-64).
* [macOS homebrew package](https://raw.githubusercontent.com/xyproto/algernon/main/system/homebrew/algernon.rb)
* [Algernon Tray Launcher for macOS, in App Store](https://itunes.apple.com/no/app/algernon-server/id1030394926?l=nb&mt=12)
* Source releases are tagged with a version number at release.


Requirements
------------

* `go 1.21` or later is a requirement for building Algernon.
* For `go 1.10`, `1.11`, `1.12`, `1.13`, `1.14`, '1.15`, `1.16` + `gcc-go <10` version `1.12.7` of Algernon is the last supported version.

Access logs
-----------

Can log to a Combined Log Format access log with the `--accesslog` flag. This works nicely together with [goaccess](https://goaccess.io/).

### Example usage

Serve files in one directory:

    algernon --accesslog=access.log -x

Then visit the web page once, to create one entry in the access.log.

The wonderful [goaccess](https://goaccess.io) utility can then be used to view the access log, while it is being filled:

    goaccess --no-global-config --log-format=COMBINED access.log

If you have goaccess setup correctly, running goaccess without any flags should work too:

    goaccess access.log

`.alg` files
------------

`.alg` files are just renamed `.zip` files, that can be served by Algernon. There is an example application here: [wercstyle](https://github.com/xyproto/wercstyle).

Logo license
------------

Thanks to [Egon Elbre](https://twitter.com/egonelbre) for the two SVG drawings that I remixed into the current logo ([CC0](https://creativecommons.org/publicdomain/zero/1.0/) licensed).

Listening to port 80 without running as root
--------------------------------------------

For Linux:

    sudo setcap cap_net_bind_service=+ep /usr/bin/algernon

Other resources
---------------

* [Algernon on docker hub](https://hub.docker.com/r/xyproto/algernon/)

General information
-------------------

* Version: 1.15.3
* License: BSD-3
* Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;

Stargazers over time
--------------------

[![Stargazers over time](https://starchart.cc/xyproto/algernon.svg)](https://starchart.cc/xyproto/algernon)

<a href="https://algernon.roboticoverlords.org"><img alt="0-0" src="img/gophereyes.png" align="right"></a>

The jump in stargazers happened when Algernon reached the front page of Hacker News:

* [Self-Contained Pure-Go Web Server with Lua, MD, HTTP/2, QUIC, Redis Support](https://news.ycombinator.com/item?id=19578351)
