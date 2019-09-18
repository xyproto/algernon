# VT100

[![Build Status](https://travis-ci.org/xyproto/vt100.svg?branch=master)](https://travis-ci.org/xyproto/vt100) [![GoDoc](https://godoc.org/github.com/xyproto/vt100?status.svg)](https://godoc.org/github.com/xyproto/vt100) [![License](https://img.shields.io/badge/license-MIT-green.svg?style=flat)](https://raw.githubusercontent.com/xyproto/vt100/master/LICENSE) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/vt100)](https://goreportcard.com/report/github.com/xyproto/vt100)

* Supports colors and attributes.
* Developed for Linux. May work on other systems, but there are no guarantees.
* Can detect the terminal size.
* Can get key-presses, including arrow keys.
* Has a Canvas struct, for drawing only the updated characters to the terminal.
* Uses the spec directly, but memoizes the commands sent to the terminal, for speed.
* Everything is ready if someone wants to use this to build a utility similar to `dialog` or `whiptail`.

### Images

![shooter example](img/shooter.gif)

Screen recording of the [`shooter`](cmd/shooter) example, where you can control a small character with the arrow keys and shoot with `space`.

---

![menu example](img/menu.gif)

Screen recording of the [`menu`](cmd/menu) example, which uses VT100 terminal codes and demonstrates a working menu.

---

![VT100 terminal](https://upload.wikimedia.org/wikipedia/commons/thumb/9/99/DEC_VT100_terminal.jpg/300px-DEC_VT100_terminal.jpg)

A physical VT100 terminal. Photo by [Jason Scott](https://www.flickr.com/photos/54568729@N00/9636183501), [CC BY 2.0](https://creativecommons.org/licenses/by/2.0)

### The `vt100` Go Module

### Simple use

Output "hi" in blue:

```go
fmt.Println(vt100.BrightColor("hi", "Blue"))
```

Erase the current line:

```go
vt100.Do("Erase Line")
```

Move the cursor 3 steps up (it's a bit verbose, but it's generated directly from spec, memoized for speed and is easy to wrap in a custom function):

```go
vt100.Set("Cursor Up", map[string]string{"{COUNT}": "3"})
```

The full overview of possible commands are at the top of `vt100.go`.

### Another example

See `cmd/move` for a more advanced example, where a character can be moved around with the arrow keys.

### Features and limitations

* Can detect letters, arrow keys and space. F12 and similar keys are not supported (they are supported by vt220 but not vt100).
* Resizing the terminal when using the Canvas struct may cause artifacts, for a brief moment.
* Holding down a key may trigger key repetition which may speed up the main loop.

### General info

* Version: 1.3.0
* Licence: MIT
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
