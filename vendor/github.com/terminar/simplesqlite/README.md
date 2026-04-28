SimpleSqlite
============

[![GoDoc](https://godoc.org/github.com/terminar/simplesqlite?status.svg)](http://godoc.org/github.com/terminar/simplesqlite)

An easy way to use a SQLite database from Go.

Online API Documentation
------------------------

[godoc.org](http://godoc.org/github.com/xyproto/simplesqlite)


Features and limitations
------------------------

* Supports simple use of lists, hashmaps, sets and key/values.
* Deals mainly with strings.
* Uses the [sqlite](https://github.com/mattn/go-sqlite3) package.
* Modeled after [simplemaria](https://github.com/xyproto/simplemaria).
* The hash maps behaves like hash maps, but are not backed by actual hashmaps, unlike with [simpleredis](https://github.com/xyproto/simpleredis). This is for keeping compatibility with simpleredis. If performance when scaling up is a concern, simpleredis backed by [redis](https://redis.io) might be a better choice.


Sample usage
------------

~~~go
package main

import (
    "log"

    "github.com/terminar/simplesqlite"
)

func main() {
    // Check if the simplesqlite is working
    if err := db.TestConnection(); err != nil {
        log.Fatalln("Could not open database file.")
    }

    // Create a new File
    file := db.New()

    // Use another filename
    //file := db.NewFile("sqlite.db")

    // Close the connection when the function returns
    defer file.Close()

    // Create a list named "greetings"
    list, err := db.NewList(file, "greetings")
    if err != nil {
        log.Fatalln("Could not create list!")
    }

    // Add "hello" to the list, check if there are errors
    if list.Add("hello") != nil {
        log.Fatalln("Could not add an item to list!")
    }

    // Get the last item of the list
    if item, err := list.GetLast(); err != nil {
        log.Fatalln("Could not fetch the last item from the list!")
    } else {
        log.Println("The value of the stored item is:", item)
    }

    // Remove the list
    if list.Remove() != nil {
        log.Fatalln("Could not remove the list!")
    }
}
~~~

Testing
-------

The tests will create a file (sqlite.db) for `go test` to work.

Version, license and author
---------------------------

* Version: 1.0.0
* License: BSD-3
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
* Author: Björn Kalkbrenner &lt;terminar@cyberphoria.org&gt;
