# Mime [![GoDoc](https://godoc.org/github.com/xyproto/mime?status.svg)](http://godoc.org/github.com/xyproto/mime)

Package for retrieving the mime type given an extension.

Features and limitations
------------------------

* Must be given a filename that contains a list of mimetypes followed by extensions. Typically `/etc/mime.types`.
* Will only read the file once, then store the lookup table in memory. This results in fast lookups.
* Has a lookup table for the most common mime types, if no mime information is found.

Example
-------

~~~ go
package main

import (
    "fmt"

    "github.com/xyproto/mime"
)

func main() {
    // Read inn the list of mime types and extensions.
    // Set everything to UTF-8 when writing headers
    m := mime.New("/etc/mime.types", true)

    // Print the mime type for svg.
    fmt.Println(m.Get("svg"))
}
~~~

* Will output: `image/svg+xml`


General information
-------------------

* Version: 0.2.0
* License: MIT
* Alexander F Rødseth
