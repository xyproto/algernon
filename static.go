package main

import (
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

const (
	defaultStaticCacheSize = 1024 * 1024 * 64 // 64 MiB
)

// Convenience function for serving only a single file
// (quick and easy way to view a README.md file)
func serveStaticFile(filename, colonPort string) {
	log.Info("Serving " + filename + " on " + serverHost + colonPort)
	mux := http.NewServeMux()
	// 64 MiB cache, use cache compression, no per-file size limit, use best gzip compression
	preferSpeed = false
	cache := newFileCache(defaultStaticCacheSize, true, 0)
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Server", versionString)
		filePage(w, req, filename, nil, nil, cache)
	})
	HTTPserver := newGracefulServer(mux, false, serverHost+colonPort, 5*time.Second)
	if err := HTTPserver.ListenAndServe(); err != nil {
		// Can't serve HTTP, give up
		fatalExit(err)
	}
}
