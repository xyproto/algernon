# TextOutput

[![Build Status](https://travis-ci.org/xyproto/textoutput.svg?branch=master)](https://travis-ci.org/xyproto/textoutput) [![GoDoc](https://godoc.org/github.com/xyproto/textoutput?status.svg)](https://godoc.org/github.com/xyproto/textoutput) [![License](https://img.shields.io/badge/license-MIT-green.svg?style=flat)](https://raw.githubusercontent.com/xyproto/textoutput/master/LICENSE) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/textoutput)](https://goreportcard.com/report/github.com/xyproto/textoutput)

Package for controlling text output, with or without colors, on Linux, using VT100 terminal codes.

## Example use

```go
package main

import (
	"fmt"
	"github.com/xyproto/textoutput"
)

func main() {
	// Enable colors, enable output
	o := textoutput.NewTextOutput(true, true)

	// Output "a" in light blue and "b" in light green
	fmt.Println(o.LightTags("<blue>", "a", "<off> <green>", "b", "<off>"))

	// Output "a" in light blue and "b c" in light green
	fmt.Println(o.Words("a b c", "blue", "green"))

	// Output "a" in light blue and "b" in light green
	fmt.Println(o.LightBlue("a") + " " + o.LightGreen("b"))

	// Output "c" in dark blue and "d" in light yellow
	fmt.Println(o.DarkTags("<blue>c</blue> <lightyellow>d<off>"))

	// Output "a" in light blue
	fmt.Println(o.LightTags("<blue>", "a", "</blue>"))

	// Output "a" in light blue
	o.OutputTags("<blue>a</blue>")

	// Output "a" in light blue
	o.OutputTags("<blue>a<off>")

	// Exit with a dark red error message
	o.ErrExit("error: too convenient")
}
```

![screenshot](img/screenshot.png)

## General info

* Version: 1.7.0
* License: MIT
* Author &lt;xyproto@archlinux.org&gt;
