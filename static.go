package main

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultStaticCacheSize = 128 * MiB

	maxAttemptsAtIncreasingPortNumber = 64
)

// nextPort increases the port number by 1
func nextPort(colonPort string) (string, error) {
	if !strings.HasPrefix(colonPort, ":") {
		return colonPort, errors.New("colonPort does not start with a colon! \"" + colonPort + "\"")
	}
	num, err := strconv.Atoi(colonPort[1:])
	if err != nil {
		return colonPort, errors.New("Could not convert port number to string: \"" + colonPort[1:] + "\"")
	}
	// Increase the port number by 1, add a colon, convert to string and return
	return ":" + strconv.Itoa(num+1), nil
}

// shortInfo outputs a short string about which file is served where
func shortInfo(filename, colonPort string) {
	hostname := "localhost"
	if serverHost != "" {
		hostname = serverHost
	}
	log.Info("Serving " + filename + " on http://" + hostname + colonPort)
}

// Convenience function for serving only a single file
// (quick and easy way to view a README.md file)
func serveStaticFile(filename, colonPort string) {
	log.Info("Single file mode. Not using the regular parameters.")

	shortInfo(filename, colonPort)

	mux := http.NewServeMux()
	// 64 MiB cache, use cache compression, no per-file size limit, use best gzip compression
	preferSpeed = false
	cache := newFileCache(defaultStaticCacheSize, true, 0)
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Server", versionString)
		filePage(w, req, filename, nil, nil, cache)
	})
	HTTPserver := newGracefulServer(mux, false, serverHost+colonPort, 5*time.Second)

	// Attempt to serve just the single file
	if err := HTTPserver.ListenAndServe(); err != nil {
		// If it fails, try several times, increasing the port by 1 each time
		for i := 0; i < maxAttemptsAtIncreasingPortNumber; i++ {
			if err := HTTPserver.ListenAndServe(); err != nil {
				if !strings.HasSuffix(err.Error(), "already in use") {
					// Not a problem with address already being in use
					fatalExit(err)
				}
				log.Warn("Address already in use. Using next port number.")
				if newPort, err2 := nextPort(colonPort); err2 != nil {
					fatalExit(err)
				} else {
					colonPort = newPort
				}
				shortInfo(filename, colonPort)
				HTTPserver = newGracefulServer(mux, false, serverHost+colonPort, 5*time.Second)
			}
		}
		// Several attempts failed
		fatalExit(err)
	}
}
