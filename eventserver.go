package main

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
	"sort"

	"github.com/howeyc/fsnotify"
)

type (
	// For buffering filesystem events
	TimeEventMap map[time.Time]*fsnotify.FileEvent

	// For being able to sort slices of time
	timeKeys []time.Time
)

func (t timeKeys) Len() int {
	return len(t)
}

func (t timeKeys) Less(i, j int) bool {
	return t[i].Before(t[j])
}

func (t timeKeys) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

// Event can write SSE events to the given ResponseWriter
// id can be nil.
func Event(w http.ResponseWriter, id *int, message string, flush bool) {
	var buf bytes.Buffer
	if id != nil {
		buf.WriteString(fmt.Sprintf("id: %d\n", *id))
	}
	for _, msg := range strings.Split(message, "\n") {
		buf.WriteString(fmt.Sprintf("data: %s\n", msg))
	}
	buf.WriteString("\n")
	io.Copy(w, &buf)
	if flush {
		Flush(w)
	}
}

// Attempt to flush the given ResponseWriter.
// Return false if it wasn't a Flusher.
func Flush(w http.ResponseWriter) bool {
	flusher, ok := w.(http.Flusher)
	if ok {
		flusher.Flush()
	}
	return ok
}

// Remove old events
// Must be run within a mutex
func removeOldEvents(events TimeEventMap, maxAge time.Duration) {
	now := time.Now()
	// Cutoff time
	longTimeAgo := now.Add(-maxAge)
	// Loop through the events and delete the old ones
	for t := range events {
		if t.Before(longTimeAgo) {
			log.Warn("DELETING " + events[t].String())
			delete(events, t)
		}
	}
}

// Gather filesystem events in a way that web handle functions can use
func collectFileChangeEvents(watcher *fsnotify.Watcher, mut sync.Mutex, events TimeEventMap, maxAge time.Duration) {
	go func() {
		for {
			select {
			case ev := <-watcher.Event:
				mut.Lock()
				// Remove old events
				removeOldEvents(events, maxAge)
				// Save the event with the current time
				events[time.Now()] = ev
				mut.Unlock()

				log.Info(ev)
			case err := <-watcher.Error:
				log.Error(err)
			}
		}
	}()
}

// Create events whenever a file in the server directory changes
func genFileChangeEvents(events TimeEventMap, mut sync.Mutex, maxAge time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "text/event-stream;charset=utf-8")
		w.Header().Add("Cache-Control", "no-cache")
		w.Header().Add("Connection", "keep-alive")
		w.Header().Add("Access-Control-Allow-Origin", "*")

		// TODO: Check for CloseNotify, for more graceful timeouts (and mut.Unlock?)

		//startTime := time.Now()

		for {
			mut.Lock()
			if len(events) > 0 {
				// Remove old keys
				removeOldEvents(events, maxAge)
				// Sort the events by the registered time
				var keys timeKeys
				for k := range events {
					keys = append(keys, k)
				}
				sort.Sort(keys)
				for _, k := range keys {
					//elapsed := int(startTime.Sub(k).Seconds())
					//log.Info(fmt.Sprintf("Elapsed: %d", elapsed))
					Event(w, nil, events[k].Name, true)
				}
			}
			mut.Unlock()
			// Wait for old events to be gone, and new to appear
			time.Sleep(maxAge)
		}
	}
}

// Serve events on a dedicated port
func bgEventServer() {
	// Create a new filesystem watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	// Start watching the server directory for changes
	err = watcher.Watch(serverDir)
	if err != nil {
		log.Fatal(err)
	}

	var mut sync.Mutex
	events := make(TimeEventMap)

	n := 10 * time.Second
	// Collect the events for the last n seconds, repeatedly
	// Runs in the background
	collectFileChangeEvents(watcher, mut, events, n)

	// Serve events
	go func() {
		if strings.Contains(eventAddr, ":") {
			fields := strings.Split(eventAddr, ":")
			log.Info("Serving events on port " + fields[1])
		}
		eventMux := http.NewServeMux()
		// Fire off events whenever a file in the server directory changes
		eventMux.HandleFunc("/", genFileChangeEvents(events, mut, n))
		eventServer := &http.Server{
			Addr:    eventAddr,
			Handler: eventMux,
			// ReadTimeout:  3600 * time.Second,
			// WriteTimeout: 3600 *	 time.Second,
			// MaxHeaderBytes: 1 << 20,
		}
		if err := eventServer.ListenAndServe(); err != nil {
			// If we can't serve HTTP on this port, give up
			log.Fatal(err)
		}
	}()
}
