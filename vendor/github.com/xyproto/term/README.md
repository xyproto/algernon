Term
====

[![Build Status](https://travis-ci.org/xyproto/term.svg?branch=master)](https://travis-ci.org/xyproto/term)

Online API documentation
------------------------

[term API documentation at godoc.org](http://godoc.org/github.com/xyproto/term)


Features and limitations
------------------------

* Provides an easy way to get started drawing colorful characters at any location in a terminal.
* Uses ncurses and the termbox-go module.
* The dialog functionality requires `/usr/bin/dialog` to be available. Use the `SetDialogPath` if it is placed elsewhere.

Simple example
--------------

~~~go
package main

import (
	. "github.com/xyproto/term"
)

func main() {
	Init()
	Clear()
	Say(10, 7, "hi")
	Flush()
	WaitForKey()
	Close()
}
~~~

Another example
---------------

~~~go
package main

import (
	"fmt"

	"github.com/xyproto/term"
)

// Loop and echo the input until "quit" is typed
func Taunt() {
	for {
		// Retrieve user input, with a prompt. Use ReadLn() for no prompt.
		line := term.Ask("> ")

		// Check if the user has had enough
		if line == "quit" {
			break
		}

		// Taunt endlessly
		fmt.Println("No, you are " + line + "!")
	}
}

func main() {
	fmt.Println(`
Welcome to Taunt 1.0!

Type "quit" when done.

Ready.

`)
	Taunt()
}
~~~

General information
-------------------

* License: MIT
* Author: Alexander F Rødseth <rodseth@gmail.com>
* Version: 0.1
