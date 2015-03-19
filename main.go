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
)

const version_string = "Algernon 0.47"

var (
	// The font that will be used
	// TODO: Make this configurable in server.lua
	font = "<link href='http://fonts.googleapis.com/css?family=Lato:300' rel='stylesheet' type='text/css'>"

	// The CSS style that will be used for directory listings and when rendering markdown pages
	// TODO: Make this configurable in server.lua
	style = "body { background-color: #f0f0f0; color: #0b0b0b; font-family: 'Lato', sans-serif; font-weight: 300; margin: 3.5em; font-size: 1.3em; } a { color: #4010010; font-family: courier; } a:hover { color: #801010; } a:active { color: yellow; } h1 { color: #101010; }"

	// List of filenames that should be displayed instead of a directory listing
	// TODO: Make this configurable in server.lua
	indexFilenames = []string{"index.lua", "index.html", "index.md", "index.txt"}
)

func main() {
	flag.Parse()

	path := "."
	addr := ":3000" // addr := ":443"
	cert := "cert.pem"
	key := "key.pem"
	redis_addr := ":6379"
	redis_dbindex := "0"

	// TODO: Use traditional args/flag handling.
	//       Add support for --help and --version.

	if len(flag.Args()) >= 1 {
		path = flag.Args()[0]
	}
	if len(flag.Args()) >= 2 {
		addr = flag.Args()[1]
	}
	if len(flag.Args()) >= 3 {
		cert = flag.Args()[2]
	}
	if len(flag.Args()) >= 4 {
		key = flag.Args()[3]
	}
	if len(flag.Args()) >= 5 {
		redis_addr = flag.Args()[4]
	}
	if len(flag.Args()) >= 6 {
		redis_dbindex = flag.Args()[5]
	}

	// Convert the dbindex from string to int
	dbindex, err := strconv.Atoi(redis_dbindex)
	if err != nil {
		// Default to 0
		dbindex = 0
	}

	fmt.Println(banner())
	fmt.Println("------------------------------- - - · ·")
	fmt.Println()
	fmt.Println("[arg 1] directory\t", path)
	fmt.Println("[arg 2] server addr\t", addr)
	fmt.Println("[arg 3] cert file\t", cert)
	fmt.Println("[arg 4] key file\t", key)
	fmt.Println("[arg 5] redis addr\t", redis_addr)
	fmt.Println("[arg 6] redis db index\t", dbindex)
	fmt.Println()

	// Request handlers
	mux := http.NewServeMux()

	// New permissions middleware
	perm := permissions.NewWithRedisConf(dbindex, redis_addr)

	registerHandlers(mux, path, perm)

	s := &http.Server{
		Addr:           addr,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Enable HTTP/2 support
	http2.ConfigureServer(s, nil)

	log.Println("Starting HTTPS server")
	if err := s.ListenAndServeTLS(cert, key); err != nil {
		log.Println(err)
		log.Println("Starting HTTP server instead")
		if err := s.ListenAndServe(); err != nil {
			log.Printf("Fail: %s\n", err)
		}
	}
}
