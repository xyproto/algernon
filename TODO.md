# Plans


Documentation and examples
--------------------------

* Installation instructions with `go get` in README.md.
* Create an example webpage where people can log in and chat.


Small tasks
-----------

* Find a better source URL for the prettyfy js file, in the prettify example.
* Handle flags and arguments with the flag package.


Authentication and authorization
--------------------------------

* Support HTTP basic auth.
* Support OAuth 2.


Lua
---

* Lua function for checking if a file exists.
* Lua function for reading the contents of a file.
* Find a good way to create a personal collection of Lua functions.
* A way to use Lua libraries, for SQLite and PostgreSQL, for insance.


Debugging
---------

* Implement a page, with admin rights, that displays the last error together with the sourcecode.
* Find a good way to store the last error (system wide? per user?).
* Decide if Lua errors can be printed to the web page, or if logging to the console is better.


Platform support
----------------

* Test on OS X and Windows


Maybe
-----

* Colored console output.
* Support for templates somehow.
* Support for the [onthefly](https://github.com/xyproto/onthefly) package.
* Websockets? WebRTC? Three.js? Web components?
