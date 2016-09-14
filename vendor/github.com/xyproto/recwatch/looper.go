// Autotesting tool with readline support.
package main

import (
	"flag"
	"github.com/gophertown/looper/gat"
	"log"
)

type Runner interface {
	RunOnChange(file string)
	RunAll()
}

func EventLoop(runner Runner, debug bool) {
	commands := CommandParser()
	watcher, err := NewRecurisveWatcher("./")
	if err != nil {
		log.Fatal(err)
	}
	watcher.Run(debug)
	defer watcher.Close()

out:
	for {
		select {
		case file := <-watcher.Files:
			runner.RunOnChange(file)
		case folder := <-watcher.Folders:
			PrintWatching(folder)
		case command := <-commands:
			switch command {
			case EXIT:
				break out
			case RUN_ALL:
				runner.RunAll()
			case HELP:
				Help()
			}
		}
	}
}

func main() {
	var tags string
	var debug bool
	flag.StringVar(&tags, "tags", "", "a list of build tags for testing.")
	flag.BoolVar(&debug, "debug", false, "adds additional logging")
	flag.Parse()

	runner := gat.Run{Tags: tags}

	Header()
	if debug {
		DebugEnabled()
	}
	EventLoop(runner, debug)
}
