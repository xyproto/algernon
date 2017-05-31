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
