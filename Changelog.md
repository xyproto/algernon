# Changelog

Changes from 1.4.1 to 1.4.2
===========================

* Update dependencies
* Minor improvements to the code
* Minor improvements to the documentation

Changes from 1.4 to 1.4.1
=========================

* Updates to the Markdown styling: tables, colors and &lt;code&gt; tags
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
* Refactoring.

Changes from 1.3.2 to 1.4
=========================

* Improve autocomplete in the REPL.
* Only add syntax highlighting to rendered HTML when needed.
* Some refactoring: made the code simpler.
* Move error checks before defer statements whenever possible.
* Set headers so that browsers will download the most common binary formats instead of displaying them.
* Update dependencies.

Changes from 1.3.1 to 1.3.2
===========================

* Remove the dependency on readline. No external dependencies.
* The beginnings of better completion in the REPL.
* Update the external dependencies using Glide.

Changes from 1.3 to 1.3.1
=========================

* Less strict headers when using the auto-refresh feature

Changes from 1.2.1 to 1.3
=========================

* Support for streaming large files (http range)
* Minor improvements to the samples

Changes from 1.2 to 1.2.1
=========================

* Improve the REPL and the pprint function
* Fix a race issue when setting up a handle function from Lua
* Add a "Host" header to the header table

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

* Release
