package recwatch

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type (
	// TimeEventMap stores filesystem events
	TimeEventMap map[time.Time]Event
)

// Customizable functions, as exported variables. Can be se to "nil".

// LogInfo logs a message as information
var LogInfo = func(msg string) {
	log.Println(msg)
}

// LogError logs a message as an error, but does not end the program
var LogError = func(err error) {
	log.Println(err.Error())
}

// FatalExit ends the program after logging a message
var FatalExit = func(err error) {
	log.Fatalln(err)
}

// Exists checks if the given path exists, using os.Stat
var Exists = func(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// SetVerbose can be used to enable or disable logging of incoming events
func SetVerbose(enabled bool) {
	if enabled {
		LogInfo = func(msg string) {
			log.Println(msg)
		}
	} else {
		LogInfo = nil
	}
}

// RemoveOldEvents can remove old filesystem events, after a certain duration.
// Needs to be called within a mutex!
func RemoveOldEvents(events *TimeEventMap, maxAge time.Duration) {
	now := time.Now()
	// Cutoff time
	longTimeAgo := now.Add(-maxAge)
	// Loop through the events and delete the old ones
	for t := range *events {
		if t.Before(longTimeAgo) {
			delete(*events, t)
		}
	}
}

// CollectFileChangeEvents gathers filesystem events in a way that web handle functions can use
func CollectFileChangeEvents(watcher *RecursiveWatcher, mut *sync.Mutex, events TimeEventMap, maxAge time.Duration) {
	go func() {
		for {
			select {
			case ev := <-watcher.Events:
				mut.Lock()
				// Remove old events
				RemoveOldEvents(&events, maxAge)
				// Save the event with the current time
				events[time.Now()] = Event(ev)
				mut.Unlock()
			case err := <-watcher.Errors:
				if LogError != nil {
					LogError(err)
				}
			}
		}
	}()
}

// GenFileChangeEvents creates an SSE event whenever a file in the server directory changes.
//
// Uses the following HTTP headers:
//   Content-Type: text/event-stream;charset=utf-8
//   Cache-Control: no-cache
//   Connection: keep-alive
//   Access-Control-Allow-Origin: (custom value)
//
// The "Access-Control-Allow-Origin" header uses the value that is passed in the "allowed" argument.
//
func GenFileChangeEvents(events TimeEventMap, mut *sync.Mutex, maxAge time.Duration, allowed string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream;charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", allowed)

		var id uint64

		for {
			func() { // Use an anonymous function, just for using "defer"
				mut.Lock()
				defer mut.Unlock()
				if len(events) > 0 {
					// Remove old keys
					RemoveOldEvents(&events, maxAge)
					// Sort the events by the registered time
					var keys timeKeys
					for k := range events {
						keys = append(keys, k)
					}
					sort.Sort(keys)
					prevname := ""
					for _, k := range keys {
						ev := events[k]
						if LogInfo != nil {
							LogInfo("EVENT " + ev.String())
						}
						// Avoid sending several events for the same filename
						if ev.Name != prevname {
							// Send an event to the client
							WriteEvent(w, &id, ev.Name, true)
							id++
							prevname = ev.Name
						}
					}
				}
			}()
			// Wait for old events to be gone, and new to appear
			time.Sleep(maxAge)
		}
	}
}

// WriteEvent writes SSE events to the given ResponseWriter.
// id can be nil.
func WriteEvent(w http.ResponseWriter, id *uint64, message string, flush bool) {
	var buf bytes.Buffer
	if id != nil {
		buf.WriteString(fmt.Sprintf("id: %v\n", *id))
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

// Flush can flush the given ResponseWriter.
// Returns false if it wasn't an http.Flusher.
func Flush(w http.ResponseWriter) bool {
	flusher, ok := w.(http.Flusher)
	if ok {
		flusher.Flush()
	}
	return ok
}

// EventServer serves events on a dedicated port.
// addr is the host address ([host][:port])
// The filesystem events are gathered independently of that.
// Allowed can be "*" or a hostname and sets a header in the SSE stream.
func EventServer(path, allowed, eventAddr, eventPath string, refreshDuration time.Duration) {

	if !Exists(path) {
		if FatalExit != nil {
			FatalExit(errors.New(path + " does not exist, can't watch"))
		}
	}

	// Create a new filesystem watcher
	rw, err := NewRecursiveWatcher(path)
	if err != nil {
		if FatalExit != nil {
			FatalExit(err)
		}
	}

	var mut sync.Mutex
	events := make(TimeEventMap)

	// Collect the events for the last n seconds, repeatedly
	// Runs in the background
	CollectFileChangeEvents(rw, &mut, events, refreshDuration)

	// Serve events
	go func() {
		eventMux := http.NewServeMux()
		// Fire off events whenever a file in the server directory changes
		eventMux.HandleFunc(eventPath, GenFileChangeEvents(events, &mut, refreshDuration, allowed))
		eventServer := &http.Server{
			Addr:    eventAddr,
			Handler: eventMux,
		}
		if err := eventServer.ListenAndServe(); err != nil {
			// If we can't serve HTTP on this port, give up
			if FatalExit != nil {
				FatalExit(err)
			}
		}
	}()
}
