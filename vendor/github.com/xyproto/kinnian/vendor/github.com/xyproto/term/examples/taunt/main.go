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
