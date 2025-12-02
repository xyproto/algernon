SimpleMaria
===========

[![GoDoc](https://godoc.org/github.com/xyproto/simplemaria?status.svg)](http://godoc.org/github.com/xyproto/simplemaria)

An easy way to use a MariaDB/MySQL database from Go.

Online API Documentation
------------------------

[godoc.org](http://godoc.org/github.com/xyproto/simplemaria)


Features and limitations
------------------------

* Supports simple use of lists, hashmaps, sets and key/values.
* Deals mainly with strings.
* Uses the [mysql](https://github.com/go-sql-driver/mysql) package.
* Modeled after [simpleredis](https://github.com/xyproto/simpleredis).
* The hash maps behaves like hash maps, but are not backed by actual hashmaps, unlike with [simpleredis](https://github.com/xyproto/simpleredis). This is for keeping compatibility with simpleredis. If performance when scaling up is a concern, simpleredis combined with [redis](https://redis.io) or [Valkey](https://github.com/valkey-io/valkey) might be a better choice.
* MariaDB/MySQL normally has issues with variable size UTF-8 strings, even for for some combinations of characters. This package avoids these problems by compressing and hex encoding the data before storing in the database. This may slow down or speed up the time it takes to access the data, depending on your setup, but it's a safe way to encode *any* string. This behavior is optional and can be disabled with `host.SetRawUTF8(true)`, (to just use `utf8mb4`).


Sample usage
------------

~~~go
package main

import (
    "log"

    "github.com/xyproto/simplemaria"
)

func main() {
    // Check if the simplemaria service is up
    if err := db.TestConnection(); err != nil {
        log.Fatalln("Could not connect to local database. Is the service up and running?")
    }

    // Create a Host, connect to the local db server
    host := db.New()

    // Connecting to a different host/port
    //host := db.NewHost("server:3306/db")

    // Connect to a different db host/port, with a username and password
    // host := db.NewHost("username:password@server:port/db")

    // Close the connection when the function returns
    defer host.Close()

    // Create a list named "greetings"
    list, err := db.NewList(host, "greetings")
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

A MariaDB/MySQL Database must be up and running locally for `go test` to work.

Version, license and author
---------------------------

* Version: 1.3.8
* License: BSD-3
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
