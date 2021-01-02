// QUIC web server with built-in support for Lua, Markdown, Pongo2 and JSX.
package main

import (
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/engine"
)

const (
	versionString = "Algernon 1.12.9"
	description   = "Web Server"
)

func main() {
	// Create a new Algernon server. Also initialize log files etc.
	algernon, err := engine.New(versionString, description)
	if err != nil {
		if err == engine.ErrVersion {
			// Exit with error code 0 if --version was specified
			return
		} else {
			// Exit if there are problems with the fundamental setup
			log.Fatalln(err)
		}
	}

	// Set up a mux
	mux := http.NewServeMux()

	// Serve HTTP, HTTP/2 and/or HTTPS. Quit when done.
	algernon.MustServe(mux)
}
