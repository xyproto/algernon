# VT [![GoDoc](https://godoc.org/github.com/xyproto/vt?status.svg)](https://godoc.org/github.com/xyproto/vt) [![License](https://img.shields.io/badge/license-BSD-green.svg?style=flat)](https://raw.githubusercontent.com/xyproto/vt/main/LICENSE) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/vt)](https://goreportcard.com/report/github.com/xyproto/vt)

This is a package for dealing with terminal emulators.

* Supports colors and attributes, including 256-color and true-color (24-bit RGB).
* Supports color operations such as `Lighten`, `Darken`, `Blend` and `ContrastRatio`.
* Supports the `NO_COLOR` environment variable.
* Supports HTML-like tagged color output, such as `<red>hello</red>`.
* Supports Linux, macOS, other Unix-like systems, and Windows.
* Can detect the terminal size.
* Can get key-presses, including arrow keys (252, 253, 254, 255), pgup/pgdn (251, 250), F1–F12, Home, End, Delete, Shift-Tab, and modifier combinations (Alt, Ctrl, Shift) on xterm-class terminals.
* Has a Canvas struct, for drawing only the updated lines to the terminal, with synchronized updates and wide character support.
* Can render a Canvas to an `image.Image`.
* Can detect terminal capabilities, such as `Multiplexed()`, `XtermLike()`, `Has256Colors()` and `GetBackgroundColor()`.
* Has `NewTTYFromReader`, for scripted or test input without a real terminal.
* Could be used for making an alternative to the `dialog` or `whiptail` utilities.

### Images

![shooter example](img/shooter.gif)

Screen recording of the [`shooter`](cmd/shooter) example, where you can control a small character with the arrow keys and shoot with `space`.

---

![menu example](img/menu.gif)

Screen recording of the [`menu`](cmd/menu) example, which uses VT100 terminal codes and demonstrates a working menu.

---

![VT100 terminal](https://upload.wikimedia.org/wikipedia/commons/thumb/9/99/DEC_VT100_terminal.jpg/300px-DEC_VT100_terminal.jpg)

A physical VT100 terminal. Photo by [Jason Scott](https://www.flickr.com/photos/54568729@N00/9636183501), [CC BY 2.0](https://creativecommons.org/licenses/by/2.0)

### Requirements

* Go 1.25 or later.

### Features and limitations

* Can detect letters, arrow keys, F1–F12, Home, End, Delete, Shift-Tab, and modifier combinations such as Ctrl, Alt and Shift combined with arrow keys, on xterm-class terminals.
* Resizing the terminal when using the Canvas struct may cause artifacts, for a brief moment.
* Holding down a key may trigger key repetition which may speed up the main loop.
* Modifier key combinations are only emitted by xterm-class terminals; use `HasAltArrows()`, `HasCtrlArrows()` or `HasShiftArrows()` to check.

### Another example

See `cmd/move` for a more advanced example, where a character can be moved around with the arrow keys.

### General info

* License: BSD-3
* Version: 1.9.10
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
