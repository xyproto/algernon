package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/bradfitz/http2"
	"github.com/xyproto/permissions2"
	"github.com/xyproto/simpleredis"
	"github.com/yuin/gopher-lua"
)

const (
	version_string                = "Algernon 0.49"
	server_configuration_filename = "server.lua"
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
	indexFilenames = []string{"index.lua", "index.html", "index.md", "index.txt"}

	// Configuration that is exposed to the server configuration script
	SERVER_DIR, SERVER_ADDR, SERVER_CERT, SERVER_KEY, SERVER_CONF_SCRIPT string

	// Redis configuration
	REDIS_ADDR string
	REDIS_DB   int
)

func main() {
	flag.StringVar(&SERVER_DIR, "dir", ".", "Server directory")
	flag.StringVar(&SERVER_ADDR, "addr", ":3000", "Server [host][:port] (ie \":443\")")
	flag.StringVar(&SERVER_CERT, "cert", "cert.pem", "Server certificate")
	flag.StringVar(&SERVER_KEY, "key", "key.pem", "Server key")
	flag.StringVar(&REDIS_ADDR, "redis", ":6379", "Redis [host][:port] (ie \":6379\")")
	flag.IntVar(&REDIS_DB, "dbindex", 0, "Redis database index")
	flag.StringVar(&SERVER_CONF_SCRIPT, "conf", "server.lua", "Server configuration")

	flag.Parse()

	// For backwards compatibility with earlier versions of algernon

	if len(flag.Args()) >= 1 {
		SERVER_DIR = flag.Args()[0]
	}
	if len(flag.Args()) >= 2 {
		SERVER_ADDR = flag.Args()[1]
	}
	if len(flag.Args()) >= 3 {
		SERVER_CERT = flag.Args()[2]
	}
	if len(flag.Args()) >= 4 {
		SERVER_KEY = flag.Args()[3]
	}
	if len(flag.Args()) >= 5 {
		REDIS_ADDR = flag.Args()[4]
	}
	if len(flag.Args()) >= 6 {
		// Convert the dbindex from string to int
		dbindex, err := strconv.Atoi(flag.Args()[5])
		if err != nil {
			REDIS_DB = dbindex
		}
	}

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
	// May change global variables.
	// TODO: Check if the file is present before trying to run it
	if exists(server_configuration_filename) {
		if runConfiguration(server_configuration_filename, perm, luapool) != nil {
			log.Fatalln("Could not use: " + server_configuration_filename)
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
