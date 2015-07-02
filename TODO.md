# Plans

Priority
--------
- [ ] File upload.
- [ ] User management interface + web REPL + stats + logs + import/export data + .alg launcher.
- [ ] Dockerfile / containers.
- [ ] Access log that does not have to be written manually in Lua.
- [ ] Use a struct for the configuration variables.
- [ ] Handle .avi and other multimedia files better. Show a page for playing and downloading.


Go / go vet / go lint
---------------------
- [ ] Two identical lines in a row that is the same assignment should result in an error.
- [ ] Constant byte slices should be allowed.


Various
-------
- [ ] A way to serve different directories for different subdomains
- [ ] Cache os.Stat also when serving directory listings
- [ ] Create a screencast
- [ ] Implement https://github.com/labstack/echo/tree/master/examples as Algernon applications
- [ ] Support the LuaPage format (".lp", HTML with <% %> and <%= %> for Lua code)
- [ ] Look into github.com/jessevdk/go-flags/
- [ ] pprint should output text to the browser when not running in the repl
- [ ] web handlers should have access to setting up additional web handlers
- [ ] Visitor graph
- [ ] See if the HTTP headers from the client + country of origin + mouse movement patterns can become some sort of pseudo ID.
      Combine with a neural net. Can be used for storing non-critical data like prefered themes, font sizes etc.
- [ ] Add editor syntax highlight files.
- [ ] Support for pretty URLs and/or routing in serverconf.lua (/position/x/2/y/4).
- [ ] Commandline utilities for editing users, permissions, databases and Lua functions in databases.
- [ ] Add a lua function for removing all cache entries without a hit.
- [ ] Add a lua function for running a lua function periodically.
- [ ] Add a cache mode for caching binary files only
- [ ] Installer for OS X (pkg)
- [ ] MSI installer.
- [ ] deb/ppa
- [/] Use some of the tricks from go-bootstrap.io
- [ ] Consider using https://github.com/sbinet/igo instead of readline.
- [ ] Create a utility for creating and running new projects, ala Meteor.


Data control
------------
- [ ] Add simpleredis/simplebolt/simplemaria functions for exporting/importing data to JSON and offer these.


Events
------
- [ ] Better 404 page not found page for users visiting "/".
- [ ] Consider using channels in a more clever way, to completely avoid sleeping.
      Possibly by sending channels over channels.
- [ ] Consider only listening for changes after a file has been visited, then stop watching it after a while.
- [ ] Use a regexp or a JavaScript minification package instead of replacing strings in insertAutoRefresh.
- [ ] In genFileChangeEvents, check for CloseNotify, for more graceful timeouts.


Server configuration
--------------------
- [ ] Prefer environment variables and flags over lua server configuration.


Routing
-------
- [ ] Server("host:port", "/srv/http/somedirectory", "/var/log/algernon/logfile.log")
- [ ] Redirect("host/path:port", ":port/path")
- [ ] Rewrite("host:port", "host:port/path")
- [ ] RewritePrefix("www.", "")
- [ ] RewritePort("host", 443, 80)


REPL
----
- [ ] Make `help` and `dir` work a bit like in Python.
- [ ] Web REPL.


Plugins
-------
- [ ] Unmarshal the CallPlugin reply into appropriate Lua structures instead of returning a JSON string
- [ ] If a plugin ends with ".go", check if go is installed and run it with "go run" (if a binary of the same name has not been provided for the current platform).
- [ ] Add a function for loading all plugins in a "plugins" directory.


Additional security
-------------------
- [ ] Consider using https://github.com/unrolled/secure
- [ ] HTTP Basic Auth using the permissions2 usernames and passwords, for selected URL prefixes.
      Use code from "scoreserver".
- [ ] Check that HTTP reads not only times out, but has a deadline.
- [ ] Flag for disabling directory listings entirely.
- [ ] OAuth 1
- [ ] OAuth 2
- [ ] The ability to set headers and do HTTP Basic Auth manually.
- [ ] Check if "*" or the server host should be used as parameter to the EventServer function


Logging
-------
- [ ] A separate debug webserver / control panel running on a different port.
      For displaying stats, access logs, break-in attempts, errors in the code.
      Should also include an interactive REPL.


Console output
--------------
- [ ] Check the terminal capabilities and terminal width. Display a smaller logo if the width is smaller. Or no logo.


Documentation and samples
-------------------------
- [ ] Port [niltalk](https://github.com/knadh/niltalk) to Algernon, in a separate repository.
- [ ] Create a sample chat application.
- [ ] Create a sample TODOMVC application.
- [ ] Document possible Markdown keywords somewhere (in a separate document).
- [ ] Write a Lua library and use it in several web handlers.
- [ ] Make an application where .alg files can be uploaded and then hosted.


Lua
---
- [ ] Wrap JNode in the same way as JFile.
- [ ] Add a Lua function ForEach that takes a data structure and a function that takes a key and a value.
- [ ] Add a function for sanitizing HTML, possibly with bluemonday.
- [ ] Create an import function for importing online lua libraries. (Like `require`, but over http)
- [ ] In runLuaString, check if L.Close() really is needed instead of luapool.Put(L)
- [ ] Way to load parts of a page asynchronously (with gopher-lua channels?)
- [ ] Way to use Lua libraries for adding ie. SQLite support.


Performance
-----------
- [ ] Minify CSS, JS and HTML (enabled by default, can be disabled)
- [ ] Find a reliable way of measuring speed and emulating users. gor? https://github.com/buger/gor
- [ ] Cache complied templates, not only the final result.


Unusual features
----------------
- [ ] A function for specifying png images by using ` `, `-` and `*` for pixels inside a `[[``]]` block, while specifying a main color. This can be used as an alternative way to serve favicon.ico files or specify icon graphics. Same thing could be used for svg, but by specifying numbered vertices in a polygon. Update: Someone else has made a format for this! https://github.com/cparnot/ASCIImage


Maybe
-----
- [ ] Add configurable log hooks for the systems logrus supports. See: https://github.com/Sirupsen/logrus
- [ ] Use the path/filepath package for walking directories.
- [ ] Add a Lua function for outputting Lua tables to the client.
- [ ] Add a Lua function for fetching a value from a table, or a blank string.
- [ ] Add a Lua function for checking if a file exists.
- [ ] Mention the `jpath` package in the README.
- [ ] Support for plugins written in BF
- [ ] A flag to store the Bolt database inside the given zip file?
- [ ] Keep all configuration settings in Redis. Use an external package for handling configuration.
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
- [ ] Read zip files directly instead of decompressing when given as the first argument (downside: some Amber functions look for files in the same directory).
- [ ] Utilities to lint and package .alg archives.
- [ ] Add caching of compiled templates, before data is inserted.
- [ ] Vagrantfile
- [ ] Add a maximum file size limit when caching
- [ ] Whitelist and blacklist for which file extensions to cache
- [ ] Use golang/pkg/net/rpc/#Client.Go for calling plugins asynchronously. Let Lua provide a callback function.
- [ ] Configuration function for whitelisting URL prefixes.
- [ ] Functions for adding URL prefixes to the whitelist
- [ ] Lua function for reading the contents of a file in the script dir, but in a cached way.
      Timestamp, filename and data are stored in redis, if timestamp changes, data is re-read.
