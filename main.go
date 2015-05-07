// HTTP/2 web server with built-in support for Lua, Markdown, GCSS, Amber and JSX.
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
	version_string = "Algernon 0.61"
	description    = "HTTP/2 web server"
)

var (
	// The default font
	// TODO: Make this configurable
	font = "<link href='//fonts.googleapis.com/css?family=Lato:300' rel='stylesheet' type='text/css'>"

	// The default CSS style
	// Will be used for directory listings and when rendering markdown pages
	// TODO: Make style.gcss override this also for directory listings
	defaultStyle = "body { background-color: #e7eaed; color: #0b0b0b; font-family: 'Lato', sans-serif; font-weight: 300; margin: 3.5em; font-size: 1.3em; } a { color: #4010010; font-family: courier; } a:hover { color: #801010; } a:active { color: yellow; } h1 { color: #101010; }"

	// List of filenames that should be displayed instead of a directory listing
	// TODO: Make this configurable
	indexFilenames = []string{"index.lua", "index.html", "index.md", "index.txt", "index.amber"}
)

func NewServerConfiguration(mux *http.ServeMux, http2support bool, addr string) *http.Server {
	// Server configuration
	s := &http.Server{
		Addr:           addr,
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

	// Dividing line between the banner and output from any of the configuration scripts
	if len(SERVER_CONFIGURATION_FILENAMES) > 0 {
		fmt.Println("--------------------------------------- - - · ·")
	}

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
	ranServerReadyFunction := FinalConfiguration(host)

	// Dividing line between the banner and output from any of the configuration scripts
	if ranServerReadyFunction {
		fmt.Println("--------------------------------------- - - · ·")
	}

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

	// Decide which protocol to listen to
	switch {
	case SERVE_PROD:
		go func() {
			log.Info("Serving HTTPS + HTTP/2 on port 443")
			HTTPS_server := NewServerConfiguration(mux, true, host+":443")
			// Listen for HTTPS + HTTP/2 requests
			err := HTTPS_server.ListenAndServeTLS(SERVER_CERT, SERVER_KEY)
			if err != nil {
				log.Error(err)
			}
		}()
		log.Info("Serving HTTP on port 80")
		HTTP_server := NewServerConfiguration(mux, false, host+":80")
		if err := HTTP_server.ListenAndServe(); err != nil {
			// If we can't serve regular HTTP on port 80, give up
			log.Fatal(err)
		}
	case SERVE_JUST_HTTP2:
		log.Info("Serving HTTP/2")
		// Listen for HTTP/2 requests
		HTTP2_server := NewServerConfiguration(mux, true, SERVER_ADDR)
		if err := HTTP2_server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	case !(SERVE_JUST_HTTP2 || SERVE_JUST_HTTP):
		log.Info("Serving HTTPS + HTTP/2")
		// Listen for HTTPS + HTTP/2 requests
		HTTPS2_server := NewServerConfiguration(mux, true, SERVER_ADDR)
		err := HTTPS2_server.ListenAndServeTLS(SERVER_CERT, SERVER_KEY)
		if err != nil {
			log.Error(err)
			// If HTTPS failed (perhaps the key + cert are missing), serve
			// plain HTTP instead, by falling through to the next case.
		} else {
			// Don't fall through to serve regular HTTP
			break
		}
		fallthrough
	default:
		log.Info("Serving HTTP")
		HTTP_server := NewServerConfiguration(mux, false, SERVER_ADDR)
		if err := HTTP_server.ListenAndServe(); err != nil {
			// If we can't serve regular HTTP, give up
			log.Fatal(err)
		}
	}
}
