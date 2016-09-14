package main

import (
	"github.com/bobappleyard/readline"
	"io"
	"log"
	"strings"
)

type Command int

const (
	UNKNOWN Command = iota
	HELP
	EXIT
	RUN_ALL
)

func CommandParser() <-chan Command {
	commands := make(chan Command, 1)

	go func() {
		for {
			in, err := readline.String("")
			if err == io.EOF { // Ctrl+D
				commands <- EXIT
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
		c = EXIT
	case "all", "a", "":
		c = RUN_ALL
	case "help", "h", "?":
		c = HELP
	default:
		UnknownCommand(command)
		c = UNKNOWN
	}
	return c
}
