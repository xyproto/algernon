Term
====

[![Build Status](https://travis-ci.org/xyproto/term.svg?branch=master)](https://travis-ci.org/xyproto/term)

Online API documentation
------------------------

[term API documentation at godoc.org](http://godoc.org/github.com/xyproto/term)


Features and limitations
------------------------

* Provides an easy way to get started drawing colorful characters at any position (X,Y) in a terminal.
* Uses ncurses and the [gdamore/tcell](https://github.com/gdamore/tcell) module.

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
func Repeat() {
	for {
		// Retrieve user input, with a prompt. Use ReadLn() for no prompt.
		line := term.Ask("> ")

		// Check if the user wants to quit
		if line == "quit" {
			break
		}

		// Repeat what was just said
		fmt.Println("You said: " + line)
	}
}

func main() {
	fmt.Print(`
Welcome to Repeat 1.0!

Type "quit" when done.

Ready.

`)
	Repeat()
}
~~~

General information
-------------------

* License: MIT
* Author: Alexander F. Rødseth &lt;rodseth@gmail.com&gt;
* Version: 0.3.0
