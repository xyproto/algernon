# Plans

Priority 1
----------
- [ ] Resolve all issues in the issue tracker.
- [ ] Refresh all files, cache and certificates when receiving the USR1 signal.
- [ ] Add a way to reload the HTTPS certificates without restarting Algernon.
- [ ] Add a flag for redirecting all `http://` traffic to `https://`.
- [ ] Add a Go function for adding a Lua function that can handle websocket requests to `/ws`.
- [ ] Add a smoother way than `CodeLib()` to define site-wide Lua values.

Priority 2
----------

- [ ] Create a video like the one at [vim-livedown](https://github.com/shime/vim-livedown), that demonstrates live editing of Markdown.
- [ ] When `-m` is given, scan the given Markdown file for images that will also need to be served, then wait until those are served before exiting.
- [ ] Add support for [metatar](https://github.com/xyproto/metatar) in Lua, to be able to offer a whole Arch Linux package repository from just a single `.lua` file, and a collection of `PKGBUILD` files.
- [ ] Integrate [boltBrowser](https://github.com/ShoshinNikita/boltBrowser).
- [ ] Make it possible to send the access log to the database.
- [ ] Make it possible to send the error log to the database.
- [ ] User management interface + web REPL + stats + logs + import/export data .alg launcher.
- [ ] Make most methods in [onthefly](https://github.com/xyproto/onthefly) available to Algernon/Lua.
- [ ] Present directories with media files with a built-in page.
- [ ] Make the behavior per file extension or mime type configurable: "raw view", "pretty view" or "download"
- [ ] Add a Lua function for upgrading a handler to a WebSocket handler, also using concurrency in Lua.
- [ ] Add support for `gccgo` by changing dependencies and/or report an issue with `gccgo`.
- [ ] Add support for pushing from emacs "writefreely mode" to Algernon with this [API](https://developers.write.as/docs/api/).

Languages other than Lua
------------------------

- [ ] Add support for [zygomys](https://github.com/glycerine/zygomys) on equal footing with Lua.
- [ ] Embed the [Fennel](https://github.com/bakpakin/Fennel) package (compiles to Lua) so that Fennel can be used on equal footing with Lua for Algernon projects.

Community
---------

- [ ] Add links to various chat pages and forums in the `README.md` file.
- [ ] Create a web page for uploading, reviewing, previewing and downloading Algernon Applications (`.alg` files).

Performance and memory usage
----------------------------

- [ ] Make calling Lua scripts thread safe without using a mutex, either by modifying gopher-lua or by creating a way of calling Lua over channels.
- [ ] Profile the startup process and make it even faster.
- [ ] Add a flag for caching to the database backend instead of to memory.
- [ ] Add an option for using **brotly** compression instead of **gzip**.
- [ ] Use fasthttp (or something equally performant) when using regular HTTP: [switching to fasthttp](https://github.com/valyala/fasthttp#switching-from-nethttp-to-fasthttp).
- [ ] When requests are handled, spawn each switch/case as a Go routine. Benchmark to see if there is a difference.

Styles and themes
-----------------

- [ ] Port the layout and concept of [werc](http://werc.cat-v.org/) to Algernon. See also [gowerc](https://bitbucket.org/mischief/gowerc/src).
- [ ] Add a flag for dumping the currently used Markdown theme to a CSS file and exit.
- [ ] Add a Markdown style similar to this one: [style.css](https://hyperapp.glitch.me/style.css)
- [ ] Add a Markdown style similar to this one: [setconf](http://setconf.roboticoverlords.org/)
- [ ] Add support for [Ghost](https://ghost.org) and/or [Hugo](https://github.com/gohugoio/hugoThemes) themes.

Documentation/tutorials
-----------------------

- [ ] Add sample for using Vue.js + Algernon.
- [ ] Look into using [slate](https://github.com/lord/slate) for documentation.
- [ ] Add sample for HyperApp + database usage.
- [ ] Terminal recording of Lua tutorial using `algernon --lua`.
- [ ] Terminal recording demonstrating creating a simple register+login site.
- [ ] Update the book to be more similar to the python Flask documentation.
- [ ] Video tutorials and screencasts.
- [ ] Document what the "current directory" is for various Lua functions that deals with files.
- [ ] Document better the order of output calls when modifying the header to redirect.
- [ ] Document how to read JSON from one place and output processed data somewhere else.
- [ ] Create a docker image that comes with all the samples.
- [ ] Create a sample TODOMVC application.
- [ ] Port [niltalk](https://github.com/knadh/niltalk) to Algernon, in a separate repository.
- [ ] Create a sample chat application.
- [ ] Document possible MultiMarkdown keywords somewhere (in a separate document).
- [ ] Add a sample for bricklayer https://github.com/ademilter/bricklayer

Various
-------

- [ ] Add a C++ plugin example.
- [ ] Check behavior of ctrl-c/ctrl-d on macOS vs Linux vs Windows.
- [ ] Add a theme that looks like [huytd.github.io](https://huytd.github.io).
- [ ] Add fastcgi support, for connecting to fastcgi servers and use them for serving content?
- [ ] Write a module for caching that can cache chunks of files and stream files that does not fit in memory directly from disk.
- [ ] Add support for systemd reload, not just restart.
- [ ] Render JavaScript server-side by using [Goja](https://github.com/dop251/goja)
- [ ] Use [cfilter](https://github.com/irfansharif/cfilter) for potentially faster cache lookups.
- [ ] Support [HAML](https://github.com/travissimon/ghaml)?
- [ ] Introduce a separate package for dealing with Lua pools, Lua states and
      adding custom functions to some Lua states. All without using mutexes.
- [ ] Support for websockets (port a small multiplayer game to test).
- [ ] Add support for Handlebars: [raymond](https://github.com/aymerick/raymond)
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
- [ ] Support SASS and HAML. Maybe.
- [ ] Port Pastecat to Algernon (https://github.com/mvdan/pastecat)
- [ ] Argon2 hashing algorithm support (https://godoc.org/github.com/magical/argon2)
- [ ] Add config Function for adding a directory listing title to a certain path regex (and/or a title.txt or common.md file).
- [ ] Add a lua function for presenting an executable as a web application, like gotty does. Create a password protected example application.
- [ ] Web application for browsing the database.
- [ ] Document the case sensitivity or add case insensitivity support.
- [ ] Create a tool that pretends to upload a file of size 128 bytes
      (Content-Length), but continues to stream data. Test with Algernon.
- [ ] Lua plugin that is not via the database
- [ ] File upload while handling gzip
- [ ] Cache os.Stat also when serving directory listings
- [ ] Implement https://github.com/labstack/echo/tree/master/examples as Algernon applications
- [ ] Look into github.com/jessevdk/go-flags/
- [ ] pprint should output text to the browser when not running in the repl (or be disabled)
- [ ] Graph of visitors over time
- [ ] See if the HTTP headers from the client + country of origin + mouse
      movement patterns can become some sort of pseudo ID. Combine with a
      neural net. Can be used for storing non-critical data like preferred
      themes, font sizes etc. Time of day may also be an input.
- [ ] Add editor syntax highlight files.
- [ ] Support for pretty URLs and/or routing in serverconf.lua (/position/x/2/y/4).
- [ ] Command line utilities for editing users, permissions, databases and Lua functions in databases.
- [ ] Add a lua function for running a lua function periodically.
- [ ] Add a cache mode for caching binary files only.
- [ ] MSI installer.
- [ ] deb/ppa
- [ ] Consider using https://github.com/sbinet/igo instead of readline.
- [ ] Create a utility for creating and running new projects, ala Meteor.
- [ ] Add Lua functions for BSON and ION?
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

- [/] Make `help` work a bit like in Python.
- [/] Make `dir` work a bit like in Python.

Plugins
-------

- [ ] Unmarshal the CallPlugin reply into appropriate Lua structures instead of returning a JSON string.
- [ ] If a plugin ends with `.go`, check if go is installed and run it with "go run" (if a binary of the same name has not been provided for the current platform).
- [ ] Add a function for loading all plugins in a "plugins" directory.

Security-related
----------------

- [ ] Consider using [secure](https://github.com/unrolled/secure).
- [ ] HTTP Basic Auth using the permissions2 usernames and passwords, for selected URL prefixes. Use code from "scoreserver".
- [ ] Check that HTTP reads not only times out, but has a deadline.
- [ ] Flag for disabling directory listings entirely.
- [ ] OAuth 1
- [ ] OAuth 2
- [ ] The ability to set headers and do HTTP Basic Auth manually.
- [ ] Check if `*` or the server host should be used as parameter to the EventServer function.
- [ ] Implement a warning when using cookies over regular HTTP.

Console output
--------------
- [ ] Check the terminal capabilities and terminal width. Display a smaller logo if the width is smaller. Or no logo.

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

Maybe
-----

- [ ] Add support for both SASS and SCSS (Perhaps https://github.com/c9s/c6)
- [ ] Add configurable log hooks for the systems logrus supports.
      See: https://github.com/Sirupsen/logrus
- [ ] When searching files and directories, do it in parallel, like [wallutils](https://github.com/xyproto/wallutils).
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
- [ ] Read zip files directly instead of decompressing when given as the
      first argument (downside: some Amber functions look for files in the
      same directory).
- [ ] Utilities to lint and package .alg archives.
- [ ] Whitelist and blacklist for which file extensions to cache
- [ ] Use golang/pkg/net/rpc/#Client.Go for calling plugins asynchronously.
      Let Lua provide a callback function.
- [ ] Configuration function for whitelisting URL prefixes.
- [ ] Functions for adding URL prefixes to the whitelist
- [ ] Lua function for reading the contents of a file in the script dir,
      but in a cached way. Timestamp, filename and data are stored in Redis,
      if timestamp changes, data is re-read.
- [ ] Add a Lua function for removing all cache entries without a hit.
- [ ] Support the LuaPage format (".lp", HTML with <% %> and <%= %> for Lua code).
- [ ] Add Lua functions for HTTP PUT without using JSON? (for etcd, but might be a bad idea in the first place).
- [ ] Rewrite in C++17 and rename the project to "FnuFnu".

Alternative databases
---------------------

- [ ] RethinkDB support.
- [ ] Add support for the [Badger](https://blog.dgraph.io/post/badger-lmdb-boltdb/).

# Future

## Algernon 2030

- [ ] Use something like fasthttp from the start.
- [ ] Drop Amber and GCSS.
- [ ] Focus on templates, markdown and possibly microservices.
- [ ] Embed a different language than Lua, perhaps Anko.
- [ ] Support `gccgo` from the start.
- [ ] Aim for a very small and specific usage pattern and try to optimize for that.
- [ ] Parse options with [docopt](https://github.com/docopt/docopt.go) or [cli](https://github.com/urfave/cli).
- [ ] Use [configparser](https://github.com/alyu/configparser) for a configuration file with port, host, keys etc.
