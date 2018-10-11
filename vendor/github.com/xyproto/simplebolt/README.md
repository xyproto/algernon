# Simple Bolt [![Build Status](https://travis-ci.org/xyproto/simplebolt.svg?branch=master)](https://travis-ci.org/xyproto/simplebolt) [![GoDoc](https://godoc.org/github.com/xyproto/simplebolt?status.svg)](http://godoc.org/github.com/xyproto/simplebolt)

Simple way to use the [Bolt](https://github.com/coreos/bbolt) database. Similar design to [simpleredis](https://github.com/xyproto/simpleredis).


Online API Documentation
------------------------

[godoc.org](http://godoc.org/github.com/xyproto/simplebolt)


Features and limitations
------------------------

* Supports simple use of lists, hashmaps, sets and key/values.
* Deals mainly with strings.
* Requires Go 1.3 or later.
* The latest version of `gccgo` (7.2.0) is able to compile `simplebolt`, but it does not appear to be able to compile it correctly. There are runtime errors when running `go test`, that work fine when compiling `simplebolt` with the regular Go compiler.

Example usage
-------------

~~~go
package main

import (
	"log"

	"github.com/xyproto/simplebolt"
)

func main() {
	// New bolt database
	db, err := simplebolt.New("bolt.db")
	if err != nil {
		log.Fatalf("Could not create database! %s", err)
	}
	defer db.Close()

	// Create a list named "greetings"
	list, err := simplebolt.NewList(db, "greetings")
	if err != nil {
		log.Fatalf("Could not create a list! %s", err)
	}

	// Add "hello" to the list
	if err := list.Add("hello"); err != nil {
		log.Fatalf("Could not add an item to the list! %s", err)
	}

	// Get the last item of the list
	if item, err := list.GetLast(); err != nil {
		log.Fatalf("Could not fetch the last item from the list! %s", err)
	} else {
		log.Println("The value of the stored item is:", item)
	}

	// Remove the list
	if err := list.Remove(); err != nil {
		log.Fatalf("Could not remove the list! %s", err)
	}
}
~~~

Version, license and author
---------------------------

* License: MIT
* Version: 3.2.0
* Author: Alexander F Rødseth &lt;xyproto@archlinux.org&gt;

