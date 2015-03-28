package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bradfitz/http2"
	"github.com/xyproto/permissions2"
	"github.com/xyproto/simpleredis"
	"github.com/yuin/gopher-lua"
)

const (
	version_string      = "Algernon 0.51"
	description         = "HTTP/2 web server"
	default_server_port = "3000"
	default_redis_port  = "6379"
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

func main() {
	// Set several configuration variables, based on the given flags and arguments
	handleFlags()

	// Console output
	fmt.Println(banner())
	fmt.Println("------------------------------- - - · ·")

	// Request handlers
	mux := http.NewServeMux()

	// TODO: Run a Redis clone in RAM if no server is available.
	if err := simpleredis.TestConnectionHost(REDIS_ADDR); err != nil {
		log.Println(err)
		log.Fatalln("A Redis database is required.")
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
				log.Println("Could not use: " + filename)
				log.Fatalln("Error: " + err.Error())
			}
		}
	}

	// Server configuration
	s := &http.Server{
		Addr:           SERVER_ADDR,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Enable HTTP/2 support
	http2.ConfigureServer(s, nil)

	log.Println("Starting HTTPS server")
	// Try listening to HTTPS requests
	if err := s.ListenAndServeTLS(SERVER_CERT, SERVER_KEY); err != nil {
		log.Println(err)
		log.Println("Starting HTTP server instead")
		// Try listening to HTTP requests
		if err := s.ListenAndServe(); err != nil {
			log.Printf("Fail: %s\n", err)
		}
	}
}
