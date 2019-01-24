# recwatch

[![Build Status](https://travis-ci.org/xyproto/recwatch.svg?branch=master)](https://travis-ci.org/xyproto/recwatch)

Watch filesystem changes recursively.

`recwatch` started out as a fork of [looper](https://github.com/nathany/looper) (by Nathan Youngman), but most of the code is now original.

## Features

* Can be used for watching filesystem changes recursively.
* Can be used for setting up a SSE ([server-sent events](https://en.wikipedia.org/wiki/Server-sent_events)) server, for serving filesystem changes as events.

## Example use

```go
package main

import (
	"github.com/xyproto/recwatch"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

func plural(passed time.Duration) string {
	if passed.Seconds() != 1 {
		return "s"
	}
	return ""
}

func main() {
	eventAddr := "0.0.0.0:5555"
	eventPath := "/"
	pathToWatch := "tempdir"
	refreshDuration, err := time.ParseDuration("350ms")
	if err != nil {
		log.Fatalln(err)
	}
	now := time.Now().UTC()
	_ = os.Mkdir(pathToWatch, 0755)
	recwatch.EventServer(pathToWatch, "*", eventAddr, eventPath, refreshDuration)
	URL := "http://" + strings.Replace(eventAddr, "0.0.0.0", "localhost", 1) + eventPath
	log.Printf("Serving filesystem events for %s as SSE (server-sent events) on %s\n", pathToWatch, URL)
	tempFileName := filepath.Join(pathToWatch, "hello.txt")
	_ = os.Remove(tempFileName)
	// Set up a handler for SIGINT (ctrl-c)
	quit := false
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			// ctrl-c was pressed
			quit = true
			log.Println("ctrl-c was pressed, cleaning up")
			_ = os.Remove(tempFileName)
			_ = os.Remove(pathToWatch) // will only remove the directory if it's empty
			log.Println("done")
		}
	}()
	for {
		passed := time.Since(now)
		log.Printf("%.0f second%s passed. Visit %s to see events appear.\n", passed.Seconds(), plural(passed), URL)
		time.Sleep(1 * time.Second)
		if quit {
			break
		}
		log.Println("Creating and writing to " + tempFileName)
		data := []byte("Hi\n")
		err := ioutil.WriteFile(tempFileName, data, 0644)
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(1 * time.Second)
		if quit {
			break
		}
		log.Println("Removing " + tempFileName)
		err = os.Remove(tempFileName)
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(1 * time.Second)
		if quit {
			break
		}
		if passed.Seconds() > 200 {
			log.Println("Time waits for no man.")
		}
	}
}
```

## General info

* Version: 0.3.0
* License: Simplified BSD
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
