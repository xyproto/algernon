# Plans


Most important
--------------

- [ ] Make cookies work not only with HTTP, but also HTTPS+HTTP/2.
- [ ] Automatic browser reload when served files are changed, for development.
- [ ] Caching of templates.
- [ ] A smoother way to combine GCSS, Amber and Lua.
- [ ] Virtual DOM?
- [ ] A separate access log.
- [ ] Modules, Lua libraries, plugins and reuse of code.


Server configuration
--------------------

- [ ] Server setting for making pages reload automatically whenever a source file changes.
- [ ] Server setting for enable the compilation of templates.
- [ ] Server setting for enabling caching.
- [ ] Add a "quiet" flag.
- [ ] Add a way to run several configuration scripts at start.
- [ ] A way to recompile templates on command while the server is running.


Database
--------

- [ ] If no Redis server is found, use an internal Ledis database that runs in RAM.


CSS
---

- [ ] Support SASS.


Logging
-------

- [ ] Add configurable log hooks for the systems logrus supports. See: https://github.com/Sirupsen/logrus


Console output
--------------

- [ ] Check the terminal capabilities and the terminal width.


Documentation and examples
--------------------------

- [ ] Create an example webpage where people can log in and chat.
- [ ] Create a TODOMVC example application.


Debugging
---------

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
- [ ] Add a function tprint("file.tmpl", table) for github.com/unrolled/render.
- [ ] Create an import function for importing online lua libraries.
- [ ] A way to use Lua libraries, for SQLite and PostgreSQL, for insance.
- [ ] Lua function for checking if a file exists.
- [ ] Lua function for reading the contents of a file.
- [ ] A way to make an interactive session in the browser.
- [ ] A way to load parts of a page asynchronously.
- [ ] A way to discover which functions are used or not in scripts that don't use "eval-like" functions like `loadstring`.
- [ ] If a Lua script only use some functions, only expose the used functions.


Packaging
---------

- [ ] Homewbrew / OS X.


Unusual features
----------------

- [ ] Find a way to set up a server that can add functions to remote LState objects on the fly, in a safe way. Perhaps by using the gob format.
- [ ] A function for specifying png images by using ` `, `-` and `*` for pixels inside a `[[``]]` block, while specifying a main color. This can be used as an alternative way to serve favicon.ico files or specify icon graphics. Same thing could be used for svg, but by specifying numbered vertices in a polygon. Update: Someone else has made a format for this! https://github.com/cparnot/ASCIImage


Benchmarking
------------

- [ ] Find a reliable way to measure how long it takes to serve a page.


Maybe
-----

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
