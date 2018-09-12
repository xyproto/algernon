# Changelog

Changes from 1.10.1 to 1.11.0
=============================

* Using the `go mod` system that came with Go 1.11.
* Experimental support for simple logging to a NCSA and/or a Combined access log, with two new commandline flags.
* Minor improvements to the help text and status messages.
* No external resources are required by Algernon, not even external fonts, ref #17.
* Refactoring: moved the event server to the recwatch package.
* Remove an unneeded space when setting `Content-Type`.
* Better keyword handling in Markdown documents.
* Set a mimetype for configuration files starting with a `.`.
* Add a flag for clearing the default path prefixes used by the permissions subsystem.
* Update test script.
* Minor changes to documentation and samples.

Changes from 1.10 to 1.10.1
===========================

* Workaround for a problem with the MINGW64 terminal + readline, on Windows.
* Let Lua handlers also configure the server.
* Release the Windows executable together with the samples.

Changes from 1.9 to 1.10
========================

* Syntax highlighting by using [chroma](https://github.com/alecthomas/chroma) instead of [highlight.js](https://highlightjs.org/).
* No external dependencies, ref issue #17.
* Add a mode for only using the Lua REPL with `-l` or `--lua`.
* New logo for the webpage, and new ANSI banner on the command line.
* Minor fix for closing `</head>` tags.
* Update vendored dependencies.

Changes from 1.8 to 1.9
=======================

* Improve error messages.
* Better support for giving a single Lua fila with handlers as an argument.
* Update documentation and samples.
* Add list:json() to make it easier to return JSON from a List. See `samples/react_db`.
* Better handling of opening documents in the browser if no certs are given.
* Update the default handling of files (view/download) based on mime type or extension.
* Add support for "go tool trace".
* Update vendored dependencies.

Changes from 1.7 to 1.8
=======================

* Fix an issue with `curl` + `algernon` that was not present with `wget` + `algernon`, related to HTTP headers and compression.
* Some refactoring and linting of the code.
* Less strict HTTP headers by default.
* Update vendored dependencies.

Changes from 1.6 to 1.7
=======================

* Experimental support for the QUIC protocol (HTTP over UDP, faster than HTTP/2).
* Improvements toward compiling Algernon with GCC (`gcc-go`).
* Update HyperApp support and samples to work with the latest version (0.15.1).
* Update dockerfiles and scripts.
* Add "material" and "neon" themes.
* Updated the documentation.
* Add support for `.algernon` files for configuring directory listings (set a theme and title).
* Support for having a port number as the only argument.
* Add a `--nodb` flag, for not using any database backend (same as `--boltdb=/dev/null`).
* Some refactoring.
* Update vendored dependencies.

Changes from 1.5.1 to 1.6
=========================

* Fix for excessive memory usage when serving and caching large files. Needs more testing.
* Should now be possible to compile with gccgo.
* Revert the refactoring to a separate "kinnian" package, for easier development and dependency handling.
* Update vendored dependencies.

Changes from 1.5 to 1.5.1
=========================

* Add the `.hyper.js` and `.hyper.jsx` extensions for HyperApp applications
* Style HyperApp applications if no style/theme/`style.gcss` is provided
* Also support HyperApp applications when using the `--theme=...` flag
* Add the `hprint` Lua function, for combining HyperApp and Lua

Changes from 1.4.5 to 1.5
=========================

* Switch JSX rendering engine to one that uses [goja](https://github.com/dop251/goja)
* Add support for HyperApp JSX apps with the `.happ` or `.hyper` extension

Changes from 1.4.4 to 1.4.5
===========================

* Performance improvements when rendering Markdown and directory listings
* Refactoring out code to the `kinnian` package
* Update samples.
* Update vendored dependencies.

Changes from 1.4.3 to 1.4.4
===========================

* Refactor code into packages.
* Update tests and documentation.

Changes from 1.4.2 to 1.4.3
===========================

* Update dependencies and the dependency configuration.

Changes from 1.4.1 to 1.4.2
===========================

* Minor improvements to the code.
* Minor improvements to the documentation.
* Update dependencies.

Changes from 1.4 to 1.4.1
=========================

* Update the Markdown styling: tables, colors and &lt;code&gt; tags
* Split out file caching to a separate package: [datablock](https://github.com/xyproto/datablock)
* Add an [example](https://github.com/xyproto/algernon/tree/master/samples/structure) for structuring a web site.
* Add a Lua `preload()` function, for caching files before they are needed.
* Let the Lua `render()` and ` serve()` functions take an optional filename.
* Fallback for the log filename.
* Add `-V` flag for "verbose".
* Add `--ctrld` flag for having to press `ctrl-d` twice to exit the REPL.
* Use BoltDB by default instead of Redis.
* Add script for testing functionality (HTTP server + curl) that is ran by the CI system.
* Fix issue when running some `.alg` files.
* Refactor.

Changes from 1.3.2 to 1.4
=========================

* Improve autocomplete in the REPL.
* Only add syntax highlighting to rendered HTML when needed.
* Some refactoring: made the code simpler.
* Move error checks before defer statements whenever possible.
* Set headers so that browsers will download the most common binary formats instead of displaying them.
* Update vendored dependencies.

Changes from 1.3.1 to 1.3.2
===========================

* Remove the dependency on readline. No external C dependencies left.
* The beginnings of better completion in the REPL.
* Update dependencies using Glide.

Changes from 1.3 to 1.3.1
=========================

* Less strict headers when using the auto-refresh feature.

Changes from 1.2.1 to 1.3
=========================

* Support for streaming large files (HTTP range).
* Minor improvements to the samples.

Changes from 1.2 to 1.2.1
=========================

* Improve the REPL and the pprint function.
* Fix a race issue when setting up a handle function from Lua.
* Add a "Host" header to the header table.

Changes from 1.1 to 1.2
=======================

* Add support for SCSS (Sass).
* Fix the Pongo2 + Lua data race issue with a Mutex.
* Vendor all dependencies and add a Glide YAML file.
* Render `* [ ]`, `* [x]` and `* [X]` in Markdown as checkboxes.
* Add support for different images per Markdown theme, using the `replace_with_theme` keyword.
* Add support for custom CSS from Markdown, using the `css` keyword.
* Add another built-in Markdown theme: redbox.
* Remove unused variables.

Changes from 1.0 to 1.1
=======================

General
-------

* Tested with Go 1.7
* Added PostgreSQL >= 9.1 support (with the HSTORE feature).
* Added two built-in themes for error pages, directory listings and Markdown.
* Added a `--theme` flag for selecting a theme.
* Added a `--nocache` flag for disabling caching.
* Added default HTTP headers, for security.
* Algernon servers now get A+ at https://securityheaders.io/.
* Added a `--noheader` flag for disabling security-related HTTP headers.
* Switched back to the official pongo2 repo after a pull request was merged.

Lua
---

* Added a `pprint` function for slightly prettier printing.
* Added a `ppstr` function for slightly prettier printing, but to a string.
* Let `redirect` take an optional HTTP status code.
* Added a `permanent_redirect` function which only takes an URL.
* Let `dofile` search the directory of the Lua file that is running.
* Fixed an issue with returning HTTP status codes from Lua in Debug mode.
* Renamed `toJSON` to just `JSON`. Both are still present and still work.

Markdown
--------

* Can now select a built-in theme with `theme:`.
* Can now select the highlight.js theme with `code_style:`.

REPL
----

* More graceful shutdown upon SIGHUP on Linux.

Deployment
----------

* Minor improvements to the `alg2docker` script.

Samples and documentation
-------------------------

* Fixed several minor typos.
* Added an URL location check in the "bob" sample.

1.0
===

* Release.
