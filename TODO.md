# Plans

Application configuration
-------------------------

- [ ] If a symbolic link to a directory is made, for instance /chat -> /data, then algernon should also apply user permissions to the symbolic link.
- [ ] If a file named "DEBUG" is present, debug mode and pretty error messages should be enabled. If not, debug messages should go to the server log. Add the symbolic directories to the permission2 URL Prefix lists, depending on if they are linking to a directory that is already in one of the lists, or not.


Flags
-----

- [ ] Handle flags and arguments with the flag package.
- [X] Add a flag for specifying a remote redis host.


Documentation and examples
--------------------------

- [ ] Installation instructions with `go get` in README.md.
- [ ] Create an example webpage where people can log in and chat.


Debugging
---------

- [ ] Add a lua function that makes the page reload whenever the lua file is changed.
- [ ] If pretty errors are turned on, the lua code together with the error message and line indicator should be shown in the browser.


Authentication and authorization
--------------------------------

- [ ] Support HTTP basic auth.
- [ ] Support OAuth 2.


Lua
---

- [ ] Create an import function for importing online lua libraries.
- [ ] Find a good way to create a personal collection of Lua functions.
- [ ] A way to use Lua libraries, for SQLite and PostgreSQL, for insance.
- [ ] Lua function for checking if a file exists.
- [ ] Lua function for reading the contents of a file.
- [ ] Add a function for calling EVAL on the redis server, while sending Lua code to the server for evaluation.
- [ ] A way to make an interactive session in the browser.
- [ ] A way to load parts of a page asynchronously.


Unusual features
------------

- [ ] Find a way to set up a server that can add functions to remote LState objects on the fly, in a safe way. Perhaps by using the gob format.
- [ ] A function for specifying png images by using ` `, `-` and `*` for pixels inside a `[[``]]` block, while specifying a main color. This can be used as an alternative way to serve favicon.ico files or specify icon graphics. Same thing could be used for svg, but by specifying numbered vertices in a polygon.


Platform support
----------------

- [X] Test on Linux
- [X] Test on OS X
- [ ] Test on Windows


Benchmarking
------------

- [ ] Find a good way to measure how long it takes to serve a page.


Maybe
-----

- [ ] Support templates. Add a function tprint("file.tmpl", table).
- [ ] Colored terminal output.
- [ ] Reading a `server.lua` file when starting the server, for configuring the permissions, URL prefixes and database hosts.
- [ ] Support for the [onthefly](https://github.com/xyproto/onthefly) package.
- [ ] Websockets? WebRTC? Three.js? Web components?
- [ ] Use the goroutine functionality provided by gopher-lua to provide "trigger functions" that sends 1 on a channel when the function triggers, perhaps when a file is changed. Combine this with javascript somehow to make it possible to change the parts of a page when a happens.
- [ ] Use a virtual DOM?
- [ ] Caching.
- [ ] Should be possible to have a file named `app.lua` that is only read and interpreted once, unless the file has changed. It should only be read when `index.lua` is accessed and it has changed since last time. Store the timestamps in memory, not in redis.
- [ ] Make it possible to toggle the debug flag in `app.lua`.
- [ ] Make it possible to set permission URL path prefixes in `app.lua`.
- [ ] User functions shared by many lua pages should not be placed in `app.lua`, nor in a place related to the server, but be imported where they are needed. Either by importing a lua file, by importing a lua file by url or by connecting to a Lua Function Server.
- [ ] Make it possible to toggle the pretty error view on or off in `app.lua`.
- [ ] Find a good way to store errors.
- [ ] Implement a page, with admin rights, that displays the last error together with the sourcecode, in a pretty way.
- [ ] Add a flag for specifying a different default set of URL prefixes with admin, user or public rights.
- [ ] Add a flag for detailed debug information at errors, or not.
