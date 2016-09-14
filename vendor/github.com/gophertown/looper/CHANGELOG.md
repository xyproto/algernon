# Looper Changelog

Roadmap & voting at the public [Trello board](https://trello.com/b/VvblYiSE).

## v0.3.3 / 2015-08-23

* Skip vendor folder on full test run when GO15VENDOREXPERIMENT is set.
* Fix typo in NewRecursiveWatcher (thanks @corrupt).

## v0.3.2 / 2014-11-13

* Don't watch directories starting with an underscore (like the go tool).

## v0.3.0 / 2014-11-12

* Add Godep support `godeps go test` (thanks @sudhirj)
* Switch tests from launchpad to gopkg.in/check.v1 (thanks @aibou)

## v0.2.3 / 2014-06-12

* Update to new gophertown/fsnotify API (v0.11.0).
* Ignore metadata changes when detecting modifications.

## v0.2.2 / 2014-05-23

* Use gophertown/fsnotify (experimenting with the API there for now)

## v0.2.1 / 2013-07-06

* Add --debug flag to help track down [#6] Tests run twice

## v0.2.0 / 2013-05-16

* Rename to Looper
* Packages are the unit of compilation in Go. Use a package-level granularity for testing.
* Don't log Stat errors (can be caused by atomic saves in editors)

## v0.1.1 / 2013-04-21

* Fixes "named files must all be in one directory" error [#2]
* Pass through for -tags command line argument. Thanks @jtacoma.

## v0.1.0 / 2013-02-24

* Recursively watches the file system, adding subfolders when created.
* Readline interaction to run all tests or exit.
* ANSI colors to add some flare.
* Focused testing of a single file for a quick TDD loop (subject to change)
