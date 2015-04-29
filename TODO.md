# Plans


Priority
--------

- [ ] Make cookies work when buffering http.RequestWriter in debug mode. Or disable buffering when cookies are involved. See if https://github.com/servemux/buffer handles cookies as well.
- [ ] A feature for automatically reloading the page whenever files change
- [ ] JSON templates
- [ ] index.jsx for styled react.js (https://facebook.github.io/react/)
- [ ] Caching. This can be configured in the server configuration. Or in Redis. Must be possible to specify the cache size.
- [ ] Chat example with websockets, modeled after https://github.com/knadh/niltalk.git


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

- [ ] Check the terminal capabilities and the terminal width.


Documentation and examples
--------------------------

- [ ] Create an example webpage where people can log in and chat.
- [ ] Create a TODOMVC example application.


Debugging
---------

- [ ] Automatic browser reload when served files are changed, for development.
- [ ] Implement the debug and logging functionality.
- [ ] Add a lua function that makes the page reload whenever the lua file is changed.
- [ ] If pretty errors are turned on, the lua code together with the error message and line indicator should be shown in the browser.
- [ ] If the server executable is named something with "debug", turn on debugging.


Authentication and authorization
--------------------------------

- [ ] Support HTTP basic auth.
- [ ] Support OAuth 1.


Lua
---

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

- [ ] Virtual DOM?
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
