# Plans

Priority
--------
- [ ] Python plugin example
- [ ] Dockerfile
- [ ] Installation instructions when installing with docker
- [ ] Add a lua function for removing all entries without a hit in the cache
- [ ] Add a lua function for running a lua function periodically
- [ ] Homebrew package for OS X
- [ ] Installation instructions when installing from source
- [ ] Installation instructions when installing from binary
- [ ] Functions for registering a route and a lua function as a HTTP handler (in serverconf.lua)
- [ ] Make it easy to host Algernon applications for other people
- [ ] User management interface
- [ ] Use a struct for the configuration variables

Go / go vet / go lint
---------------------
- [ ] Two identical lines in a row that is the same assignment should be an error

Various
-------
- [ ] Add a cache mode for caching binary files only
- [ ] Installer for OS X (pkg)
- [ ] Installer for Windows (MSI)
- [ ] Use golang/pkg/net/rpc/#Client.Go for calling plugins asynchronously.
      Let Lua provide a callback function.
- [ ] pprint should output text to the browser when not running in the repl
- [ ] Chat example with websockets, modeled after https://github.com/knadh/niltalk.git
- [ ] Support for pretty URLs and/or routing in serverconf.lua (/position/x/2/y/4)
- [/] Use some of the tricks from go-bootstrap.io
- [ ] Downloading and uploading files
- [ ] Add a Lua function ForEach that takes a data structure and a function that takes a key and a value.
- [ ] Use https://github.com/sbinet/igo instead of readline.
- [ ] Use the JSON code from https://github.com/layeh/gopher-json
- [ ] Consider using MegaJSON
- [ ] Create a utility for creating and running new projects, ala Meteor
- [ ] Caching for GCSS, Amber templates, JSX/Javascript and Markdown when production mode is enabled
- [ ] MSI installer
- [ ] deb/ppa
- [ ] Marshalling and unmarshalling from Lua tables to and from JSON, regardless of LValue


Events
------

- [ ] A better 404 page not found page for users visiting "/"
- [ ] Consider using channels in a more clever way, to avoid sleeping.
      Possibly by sending channels over channels.
- [ ] Consider only listening for changes after a file has been visited, then
      stop watching it after a while.
- [ ] Use a regexp or a JavaScript minification package instead of replacing strings in insertAutoRefresh
- [ ] In genFileChangeEvents, check for CloseNotify, for more graceful timeouts


Server configuration
--------------------

- [ ] Add a "quiet" flag.
- [ ] Prefer environment variables and flags over lua server configuration.
- [ ] Server setting for making pages reload automatically whenever a source file changes.
- [ ] Server setting for enable the compilation of templates.
- [ ] Server setting for enabling caching.
- [ ] A way to recompile templates on command while the server is running.

REPL
----
- [ ] See if a package related to gopher-lua can do the same as the pprint function
- [ ] If so, use the same functionality when converting from Lua tables to JSON

Plugins
-------
- [ ] Unmarshal the CallPlugin reply into appropriate Lua structures instead of returning a JSON string

Additional security
-------------------

- [X] Rate limiting
- [ ] Option to disable directory listings
- [ ] Option to only allow whitelisted URL prefixes
- [ ] Functions for adding URL prefixes to the whitelist
- [ ] OAuth 1
- [ ] OAuth 2
- [ ] HTTP Basic Auth using the permissions2 usernames and passwords, for selected URL prefixes. Use code from "scoreserver".
- [ ] The ability to set headers and do HTTP Basic Auth manually.
- [ ] Check if "*" or the server host should be used as parameter to the EventServer function


Examples
--------

