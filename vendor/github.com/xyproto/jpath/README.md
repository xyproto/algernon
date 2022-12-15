# JSON Path [![GoDoc](https://godoc.org/github.com/xyproto/jpath?status.svg)](http://godoc.org/github.com/xyproto/jpath)

A go package and a set of utilities for interacting with arbitrary JSON data.

### Example usage

~~~go
package main

import (
    "fmt"
    "github.com/xyproto/jpath"
    "log"
)

func main() {
    // Some JSON
    data := []byte(`{"a":2, "b":3, "people":{"names": ["Bob", "Alice"]}}`)

    // Create a new *jpath.Node
    document, err := jpath.New(data)
    if err != nil {
        log.Fatal(err)
    }

    // Retrieve the value of "a", as an int
    val := document.Get("a").Int()
    fmt.Println("a is", val)

    // Retrieve the first name, using a path expression
    name := document.GetNode(".people.names[0]").String()
    fmt.Println("The name is", name)
}
~~~

### Installation

    go get github.com/xyproto/jpath/cmd/...

### Path expressions

Several of the available functions takes a simple JSON path expression, like `x.books[1].author`. Only simple expressions using `x` for the root node, names and integer indexes are supported as part of the path. For more advanced JSON path expressions, see [this blog post](http://goessner.net/articles/JsonPath/).

The `SetBranch` method for the `Node` struct also provides a way of accessing JSON nodes, where the JSON names are supplied as a slice of strings.

### Utilities

Four small utilities for interacting with JSON files are included. Note that these deals with strings only, not numbers or anything else!

* jget - for retrieving a string value from a JSON file. Takes a filename and a simple JSON path expression.
  * Example: `jget books.json x[1].author`
* jset - for setting JSON string values in a JSON file. Takes a filename, simple JSON path expression and a string.
  * Example: `jset books.json x[1].author Suzanne`
* jdel - for removing a key from a map in a JSON file. Takes a filename and a simple JSON path expression.
  * Example: `jdel abc.json b`
* jadd - for adding JSON data to a JSON file. Takes a filename, simple JSON path expression and JSON data.
  * Example: `jadd books.json x '{"author": "Joan Grass", "book": "The joys of gardening"}'`

### General information

* Version: 0.6.1
* License: BSD-3
* Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
