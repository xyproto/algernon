// QUIC web server with built-in support for Lua, Markdown, Pongo2 and JSX.
package main

import (
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/engine"
)

const (
	versionString = "Algernon 1.17.5"
	description   = "Web Server"
)

func main() {
	// Create a new Algernon server. Also initialize log files etc.
	algernon, err := engine.New(versionString, description)
	if err != nil {
		if err == engine.ErrVersion {
			// Exit with error code 0 if --version was specified
			return
		}
		// Exit if there are problems with the fundamental setup
		logrus.Fatalln(err)
	}

	// Set up a mux
	mux := http.NewServeMux()

	// Serve HTTP, HTTP/2 and/or HTTPS. Quit when done.
	algernon.MustServe(mux)
}