- [ ] Port [niltalk](https://github.com/knadh/niltalk) to Algernon, in a separate repository.


Logging
-------

- [ ] Add configurable log hooks for the systems logrus supports. See: https://github.com/Sirupsen/logrus
- [ ] A separate debug webserver / control panel running on a different port.
      For displaying stats, access logs, break-in attempts and errors in the code.
- [ ] Make sure to close the log file when the server shuts down


Console output
--------------

- [ ] Check the terminal capabilities and terminal width. Display a smaller logo if the width is smaller. Or no logo.


Documentation and samples
-------------------------

- [ ] Create a sample webpage where people can log in and chat.
- [ ] Create a TODOMVC sample application.
- [ ] Document possible Markdown keywords somewhere (in a separate document).
- [ ] Create a React application that uses Algernon as the backend as well.


Screenshots and graphics
------------------------
- [ ] The three.js sample
- [ ] Of one of the React samples


Debugging
---------

- [ ] Implement the debug and logging functionality.
- [ ] Add a lua function that adds a html header and footer, including auto-refresh (if enabled)


Authentication and authorization
--------------------------------

- [ ] Support HTTP basic auth.
- [ ] Support OAuth 1.


Lua
---

- [ ] Add a function for priting Lua tables
- [ ] Add a function for fetching a value from a table, or a blank string
- [ ] Add a function for sanitizing HTML, possibly with bluemonday
- [ ] A way to store and load functions to the database:
      register("namespace name", "function name", luafunction)
      luafunction = getfunction("namespace name", "function name")
      import("namespace name")
- [ ] A way to have several webhandlers in one Lua script. Look for a function name in index.lua if a subdirectory is not found.
- [ ] Find a good way to create a personal collection of Lua functions.
- [ ] Support the re-use of templates by introducing functions for compiling templates and executing, saving and loading compiled templates.
- [ ] Create an import function for importing online lua libraries.
- [ ] A way to use Lua libraries, for SQLite and PostgreSQL, for insance.
- [ ] Lua function for checking if a file exists.
- [ ] A way to make an interactive session in the browser.
- [ ] A way to load parts of a page asynchronously.
- [_] Lua function for reading the contents of a file in the script dir, but in a cached way. Timestamp, filename and data are stored in redis, if timestamp changes, data is re-read.
- [ ] A way to have external projects written in Go that can extend
      the Lua state by adding functions. Perhaps adding functions to
      the Lua State object by sending it packed over the network and
      then receiving the modified Lua State.
- [ ] Modules, Lua libraries, plugins and reuse of code.
- [ ] In runLuaString, check if L.Close() really is needed instead of luapool.Put(L)


Performance
-----------

- [ ] Minify CSS, JS and HTML (on by default, can be disabled)
- [ ] Compress pages
- [ ] Caching. This can be configured in the server configuration. Or in Redis. Must be possible to specify the cache size.
- [ ] A way to toggle which files and directories that should be cached, in Redis.
- [ ] Three different cache settings: not cached, cached until file timestamp changes, always use cache.
- [ ] Find a reliable way to measure serving speed and emulate users.


Packaging
---------

- [/] Homewbrew / OS X.


Unusual features
----------------

- [ ] A function for specifying png images by using ` `, `-` and `*` for pixels inside a `[[``]]` block, while specifying a main color. This can be used as an alternative way to serve favicon.ico files or specify icon graphics. Same thing could be used for svg, but by specifying numbered vertices in a polygon. Update: Someone else has made a format for this! https://github.com/cparnot/ASCIImage


Various
-------

- [ ] Consider using the path/filepath package for walking directories
- [ ] Add editor syntax highlight files


Maybe
-----

- [ ] A flag to store the Bolt database inside the given zip file?
- [ ] Keep all configuration settings in Redis. Use an external package for handling configuration.
- [ ] Add a flag for acting like a static file server, over HTTP, without using Redis. Perhaps --static.
- [ ] The first argument should be a directory or a .alg file, the rest should be regular flags.
      An alg file can be a zipped or tar xz-ed directory with a server.lua file and all needed files. A bit like a .war file.
- [ ] Support OAuth 2, as a client.
- [ ] Support OAuth 2, as a server.
- [ ] Support for the [onthefly](https://github.com/xyproto/onthefly) package, as a virtual DOM.
- [ ] Websockets? WebRTC? Three.js? Web components?
- [ ] Use the goroutine functionality provided by gopher-lua to provide "trigger functions" that sends 1 on a channel when the function triggers, perhaps when a file is changed. Combine this with javascript somehow to make it possible to change the parts of a page when a happens.
- [ ] User functions shared by many lua pages should not be placed in `app.lua`, nor in a place related to the server, but be imported where they are needed. Either by importing a lua file, by importing a lua file by url or by connecting to a Lua Function Server.
- [ ] Make it possible to toggle the pretty error view on or off in `server.lua`.
- [ ] Find a good way to store errors.
- [ ] Implement a page, with admin rights, that displays the last error together with the sourcecode, in a pretty way.
- [ ] Add a flag for specifying a different default set of URL prefixes with admin, user or public rights.
- [ ] Add a flag for detailed debug information at errors, or not.
- [ ] If a symbolic link to a directory is made, for instance /chat -> /data, then algernon should also apply user permissions to the symbolic link.
- [ ] Consider creating an alternative version that users permissionsql instead of permissions2
- [ ] Add a function for calling EVAL on the redis server, while sending Lua code to the server for evaluation.
- [ ] Re-run `server.lua` if it is changed. Restart the server if the addr or port is changed.
- [ ] Support SASS.
- [ ] Add a function tprint("file.tmpl", table) for github.com/unrolled/render.
- [ ] Add an option for exiting after any page has been visited once.
- [ ] simplegres and permissiongres, for PostgreSQL.
- [ ] Read zip files directly instead of decompressing when given as the first argument (downside: some Amber functions look for files in the same directory).
- [ ] Utilities to lint and package .alg archives.
- [ ] Add caching of compiled templates, before data is inserted.
- [ ] Vagrantfile
- [ ] Add a maximum file size limit when caching
- [ ] Whitelist and blacklist for which file extensions to cache
