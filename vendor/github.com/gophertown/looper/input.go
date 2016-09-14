package main

import (
	"io"
	"log"
	"strings"

	"github.com/bobappleyard/readline"
)

type Command int

const (
	Unknown Command = iota
	Help
	Exit
	RunAll
)

func CommandParser() <-chan Command {
	commands := make(chan Command, 1)

	go func() {
		for {
			in, err := readline.String("")
			if err == io.EOF { // Ctrl+D
				commands <- Exit
				break
			} else if err != nil {
				log.Fatal(err)
			}

			commands <- NormalizeCommand(in)
			readline.AddHistory(in)
		}
	}()

	return commands
}

func NormalizeCommand(in string) (c Command) {
	command := strings.ToLower(strings.TrimSpace(in))
	switch command {
	case "exit", "e", "x", "quit", "q":
		c = Exit
	case "all", "a", "":
		c = RunAll
	case "help", "h", "?":
		c = Help
	default:
		UnknownCommand(command)
		c = Unknown
	}
	return c
}
