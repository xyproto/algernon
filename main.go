// HTTP/2 web server with built-in support for Lua, Markdown, GCSS, Amber and JSX.
package main

import (
	"net/http"

	"github.com/xyproto/kinnian/engine"
)

const (
	versionString = "Algernon 1.4.3"
	description   = "HTTP/2 Web Server"
)

func main() {
	// Create a new Algernon server. Also initialize log files etc.
	algernon := engine.New(versionString, description)

	// Set up a mux
	mux := http.NewServeMux()

	// Serve HTTP, HTTP/2 and/or HTTPS. Quit when done.
	algernon.MustServe(mux)
}
