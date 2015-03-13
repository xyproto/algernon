# Plans


Flags
-----

* Handle flags and arguments with the flag package.
* Add a flag for specifying a remote redis host.
* Add a flag for specifying a different default set of URL prefixes with admin, user or public rights.
* Add a flag for detailed debug information at errors, or not.


Documentation and examples
--------------------------

* Installation instructions with `go get` in README.md.
* Create an example webpage where people can log in and chat.


Debugging
---------

* Add a lua function that makes the page reload whenever the lua file is changed.
* Implement a page, with admin rights, that displays the last error together with the sourcecode, in a pretty way.
* Find a good way to store the last error (system wide? per user?).
* Decide if Lua errors can be printed to the web page, or if logging to the console is better.


Authentication and authorization
--------------------------------

* Support HTTP basic auth.
* Support OAuth 2.


Lua
---

* Find a good way to create a personal collection of Lua functions.
* A way to use Lua libraries, for SQLite and PostgreSQL, for insance.
* Lua function for checking if a file exists.
* Lua function for reading the contents of a file.


Platform support
----------------

* Test on OS X and Windows


Maybe
-----

* Support for templates somehow.
* Colored terminal output.
* Reading a `server.lua` file when starting the server, for configuring the permissions, URL prefixes and database hosts.
* Support for the [onthefly](https://github.com/xyproto/onthefly) package.
* Websockets? WebRTC? Three.js? Web components?

