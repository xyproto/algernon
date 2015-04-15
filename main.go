// HTTP/2 web server with built-in support for Lua, Markdown, GCSS and Amber.
package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	internallog "log"
	"net/http"
	"os"
	"time"

	"github.com/bradfitz/http2"
	"github.com/xyproto/permissions2"
	"github.com/xyproto/simpleredis"
	"github.com/yuin/gopher-lua"
)

const (
	version_string = "Algernon 0.56"
	description    = "HTTP/2 web server"
)

var (
	// The font that will be used
	// TODO: Make this configurable in server.lua
	font = "<link href='//fonts.googleapis.com/css?family=Lato:300' rel='stylesheet' type='text/css'>"

	// The CSS style that will be used for directory listings and when rendering markdown pages
	// TODO: Make this configurable in server.lua
	style = "body { background-color: #f0f0f0; color: #0b0b0b; font-family: 'Lato', sans-serif; font-weight: 300; margin: 3.5em; font-size: 1.3em; } a { color: #4010010; font-family: courier; } a:hover { color: #801010; } a:active { color: yellow; } h1 { color: #101010; }"

	// List of filenames that should be displayed instead of a directory listing
	// TODO: Make this configurable in the server configuration script
	indexFilenames = []string{"index.lua", "index.html", "index.md", "index.txt", "index.amber"}
)

func NewServerConfiguration(mux *http.ServeMux, http2support bool) *http.Server {
	// Server configuration
	s := &http.Server{
		Addr:           SERVER_ADDR,
		Handler:        mux,
		ReadTimeout:    7 * time.Second,
		WriteTimeout:   7 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	if http2support {
		// Enable HTTP/2 support
		http2.ConfigureServer(s, nil)
	}
	return s
}

func main() {
	// Set several configuration variables, based on the given flags and arguments
	host := handleFlags()

	// Console output
	fmt.Println(banner())
	fmt.Println("--------------------------------------- - - · ·")

	// Request handlers
	mux := http.NewServeMux()

	// TODO: Run a Redis clone in RAM if no server is available.
	if err := simpleredis.TestConnectionHost(REDIS_ADDR); err != nil {
		log.Info("A Redis database is required.")
		log.Fatal(err)
	}

	// New permissions middleware
	perm := permissions.NewWithRedisConf(REDIS_DB, REDIS_ADDR)

	// Lua LState pool
	luapool := &lStatePool{saved: make([]*lua.LState, 0, 4)}
	defer luapool.Shutdown()

	// Register HTTP handler functions
	registerHandlers(mux, SERVER_DIR, perm, luapool)

	// Read server configuration script, if present.
	// The scripts may change global variables.
	for _, filename := range SERVER_CONFIGURATION_FILENAMES {
		if exists(filename) {
			if err := runConfiguration(filename, perm, luapool); err != nil {
				log.Error("Could not use configuration script: " + filename)
				log.Fatal(err)
			}
		}
	}

	// Set the values that has not been set by flags nor scripts (and can be set by both)
	FinalConfiguration(host)

	fmt.Println("--------------------------------------- - - · ·")

	s := NewServerConfiguration(mux, true)

	// If we are not keeping the logs, reduce the verboseness
	http2.VerboseLogs = (SERVER_HTTP2_LOG != "/dev/null")

	// Direct the logging from the http2 package elsewhere
	f, err := os.Open(SERVER_HTTP2_LOG)
	defer f.Close()
	if err != nil {
		// Could not open the SERVER_HTTP2_LOG filename, try using another filename
		f, err := os.OpenFile("http2.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		defer f.Close()
		if err != nil {
			log.Fatalf("Could not write to %s nor %s.", SERVER_HTTP2_LOG, "http2.log")
		}
		internallog.SetOutput(f)
	} else {
		internallog.SetOutput(f)
	}

	err = nil
	if !(SERVE_JUST_HTTP2 || SERVE_JUST_HTTP) {
		log.Info("Serving HTTPS + HTTP/2")
		// Try listening to HTTPS requests
		err = s.ListenAndServeTLS(SERVER_CERT, SERVER_KEY)
		if err != nil {
			log.Warn(err)
		}
	}
	if (err != nil) || SERVE_JUST_HTTP2 {
		log.Info("Serving HTTP/2, not HTTPS + HTTP/2")
		// Try listening to HTTP requests
		if err := s.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	} else if SERVE_JUST_HTTP {
		log.Info("Serving HTTP, not HTTP/2")
		s = NewServerConfiguration(mux, false)
		if err := s.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}
}
