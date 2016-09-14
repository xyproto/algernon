# Looper

Looper is a development tool for the [Go Programming Language][go]. It automatically runs your tests and (will eventually) hot compile your code when it detects file system changes.

## Status

[![Build Status](https://drone.io/github.com/gophertown/looper/status.png)](https://drone.io/github.com/gophertown/looper/latest) [![Coverage](http://gocover.io/_badge/github.com/gophertown/looper)](http://gocover.io/github.com/gophertown/looper) [![GoDoc](https://godoc.org/github.com/gophertown/looper?status.png)](http://godoc.org/github.com/gophertown/looper) [![Stories in Ready](https://badge.waffle.io/gophertown/looper.png?label=ready&title=Ready)](https://waffle.io/gophertown/looper)

This is an *early alpha*. There is still quite a lot to do (Hot Compiles, Growl notifications, and interactions for profiling, benchmarking, etc.).

See the public [Trello board](https://trello.com/b/VvblYiSE) for the Roadmap. Looking into [Waffle](https://waffle.io/gophertown/looper) as an alternative.

## Get Going

If you are on OS X, you need to first install GNU Readline via [Homebrew](http://mxcl.github.com/homebrew/):

``` console
$ brew install readline
```

Note: If you upgraded to Xcode 5 you will need Go v1.2.0-rc.1 or better.

To install Looper, or to update your installation, run:

``` console
$ go get -u github.com/gophertown/looper
```

Then run `looper` in your project folder:

``` console
$ looper
Looper 0.2.3 is watching your files
Type help for help.

Watching path ./
```

Note: There is [a known issue](https://github.com/gophertown/looper/issues/6) where tests may run multiple times on OS X. Until this is resolved, please add your development folder to the Spotlight Privacy in System Preferences.

## Gat (Go Autotest)

Packages are the unit of compilation in Go. By convention, each package has a separate folder, though a single folder may also have a `*_test` package for black box testing.

When Looper detects a change to a `*.go file`, it will build & run the tests for that directory. You can also run all tests against all packages at once.

To setup a Suite definition ([Gocheck][], [PrettyTest][pat]), additional Checkers, or other test helpers, use any test file you like in the same folder (eg. `suite_test.go`).

Gat is inspired by Andrea Fazzi's [PrettyAutoTest][pat].

## Blunderbuss (Hot Compile)

...to be determined...

Blunderbuss is inspired by [shotgun][], the reloading rack development server for Ruby.

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
* [shotgun-go](https://github.com/danielheath/shotgun-go) by Daniel Heath
* [Revel](http://robfig.github.io/revel/) by Rob Figueiredo does Hot Code Reloading

### Comprehensive

* [golab](https://github.com/mb0/lab) Linux IDE by Martin Schnabel
* [GoTray](http://gotray.extremedev.org/) for OS X

## Thanks

Special thanks to Chris Howey for the [fsnotify][] package.

[go]: http://golang.org/
[fsnotify]: https://github.com/howeyc/fsnotify
[pat]: https://github.com/remogatto/prettytest
[shotgun]: https://rubygems.org/gems/shotgun
[Gocheck]: http://labix.org/gocheck

