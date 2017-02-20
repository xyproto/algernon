package main

// This source file is for the special case of serving a single file.

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultStaticCacheSize = 128 * MiB

	maxAttemptsAtIncreasingPortNumber = 128

	waitBeforeOpen = time.Millisecond * 200
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

// This is a bit hacky, but it's only used when serving a single static file
func openAfter(wait time.Duration, hostname, colonPort string, https bool, cancelChannel chan bool) {
	// Wait a bit
	time.Sleep(wait)
	select {
	case <-cancelChannel:
		// Got a message on the cancelChannel:
		// don't open the URL with an external application.
		return
	case <-time.After(waitBeforeOpen):
		// Got timeout, assume the port was not busy
		openURL(hostname, colonPort, https)
	}
}

// shortInfo outputs a short string about which file is served where
func shortInfoAndOpen(filename, colonPort string, cancelChannel chan bool) {
	hostname := "localhost"
	if serverHost != "" {
		hostname = serverHost
	}
	log.Info("Serving " + filename + " on http://" + hostname + colonPort)

	if openURLAfterServing {
		go openAfter(waitBeforeOpen, hostname, colonPort, false, cancelChannel)
	}
}

// Convenience function for serving only a single file
// (quick and easy way to view a README.md file)
func serveStaticFile(filename, colonPort string, pongomutex *sync.RWMutex) {
	log.Info("Single file mode. Not using the regular parameters.")

	cancelChannel := make(chan bool, 1)

	shortInfoAndOpen(filename, colonPort, cancelChannel)

	mux := http.NewServeMux()
	// 64 MiB cache, use cache compression, no per-file size limit, use best gzip compression
	preferSpeed = false
	cache := newFileCache(defaultStaticCacheSize, true, 0)
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Server", versionString)
		filePage(w, req, filename, defaultLuaDataFilename, nil, nil, cache, pongomutex)
	})
	HTTPserver := newGracefulServer(mux, false, serverHost+colonPort, 5*time.Second)

	// Attempt to serve just the single file
	if errServe := HTTPserver.ListenAndServe(); errServe != nil {
		// If it fails, try several times, increasing the port by 1 each time
		for i := 0; i < maxAttemptsAtIncreasingPortNumber; i++ {
			if errServe = HTTPserver.ListenAndServe(); errServe != nil {
				cancelChannel <- true
				if !strings.HasSuffix(errServe.Error(), "already in use") {
					// Not a problem with address already being in use
					fatalExit(errServe)
				}
				log.Warn("Address already in use. Using next port number.")
				if newPort, errNext := nextPort(colonPort); errNext != nil {
					fatalExit(errNext)
				} else {
					colonPort = newPort
				}

				// Make a new cancel channel, and use the new URL
				cancelChannel = make(chan bool, 1)
				shortInfoAndOpen(filename, colonPort, cancelChannel)

				HTTPserver = newGracefulServer(mux, false, serverHost+colonPort, 5*time.Second)
			}
		}
		// Several attempts failed
		fatalExit(errServe)
	}
}
