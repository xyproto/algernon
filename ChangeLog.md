# Changelog

Changes from 1.17.0 to 1.17.1
=============================

* Patch the `pingcap/tidb` dependency so that Algernon still compiles for ARM 6 and ARM 7.
* Add the `oc:distance()` Lua method for finding the distance between two LLM prompts.
* Let the `embeddedDistance` Lua function take a 3rd argument: `euclidean`, `manhattan`, `chebyshev`, `hamming` or `cosine`.
* Improve the image description example.
* Improve how images are served when serving a single Markdown document.
* Fix the double opening of the browser when both `-m` and `-o` are specified.
* Also cache `.webp` images, if caching of images is enabled.
* Update dependencies.
* Update documentation.

Changes from 1.16.0 to 1.17.0
=============================

* Add three Ollama-related functions: `base64EncodeFile`, `describeImage` and `embeddedDistance`.
* Add an example for uploading and describing images with Ollama and the `llava-llama3` model.
* Add a `base64` method to `UploadedFile` objects.
* Update the Teal example.
* Improve the "pretty error page" that appears if there is an error in a Lua script.
* Let some variables be constants instead.
* Avoid using `runtime.GOOS` and detect most features at compile time instead.
* Use two `atomic.Bool` variables instead of a mutex for keeping track of how data is being served.
* Import `logrus` as "logrus" instead of as "log".
* Use `strconf.FormatInt` instead of `fmt.Sprintf` whenever possible.
* Use `github.com/pkg/browser` for opening URLs in a browser.
* Fix an issue with inserting JS into HTML.
* Serve HTTP with `fasthttp´ (ref #4)
* Improve how favicons are handled.
* Make it possible to ignore files in a directory listing (ref #149).
* Add a `sanhtml` function for sanitizing HTML.
* Add experimental support for mathematical formulas in Markdown using MathJax (ref #150).
* Add an example that renders mathematical formulas.
* Remove `GOEXPERIMENT=loopvar` since it is no longer needed.
* Follow the advice of `golint`.
* Update documentation.
* Update dependencies.

Changes from 1.15.5 to 1.16.0
=============================

* Make it possible to clear the AI cache with `ClearCache()`.
* Add support for using AI/LLMs (Ollama) from Lua.
* Add support for `.prompt` files that contains a content-type, a model name, a blank line and a prompt.
* Make small changes to the built-in themes.
* Update CI configuration.
* Require Go 1.21 or later, mostly because of the QUIC dependency.
* Update the year in the license file.
* Minor changes to the `alg2docker` script.
* Update the react jsonfile example.
* Update the TODO list example to the latest version of React.
* Update the React + JSX + database example.
* Add the loopvar feature to the dockerfiles.
* Use LABEL maintainer in the Dockerfiles.
* Combine several build and run scripts related to docker.
* Add a simple tutorial.
* Remove some unused code.
* Update documentation.
* Update dependencies.

Changes from 1.15.4 to 1.15.5
=============================

* Make the Makefile clearer.
* Update the form/registration example.
* Update dependencies.

Changes from 1.15.3 to 1.15.4
=============================

* Improve field alignment using `dkorunic/betteralign`.
* Update dependencies.

Changes from 1.15.2 to 1.15.3
=============================

* Set `GOEXPERIMENT=loopvar` and adjust the build flags.
* Enable profile-guided optimization.
* Update the benchmark script.
* Check the arguments for the `servedir` Lua function.
* Update the GH action for Homebrew.
* Make the prefixmatch tests pass.
* Remove the `github.com/bmizerany/assert` dependency.
* Update the CI configuration.
* Move two functions to `github.com/xyproto/files`.
* Minor changes to the welcome script.
* Minor changes to a test.
* Minor changes to the Teal example.
* Update dependencies.

Changes from 1.15.1 to 1.15.2
=============================

* Serve `.json` files a tiny bit faster.
* Serve Algernon web applications (`.alg` files) from memory, ref #132 (thanks Dialga / @Dialga).
* Remove a duplicate word from the `README.md` file (thanks Philipp Gillé / @philippgille).
* Update dependencies.

Changes from 1.15.0 to 1.15.1
=============================

* Switch from `blackfriday` to `gomarkdown/markdown`.
* Add a simple example that uses the `markdown` function.
* Update the CI configuration.
* Update dependencies.
* Update documentation.

Changes from 1.14.0 to 1.15.0
=============================

* Compile the release binaries with Go 1.20.4.
* Add a `close()` function, ref #124 (thanks Malcolm Ke Win / @diyism).
* Add a shell linter to the CI configuration, ref #120 (thanks Jan Macku / @jamacku).
* Support reverse proxies, ref #131 (thanks Mohamed Abdel Maksoud / @mohamed--abdel-maksoud).
* Look for `handler.lua` in parent directories, ref #95, #112 and #130 (thanks Giulio Lunati / @giuliolunati).
* Add initial support for JWT tokens.
* Use `os` and `io`instead of the deprecated `ioutil` package.
* Use `any` instead of `interface{}`.
* Use the new `unix` build constraint.
* Use `strings.ReplaceAll` and `bytes.ReplaceAll`.
* Use `simpleredis/v2`.
* Use `math.Round`.
* Add an `ulimit` check to the `welcome.sh` script that also works on macOS Ventura.
* Format/lint the code with `gofumpt`, `golint` and `staticcheck`.
* Use the `betteralign` tool, to improve struct field alignment.
* Make the code debug/tracing/profiling features optional at compile time, using build tags.
* Fix a typo in one of the examples.
* Update the CI configuration.
* Update dependencies.
* Update documentation.

Changes from 1.13.0 to 1.14.0
=============================

* Compile the release binaries with Go 1.19.
* Improve the documentation (thanks Matt Mc / @tooolbox ).
* Add support for Teal together with a Teal sample (thanks Matt Mc / @tooolbox).
* Fix an issue with how Lua tables were pretty printed in the REPL.
* Fix an issue with conversion from Lua tables to JSON, ref #107, #108 (thanks @linkerlin).
* Fix an issue with the generated directory listing pages, where `%2F` would appear in the URL instead of `/`, ref #117.
* Follow the advice of these utilities: `go fmt`, `golint`, `staticcheck` and to some extent `fieldalignment`.
* Update dependencies.

Changes from 1.12.14 to 1.13.0
==============================

* Add a flag for serving domains with CertMagic and Let's Encrypt
* Add a flag for redirecting from HTTP to HTTPS
* Use `req.Context` since `CloseNotifier` has been deprecated
* Switch to Go 1.18
* Switch from the MIT license to BSD-3
* Fix double drawn frames around syntax highlighted code in Markdown documents
* URL encode links when listing directories
* Use the same directory as the pongo2 template when importing macros, ref #84
* Let plugins continue to run if an optional argument is passed in, ref #64 (otherwise close them)
* Switch from jvatic/goja-babel to wvanw/esbuild, ref #77 (#91)
* Improve JSX-related error messages
* Use yuin/gopher-lua and yuin/gluamapper
* Use a context when running Lua functions and use the background context when creating the Lua pool
* Update the alg2docker and benchmark scripts
* Update the `--help` output
* Fix a typo in the "single.alg" example Algernon application
* Update example service and Dockerfiles
* Add a base URL flag for the directory listing (#90 ?)
* Follow the advice of the "fieldalignment" and "staticcheck" utilies
* Fix the `serve2` function so that the registration form example works
* Update tests, dependencies, examples and documentation

Changes from 1.12.13 to 1.12.14
===============================

* Downgrade fsnotify to v1.4.9 so that building with GOOS=freebsd works again

Changes from 1.12.12 to 1.12.13
===============================

* Fix a typo in the documentation (thanks Felix Yan)
* Add support for simple MSSQL queries, ref #57
* Improve MSSQL support (thanks Matt Mc)
* Improvements to table mappings in Lua, including changes to gluamapper (thanks Matt Mc)
* Support headers in buffered responses, ref #75 (thanks Matt Mc)
* Improvements to the file upload functionality (thanks Matt Mc)
* Various minor fixes and improvements (thanks Matt Mc)
* Add three new repl commands: `pwd`, `serverdir` and `serverfile`
* Add nicer help output for built-in commands to the repl
* Add a `ServerDir` function for the server configuration Lua script
* Fix wasm mimetype issue, ref #82
* Fix the Babl plugin configuration after updating the Babl dependency
* Various improvements to the samples and to the "Welcome" page
* Follow the advice of `go vet`, `golint` and `staticcheck`
* Support Go 1.16 and Go 1.17 only, for now
* Update CI configuration
* Update dependencies
* Update documentation

Changes from 1.12.11 to 1.12.12
===============================

* Only include QUIC support on supported platforms. This should let Algernon build for the Apple M1 CPU.

Changes from 1.12.10 to 1.12.11
===============================

* Remove OpenBSD support while waiting for [pkg/term](https://github.com/pkg/term) to support it.

Changes from 1.12.9 to 1.12.10
==============================

* Use a specific commit of [pkg/term](https://github.com/pkg/term) so that it also compiles for FreeBSD.

Changes from 1.12.8 to 1.12.9
=============================

* Improve the man page.
* Minor improvements for the help and completion functionality in the REPL.
* Let several `algernon --lua` instances not use the same temporary database.
* Let `.mk`, `.ts` and `.tsx` be served as `text/plain;charset=utf-8`.
* Initial support for rendering `.frm` and `.form` files written in SimpleForm.
* Fix for making it possible to use `.` together with `--autorefresh`.
* Minor fixes to the docker example files.
* Correct a typo in a comment (thanks Felix Yan).
* Update the Travis CI configuration (thanks Rui Chen).
* Follow the advice of the very useful `staticcheck` utility.
* Update documentation.
* Update dependencies.

Changes from 1.12.7 to 1.12.8
=============================

* Update documentation.
* Improve CI config and Homebrew release process (thanks Rui Chen!).
* Update supplied systemd configuration.
* Remove mentions of nacl.
* Remove the `mitchellh/colorstring` dependency.
* Update dependencies.
* Use `algernon_history.txt` as the REPL history filename on Windows.
* Don't output raw color codes on Windows, use ANSI colors or disable the color.
* Remove symlinks from the "welcome" sample.
* Update the release script to also build with GOARM=7 for Raspberry Pi 2, 3 and 4.

Changes from 1.12.6 to 1.12.7
=============================

* Issues with bolt db, simplebolt and `gccgo` are resolved. Algernon now also supports `gccgo`.
* Now requires Go 1.11 or later.
* Respect `TMPDIR`, for improved Termux support.
* Fix issue #42, when `--dir` is used together with a trailing slash.
* Don't force the use of the bolt database when in development mode.
* Update dependencies.

Changes from 1.12.5 to 1.12.6
=============================

* Now using a fork of the quic package, since there were build issues with it (could not build with `gccgo` and issue #41).
* Updated dependencies.
* There are still issues with compiling simplebolt with gccgo, which is why Algernon can not be compiled with gccgo in a way where simplebolt works, yet. This is related to different behavior between go and gccgo and will be worked around in simplebolt. See: golang/go#36430
* The autorefresh feature (-a or --autorefresh) may now follow symlinks to diretories, to make the ./welcome.sh script and example more user-friendly when live editing for instance samples/greetings/index.md.
* The file-search backend of the autofresh feature is now also concurrent.
* Tested with the latest version of Go (1.13.5) on 64-bit Arch Linux.

Changes from 1.12.4 to 1.12.5
=============================

* Tested with Go 1.13.
* Adds support for PostgreSQL queries with the PQ function, from Lua.
* Updated dependencies, especially with QUIC and HTTP/2 in mind.
* Updated the JSX sample to use the latest version of React.
* The static executable for Linux is now built with `-trimpath`.
* New HTTP client functionality from Lua, using GET or HTTPClient.
* `CookieSecret` and `SetCookieSecret` can now be used to get and set the secure cookie secret from Lua, or it can be set with the `--cookiesecret` flag.

Changes from 1.12.3 to 1.12.4
=============================

* Fix #26, an issue with using Lua tables together with Pongo2 and the serve2 function.
* Update dependencies.
* Improved help function on the Lua prompt.
* Support the `IGNOREEOF` environment variable.
* Update documentation.

Changes from 1.12.2 to 1.12.3
=============================

* Fix #25, where an attack with vegeta could make Algernon crash.
* Update dependencies (boltdb has a new home, TLS 1.3 has further improvements).

Changes from 1.12.0 to 1.12.2
=============================

* Update dependencies.
* Better output to stdout when loading configuration files (lists the names of all loaded configuration files).
* A timestamp is added to the command line output when starting Algernon.
* Slightly modified console text colors.
* Minor changes to recognized filename extensions.
* Update documentation to mention welcome.sh (fixes issue #23).
* Minor updates to javascript libraries used by two of the samples.
* Improved support for streaming large files (fixes issue #13).
* Added two new flags:
  * `--timeout=N` for setting a timeout in seconds, when serving large files (but there is range support, so if a download times out, the client can continue where it left).
  * `--largesize=N` for setting a threshold for when a file is too large to be read into memory (the default is 42 MiB).

Changes from 1.11.0 to 1.12.0
=============================

* Favicon support when serving Markdown files.
* More minimal `--lua` mode.
* Using the new `strings.Builder` in Go.
* Better Markdown keyword handling.
* Update vendored dependencies.
* Include transmitted bytes in the access log.
* Detect `style.css` in addition to `style.gcss`.
* Better Markdown checkbox support.
* Some refactoring and linting.

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
* Add an [example](https://github.com/xyproto/algernon/tree/main/samples/structure) for structuring a web site.
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
