# Plans

Priority 1
----------
- [ ] No external links for syntax highlighting in themes (no external dependencies in general). #17
- [ ] Make it possible to send the access log to the database.
- [ ] Output an access log in a [goaccess.io](https://goaccess.io) friendly format.
- [ ] Make it possible to send the error log to the database.
- [ ] Profile the startup process and try to make it even faster.
- [ ] Port the layout and concept of [werc](http://werc.cat-v.org/) to Algernon.  See also [gowerc](https://bitbucket.org/mischief/gowerc/src).
- [ ] Make most methods in [onthefly](https://github.com/xyproto/onthefly) available to Algernon/Lua.
- [ ] User management interface + web REPL + stats + logs + import/export data .alg launcher.
- [ ] Present directories with media files with a built-in page.
- [ ] Add an option for using brotly compression instead of gzip.
- [ ] Make the behavior per file extension or mime type configurable: "raw view", "pretty view" or "download"

Priority 2
----------
- [ ] Add a Markdown style similar to this one: https://hyperapp.glitch.me/style.css
- [ ] Add a Markdown style similar to this one: http://setconf.roboticoverlords.org/
- [ ] Use fasthttp when using regular HTTP:[switching to fasthttp](https://github.com/valyala/fasthttp#switching-from-nethttp-to-fasthttp).
- [ ] Parse options with [docopt](https://github.com/docopt/docopt.go) or [cli](https://github.com/urfave/cli).
- [ ] Also support a configuration file using [configparser](https://github.com/alyu/configparser), for port, host, keys etc.
- [ ] Add a flag for caching to the database backend instead of to memory.
- [ ] Add support for [Ghost](https://ghost.org) and/or [Hugo](https://github.com/gohugoio/hugoThemes) themes.
- [ ] Add support for the [Badger](https://blog.dgraph.io/post/badger-lmdb-boltdb/) database.
- [ ] Add support for gccgo, if Badger works better with gccgo than BoltDB.
- [ ] Create a web page for uploading, reviewing, previewing and downloading Algernon Applications.

Documentation/tutorials
-----------------------
- [ ] Add example for HyperApp + database usage
- [ ] Use `peek` for recording a series of videos.
- [ ] Look into using [slate](https://github.com/lord/slate) for documentation.
- [ ] Terminal recording demostrating creating a simple register+login site.
- [ ] Terminal recording demonstrating the Lua interpreter.
- [ ] Video tutorials and screencasts.
- [ ] Update the book to be more similar to the python Flask documentation.
- [ ] Create a docker image that comes with all the samples.
- [ ] Document what the "current directory" is for various Lua functions that deals with files.
- [ ] Document better the order of output calls when modifying the header to redirect.
- [ ] Document how to read JSON from one place and output processed data somewhere else.

Various
-------
- [ ] When requests are handled, spawn each switch/case as a Go routine. Benchmark to see if there is a difference.
- [ ] Write a module for caching that can cache chunks of files and stream files that does not fit in memory directly from disk.
- [ ] Add support for systemd reload, not just restart.
- [ ] Render JavaScript server-side by using [Goja](https://github.com/dop251/goja)
- [ ] Larger selection of built-in Markdown styles, with a flag for dumping them as a style.gcss, for easy modification. Or use a system directory for this.
- [ ] Use [cfilter](https://github.com/irfansharif/cfilter) for potentially faster cache lookups.
- [ ] Support [HAML](https://github.com/travissimon/ghaml)?
- [ ] Introduce a separate package for dealing with Lua pools, Lua states and
      adding custom functios to some Lua states. All without using mutexes.
- [ ] Support for websockets (port a small multiplayer game to test).
- [ ] Add support for Handlebars: [raymond](https://github.com/aymerick/raymond)
- [ ] Check behavior of ctrl-c/ctrl-d on OS X vs Linux.
- [ ] Server side support for [sw-delta](https://github.com/gmetais/sw-delta)
- [ ] Add a flag to minify all transmitted CSS/HTML/JS/JSON/SVG/XML files
      https://github.com/tdewolff/minify
- [ ] Draw inspiration from https://github.com/olebedev/go-starter-kit
- [ ] Draw inspiration from https://github.com/disintegration/bebop
- [ ] Provide a Lua sample/command for listing files and directories with dates and sizes.
- [ ] Find a way to redirect while preserving headers and/or use a mux package.
- [ ] Implement a documentation server that can convert files with pandoc.
- [ ] Make it easy to apply patches on the fly, when GET-ting the resulting file
- [ ] Built in support for running the Lua REPL in the browser (possibly by using "gotty", either as a package or wrapped in a script).
- [ ] Create a sample that is inspired by this design: http://codepen.io/KtorZ/pen/ZOzdqG
- [ ] Add Markdown themes from: https://github.com/mixu/markdown-styles
- [ ] Add a similar boilerplate as Jekyll to megaboilerplate.com
- [ ] Describe how to set up a system a bit similar to a wiki, but more lightweight, using git + git hooks + algernon.
- [ ] Add a flag for listing and selecting styles for Markdown and directory listings.
- [ ] Specify if rate limiting is per user/ip/handler
- [ ] Add a flag for serving with fasthttp: https://github.com/valyala/fasthttp
- [ ] Create alg2systemd-nspawn and alg2runc.
- [ ] Create a site generator for Algernon. Draw inspiration from http://nanoc.ws/doc/tutorial/
- [ ] Draw inspiration from https://lwan.ws/
- [ ] Check out https://github.com/peterh/liner
- [ ] Support SASS and HAML.
- [ ] Port Pastecat to Algernon (https://github.com/mvdan/pastecat)
- [ ] Argon2 support (https://godoc.org/github.com/magical/argon2)
- [ ] Add config Function for adding a directory listing title to a certain path regex (and/or a title.txt or common.md file).
- [ ] Add a lua function for presenting an executable as a web application, like gotty does. Create a password protected example application.
- [ ] Use a struct for the configuration variables.
- [ ] Web application for browsing the database.
- [ ] Document the case sensitivity or add case insensitivity support.
- [ ] Create a tool that pretends to upload a file of size 128 bytes
      (Content-Length), but continues to stream data. Test with Algernon.
- [ ] Automatic redirect from http to https, or the other way around.
- [ ] Lua plugin that is not via the database
- [ ] C plugin
- [ ] File upload while handling gzip
- [ ] Cache os.Stat also when serving directory listings
- [ ] Implement https://github.com/labstack/echo/tree/master/examples as Algernon applications
- [ ] Look into github.com/jessevdk/go-flags/
- [ ] pprint should output text to the browser when not running in the repl (or be disabled)
- [ ] Graph of visitors over time
- [ ] See if the HTTP headers from the client + country of origin + mouse
      movement patterns can become some sort of pseudo ID. Combine with a
      neural net. Can be used for storing non-critical data like prefered
      themes, font sizes etc. Time of day may also be an input.
- [ ] Add editor syntax highlight files.
- [ ] Support for pretty URLs and/or routing in serverconf.lua (/position/x/2/y/4).
- [ ] Commandline utilities for editing users, permissions, databases and Lua functions in databases.
- [ ] Add a lua function for running a lua function periodically.
- [ ] Add a cache mode for caching binary files only.
- [ ] MSI installer.
- [ ] deb/ppa
- [/] Use some of the tricks from go-bootstrap.io
- [ ] Consider using https://github.com/sbinet/igo instead of readline.
- [ ] Create a utility for creating and running new projects, ala Meteor.
- [ ] Add Lua functions for BSON and ION?
- [ ] Add Lua methods for sending JSON with a custom HTTP verb
- [ ] Add simpleredis/simplebolt/simplemaria functions for exporting/importing data to JSON and offer these.

Events
------
- [ ] Better 404 page not found page for users visiting "/".
- [ ] Consider only listening for changes after a file has been visited, then stop watching it after a while.
- [ ] Use a regexp or a JavaScript minification package instead of replacing strings in insertAutoRefresh.
- [ ] In genFileChangeEvents, check for CloseNotify for more graceful timeouts.

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

Plugins
-------
- [ ] Unmarshal the CallPlugin reply into appropriate Lua structures instead of returning a JSON string.
- [ ] If a plugin ends with ".go", check if go is installed and run it with "go run" (if a binary of the same name has not been provided for the current platform).
- [ ] Add a function for loading all plugins in a "plugins" directory.

Additional security
-------------------
- [ ] Consider using [secure](https://github.com/unrolled/secure).
- [ ] HTTP Basic Auth using the permissions2 usernames and passwords, for selected URL prefixes. Use code from "scoreserver".
- [ ] Check that HTTP reads not only times out, but has a deadline.
- [ ] Flag for disabling directory listings entirely.
- [ ] OAuth 1
- [ ] OAuth 2
- [ ] The ability to set headers and do HTTP Basic Auth manually.
- [ ] Check if `*` or the server host should be used as parameter to the EventServer function
- [ ] Implement a warning when using cookies over regular HTTP.

Logging
-------
- [ ] A separate debug webserver / control panel running on a different port.
      For displaying stats, access logs, break-in attempts, errors in the code.
      Should also include an interactive REPL.

Console output
--------------
- [ ] Check the terminal capabilities and terminal width. Display a smaller
      logo if the width is smaller. Or no logo.

Documentation and samples
-------------------------
- [ ] Create a sample TODOMVC application.
- [ ] Port [niltalk](https://github.com/knadh/niltalk) to Algernon,
      in a separate repository.
- [ ] Create a sample chat application.
- [ ] Document possible Markdown keywords somewhere (in a separate document).
- [ ] Write a Lua library and use it in several web handlers.
- [ ] Make an application where .alg files can be uploaded and then hosted.
- [ ] Add a sample for bricklayer https://github.com/ademilter/bricklayer

Lua
---
- [ ] Add a function named "sort" for quickly sorting tables by key or by value, numerical or lexical.
- [ ] Add a Lua function ForEach that takes a data structure and a function
      that takes a key and a value.
- [ ] Wrap JNode in the same way as JFile.
- [ ] Change the "JSON" function and create some sort of JSON object that returns the string by default.
- [ ] Add a function for sanitizing HTML, possibly with bluemonday.
- [ ] Create an import function for importing online lua libraries.
      (Like `require`, but over http). (possibly luarocks packages).
- [ ] In runLuaString, check if L.Close() really is needed instead of
      luapool.Put(L)
- [ ] Way to load parts of a page asynchronously (with gopher-lua channels?)
- [ ] Way to use Lua libraries for adding ie. SQLite support.


Performance
-----------
- [ ] Minify CSS, JS and HTML (as enabled by default, but can be disabled)
- [ ] Find a reliable way of measuring speed and emulating users.
      gor? https://github.com/buger/gor
- [ ] Cache compiled templates as well, not just the final result.


Unusual features
----------------
- [ ] A function for specifying png images by using ` `, `-` and `*` for
      pixels inside a `[[``]]` block, while specifying a main color. This can
      be used as an alternative way to serve favicon.ico files or specify icon
      graphics. Same thing could be used for svg, but by specifying numbered
      vertices in a polygon. Update: Someone else has made a format for this!
      https://github.com/cparnot/ASCIImage


Serving several domains
-----------------------
- [ ] HTTP/2 + HTTPS + certificates per subdomain (a parameter for a
      subdomain, when using the --domain parameter. Then only serve that
      directory with HTTPS for :443). (Can be solved by starting several
      instances of Algernon istead).

Maybe
-----
- [ ] Add support for both SASS and SCSS (Perhaps https://github.com/c9s/c6)
- [ ] Add configurable log hooks for the systems logrus supports.
      See: https://github.com/Sirupsen/logrus
- [ ] RethinkDB support.
- [ ] Use the path/filepath package for walking directories.
- [ ] Add a Lua function for outputting Lua tables to the client.
- [ ] Add a Lua function for fetching a value from a table, or a blank string.
- [ ] Add a Lua function for checking if a file exists.
- [ ] Mention the `jpath` package in the README.
- [ ] Support for plugins written in BF.
- [ ] A flag to store the Bolt database inside the given zip file?
- [ ] Keep all configuration settings in Redis. Use an external package for
      handling configuration.
- [ ] Support for the [onthefly](https://github.com/xyproto/onthefly) package,
      as a virtual DOM.
- [ ] WebRTC? Three.js? Web components?
- [ ] Use the goroutine functionality provided by gopher-lua to provide
      "trigger functions" that sends 1 on a channel when the function
      triggers, perhaps when a file is changed. Combine this with javascript
      somehow to make it possible to change the parts of a page when an
      event happens.
- [ ] User functions shared by many lua pages should not be placed in
      `app.lua`, nor in a place related to the server, but be imported where
      they are needed. Either by importing a lua file, by importing a lua
      file by url or by connecting to a Lua Function Server.
- [ ] Make it possible to toggle the pretty error view on or off in
      `serverconf.lua`, for temporary debugging.
- [ ] Find a good way to store errors.
- [ ] Implement a page, with admin rights, that displays the last error
      together with the sourcecode, in a pretty way.
- [ ] Add a flag for specifying a different default set of URL prefixes with
      admin, user or public rights.
- [ ] Add a flag for detailed debug information at errors, or not.
- [ ] If a symbolic link to a directory is made, for instance /chat -> /data,
      then algernon should also apply user permissions to the symbolic link.
- [ ] Add a function for calling EVAL on the redis server, while sending Lua
      code to the server for evaluation.
- [ ] Re-run the Lua server script if changed. Restart the server if the addr
      or port is changed.
- [ ] Add a function tprint("file.tmpl", table) for github.com/unrolled/render.
- [ ] Add an option for exiting after any page has been visited once.
- [X] simplehstore and permissionwrench, for PostgreSQL.
- [ ] Read zip files directly instead of decompressing when given as the
      first argument (downside: some Amber functions look for files in the
      same directory).
- [ ] Utilities to lint and package .alg archives.
- [ ] Vagrantfile
- [ ] Add a maximum file size limit when caching
- [ ] Whitelist and blacklist for which file extensions to cache
- [ ] Use golang/pkg/net/rpc/#Client.Go for calling plugins asynchronously.
      Let Lua provide a callback function.
- [ ] Configuration function for whitelisting URL prefixes.
- [ ] Functions for adding URL prefixes to the whitelist
- [ ] Lua function for reading the contents of a file in the script dir,
      but in a cached way. Timestamp, filename and data are stored in Redis,
      if timestamp changes, data is re-read.
- [ ] web handlers should have access to setting up additional web handlers
- [ ] Add a Lua function for removing all cache entries without a hit.
- [ ] Support the LuaPage format (".lp", HTML with <% %> and <%= %> for Lua code).
- [ ] Add Lua functions for HTTP PUT without using JSON? (for etcd, but might be a bad idea in the first place).
- [ ] Rewrite in C++17 and rename the project to "FnuFnu".

# Future

## Algernon 2

* Use Badger instead of BoltDB (if it really is just as stable and 30x faster).
* Use fasthttp from the start.
* Drop Amber and GCSS.
* Focus on templates, markdown and possibly microservices.
* Embed a different language than Lua, perhaps Anko.
* Support gccgo from the start.
* Aim for a very small and specific usage pattern and try to optimize for that.
