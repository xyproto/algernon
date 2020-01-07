# Simple Bolt [![Build Status](https://travis-ci.com/xyproto/simplebolt.svg?branch=master)](https://travis-ci.com/xyproto/simplebolt) [![GoDoc](https://godoc.org/github.com/xyproto/simplebolt?status.svg)](http://godoc.org/github.com/xyproto/simplebolt) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/simplebolt)](https://goreportcard.com/report/github.com/xyproto/simplebolt)

Simple way to use the [Bolt](https://github.com/coreos/bbolt) database. Similar design to [simpleredis](https://github.com/xyproto/simpleredis).


Online API Documentation
------------------------

[godoc.org](http://godoc.org/github.com/xyproto/simplebolt)


Features and limitations
------------------------

* Supports simple use of lists, hashmaps, sets and key/values.
* Deals mainly with strings.
* Requires Go 1.9 or later.
* Note that `HashMap` is implemented only for API-compatibility with [simpleredis](https://github.com/xyproto/simpleredis), and does not have the same performance profile as the `HashMap` implementation in [simpleredis](https://github.com/xyproto/simpleredis), [simplemaria](https://github.com/xyproto/simplemaria) (MariaDB/MySQL) or [simplehstore](https://github.com/xyproto/simplehstore) (PostgreSQL w/ HSTORE).

Example usage
-------------

~~~go
package main

import (
	"log"

	"github.com/xyproto/simplebolt"
)

func main() {
	// New bolt database struct
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
	if item, err := list.Last(); err != nil {
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
* Version: 4.0.3
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
