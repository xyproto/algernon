package main

import (
	log "github.com/sirupsen/logrus"
	"net/http"
)

// Convenience function for serving only a single file
// (quick and easy way to view a README.md file)
func serveStaticFile(filename, colonPort string) {
	log.Info("Serving " + filename + " on " + serverHost + colonPort)
	mux := http.NewServeMux()
	cache := newFileCache(defaultCacheSize)
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Server", versionString)
		filePage(w, req, filename, nil, nil, cache)
	})
	HTTPserver := newServerConfiguration(mux, false, serverHost+colonPort)
	if err := HTTPserver.ListenAndServe(); err != nil {
		// Can't serve HTTP, give up
		fatalExit(err)
	}
}
