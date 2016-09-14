# Looper

Looper is a development tool for the [Go Programming Language][go]. It automatically runs your tests and (will eventually) hot compile your code when it detects file system changes.

![Looper screenshot](https://raw.githubusercontent.com/nathany/looper/master/looper.png)

## Status

[![Stories in Ready](https://badge.waffle.io/nathany/looper.svg?label=ready&title=Ready)](http://waffle.io/nathany/looper)  [![Build Status](https://drone.io/github.com/nathany/looper/status.png)](https://drone.io/github.com/nathany/looper/latest) [![Coverage](http://gocover.io/_badge/github.com/nathany/looper)](http://gocover.io/github.com/nathany/looper) [![GoDoc](https://godoc.org/github.com/nathany/looper?status.svg)](http://godoc.org/github.com/nathany/looper) 

[![Throughput Graph](https://graphs.waffle.io/nathany/looper/throughput.svg)](https://waffle.io/nathany/looper/metrics)

This is an *early alpha*. There is still quite a lot to do (Hot Compiles, Growl notifications, and interactions for profiling, benchmarking, etc.).

## Get Going

If you are on OS X, you need to first install GNU Readline via [Homebrew](http://mxcl.github.com/homebrew/):

``` console
$ brew install readline
```

If you are on Linux, you'll need the readline development headers:

Debian/Ubuntu:

```console
sudo apt-get install libreadline-dev
```

Red Hat-based systems:

```console
sudo yum install readline-devel
```

To install Looper, or to update your installation, run:

``` console
$ go get -u github.com/nathany/looper
```

Then run `looper` in your project folder:

``` console
$ looper
Looper 0.3.3 is watching your files
Type help for help.

Watching path ./
```

Note: There is [a known issue](https://github.com/nathany/looper/issues/6) where tests may run multiple times on OS X. Until this is resolved, please add your development folder to Spotlight Privacy in System Preferences.

## Gat (Go Autotest)

Packages are the unit of compilation in Go. By convention, each package has a separate folder, though a single folder may also have a `*_test` package for black box testing.

When Looper detects a change to a `*.go file`, it will build & run the tests for that directory. You can also run all tests against all packages at once.

To setup a Suite definition ([Gocheck][], [PrettyTest][pat]), additional Checkers, or other test helpers, use any test file you like in the same folder (eg. `suite_test.go`).

Gat is inspired by Andrea Fazzi's [PrettyAutoTest][pat].

## Blunderbuss (Hot Compile)

...to be determined...

Blunderbuss is inspired by [shotgun][], both in name and purpose.

## Interactions

* `a`, `all`, `↩`: Run all tests.
* `h`, `help`: Show help.
* `e`, `exit`: Quit Looper

## Related Projects

### General purpose

* [Reflex](https://github.com/cespare/reflex) by Caleb Spare
* [rerun](https://github.com/skelterjohn/rerun) by John Asmuth to autobuild and kill/relaunch
* [Watch](https://github.com/eaburns/Watch) by Ethan Burns includes acme integration
* [watcher](https://github.com/tmc/watcher) by Travis Cline

### Testing

* [PrettyAutoTest][pat] by Andrea Fazzi
* [Glitch](https://github.com/levicook/glitch) by Levi Cook

### Web development

* [App Engine devserver](https://developers.google.com/appengine/docs/go/tools/devserver)
* [devweb](http://code.google.com/p/rsc/source/browse/devweb/) by Russ Cox
* [gin](https://github.com/codegangsta/gin) by Jeremy Saenz
* [rego](https://github.com/sqs/rego) by Quinn Slack
* [shotgun-go](https://github.com/danielheath/shotgun-go) by Daniel Heath
* [Revel](http://revel.github.io/) by Rob Figueiredo does Hot Code Reloading

### Comprehensive

* [golab](https://github.com/mb0/lab) Linux IDE by Martin Schnabel
* [GoTray](http://gotray.extremedev.org/) for OS X

## Thanks

Special thanks to Chris Howey for the [fsnotify][] package.

[go]: http://golang.org/
[fsnotify]: http://fsnotify.org/
[pat]: https://github.com/remogatto/prettytest
[shotgun]: https://rubygems.org/gems/shotgun
[Gocheck]: http://labix.org/gocheck

