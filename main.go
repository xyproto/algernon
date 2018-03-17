// HTTP/2 web server with built-in support for Lua, Markdown, GCSS, Amber and JSX.
package main

import (
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/engine"
)

const (
	versionString = "Algernon 1.9.1"
	description   = "QUIC Web Server"
)

func main() {
	// Create a new Algernon server. Also initialize log files etc.
	algernon, err := engine.New(versionString, description)
	if err != nil {
		if err == engine.ErrVersion {
			// Exit with error code 0 if --version was specified
			os.Exit(0)
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
