# Plans


Various
-------
- [ ] Close open file handles at shutdown
- [ ] Add a Lua function for shutting down the server gracefully.
- [ ] Adding an option for exiting after a file has been downloaded once.
- [ ] Make sure requests are closed properly after being served.
- [ ] Use https://github.com/sbinet/igo instead of readline.
- [ ] A way to load Lua libraries that are available online, like http://json.luaforge.net/
- [ ] A way to extend Algernon with Go
- [ ] Use the JSON code from https://github.com/layeh/gopher-json
- [ ] Create a utility for creating and running new projects, ala Meteor
- [ ] Caching for GCSS, Amber templates, JSX/Javascript and Markdown when production mode is enabled
- [ ] Chat example with websockets, modeled after https://github.com/knadh/niltalk.git
- [ ] Support for pretty URLs (/position/x/2/y/4)
- [ ] JSON templates
- [ ] A way to make new Lua functions available while the server is running, over the network. Perhaps by using microservices that can serve Lua code.
- [ ] Support for key/values in PostgreSQL as an alternative to Redis. Create dbp and permissiongres.
- [ ] Create a simple way for people that wish to host Algernon applications for other people. Applications as zip-files?
- [ ] MSI installer
- [ ] OS X package + homebrew
- [ ] deb/ppa


Server configuration
--------------------

- [ ] Prefer environment variables and flags over lua server configuration.
- [ ] Server setting for making pages reload automatically whenever a source file changes.
- [ ] Server setting for enable the compilation of templates.
- [ ] Server setting for enabling caching.
- [ ] Add a "quiet" flag.
- [ ] Add a way to run several configuration scripts at start.
- [ ] A way to recompile templates on command while the server is running.
- [ ] If no Redis server is found, start an internal Ledis database that runs in RAM (see https://github.com/siddontang/ledisdb/blob/master/cmd/ledis-server/main.go)


Additional security
-------------------

- [ ] Rate limiting
- [ ] Option to disable directory listings
- [ ] Option to only allow whitelisted URL prefixes
- [ ] Functions for adding URL prefixes to the whitelist
- [ ] OAuth 1
- [ ] OAuth 2
- [ ] HTTP Basic Auth using the permissions2 usernames and passwords, for selected URL prefixes. Use code from "scoreserver ".
- [ ] The ability to set headers and do HTTP Basic Auth manually.


Examples
--------

- [ ] Port [niltalk](https://github.com/knadh/niltalk) to Algernon, in a separate repository.


Logging
-------

- [ ] Add configurable log hooks for the systems logrus supports. See: https://github.com/Sirupsen/logrus
- [ ] A separate debug webserver / control panel running on a different port.
      For displaying stats, access logs, break-in attempts and errors in the code.


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

- [ ] Homewbrew / OS X.


Unusual features
----------------

- [ ] A function for specifying png images by using ` `, `-` and `*` for pixels inside a `[[``]]` block, while specifying a main color. This can be used as an alternative way to serve favicon.ico files or specify icon graphics. Same thing could be used for svg, but by specifying numbered vertices in a polygon. Update: Someone else has made a format for this! https://github.com/cparnot/ASCIImage
- [ ] Find a way to set up a server that can add functions to remote LState objects on the fly, in a safe way. Perhaps by using the gob format.


Various
-------

- [ ] Consider using the path/filepath package for walking directories
- [ ] Add editor syntax highlight files


Maybe
-----

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
