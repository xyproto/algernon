// +build gccgo

package engine

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/radovskyb/watcher"
	log "github.com/sirupsen/logrus"
)

// Create events whenever a file in the server directory changes
func genFileChangeEvents(wa *watcher.Watcher, maxAge time.Duration, allowed string, ac *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream;charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", allowed)

		var (
			id       uint64
			prevname string
		)
		for {
			select {
			case event := <-wa.Event:
				if ac.verboseMode {
					log.Info("EVENT " + event.String())
				}
				// Avoid sending several events for the same filename
				if event.String() != prevname {
					// Send an event to the client
					Event(w, &id, event.String(), true)
					id++
					prevname = event.String()
				}
			case err := <-wa.Error:
				log.Error(err)
			case <-wa.Closed:
				break
			}
		}
	}
}

// EventServer serves events on a dedicated port.
// addr is the host address ([host][:port])
// The filesystem events are gathered independently of that.
// Allowed can be "*" or a hostname and sets a header in the SSE stream.
func (ac *Config) EventServer(path, allowed string) {

	if !ac.fs.Exists(path) {
		ac.fatalExit(errors.New(path + " does not exist, can't watch"))
	}

	// Create a new filesystem watcher
	wa := watcher.New()

	// Ignore hidden files
	wa.IgnoreHiddenFiles(true)

	// If SetMaxEvents is not set, the default is to send all events.
	//wa.SetMaxEvents(1)

	// Only notify rename and move events.
	//wa.FilterOps(watcher.Rename, watcher.Move)

	err := wa.AddRecursive(path)
	if err != nil {
		ac.fatalExit(err)
	}

	// Start watching for changes
	if err := wa.Start(ac.refreshDuration); err != nil {
		ac.fatalExit(err)
	}

	// Examine if a port has already been provided in the address
	if strings.Contains(ac.eventAddr, ":") {
		fields := strings.Split(ac.eventAddr, ":")
		log.Info("Serving filesystem events on port " + fields[1])
	}

	// Serve events
	go func() {
		eventMux := http.NewServeMux()
		// Fire off events whenever a file in the server directory changes
		eventMux.HandleFunc(ac.defaultEventPath, genFileChangeEvents(wa, ac.refreshDuration, allowed, ac))
		eventServer := &http.Server{
			Addr:    ac.eventAddr,
			Handler: eventMux,
			// ReadTimeout:  3600 * time.Second,
			// WriteTimeout: 3600 *	 time.Second,
			// MaxHeaderBytes: 1 << 20,
		}
		if err := eventServer.ListenAndServe(); err != nil {
			// If we can't serve HTTP on this port, give up
			ac.fatalExit(err)
		}
	}()
}
