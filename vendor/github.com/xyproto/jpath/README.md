#JSON Path [![Build Status](https://travis-ci.org/xyproto/jpath.svg?branch=master)](https://travis-ci.org/xyproto/jpath) [![GoDoc](https://godoc.org/github.com/xyproto/jpath?status.svg)](http://godoc.org/github.com/xyproto/jpath)

Interact with arbitrary JSON. Use simple JSON path expressions.

### Simple usage

~~~go
package main

import (
	"fmt"
	"github.com/xyproto/jpath"
	"log"
)

func main() {
	// Some JSON
	data := []byte(`{"a":2, "b":3}`)

	// Create a new *jpath.Node
	js, err := jpath.New(data)
	if err != nil {
		log.Fatal(err)
	}

	// Retrieve the value of "a", as an int
	val := js.Get("a").Int()

	// Output the result
	fmt.Println("a is", val)
}
~~~

### JSON path expressions

Several of the available functions takes a simple JSON path expression, like `x.books[1].author`. Only simple expressions using `x` for the root node, names and integer indexes are supported as part of the path. For more advanced JSON path expressions, see [this blog post](http://goessner.net/articles/JsonPath/).

The `SetBranch` method for the `Node` struct also provides a way of accessing JSON nodes, where the JSON names are supplied as a slice of strings.

### Requirements

* go >= 1.2

### Utilities

Four small utilities for interacting with JSON files are included:

* jget - for retrieving a string value from a JSON file. Takes a filename and a simple JSON path expression.
  * Example: `jget books.json x[1].author`
* jset - for setting JSON string values in a JSON file. Takes a filename, simple JSON path expression and a string.
  * Example: `jset books.json x[1].author Catniss`
* jdel - for removing a key from a map in a JSON file. Takes a filename and a simple JSON path expression.
  * Example: `jdel abc.json b`
* jadd - for adding JSON data to a JSON file. Takes a filename, simple JSON path expression and JSON data.
  * Example: `jadd books.json x '{"author": "Joan Grass", "book": "The joys of gardening"}'`

General information
-------------------

* Version: 0.5
* License: MIT
* Alexander F Rødseth
