package main

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xyproto/recwatch"
)

// TODO: Consider using channels in a more clever way, to avoid sleeping.
//       Possibly by sending channels over channels.

type (
	// For buffering filesystem events
	TimeEventMap map[time.Time]recwatch.Event

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
func Event(w http.ResponseWriter, id *uint64, message string, flush bool) {
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
func removeOldEvents(events *TimeEventMap, maxAge time.Duration) {
	now := time.Now()
	// Cutoff time
	longTimeAgo := now.Add(-maxAge)
	// Loop through the events and delete the old ones
	for t := range *events {
		if t.Before(longTimeAgo) {
			//log.Warn("DELETING " + (*events)[t].String())
			delete(*events, t)
		}
	}
}

// Gather filesystem events in a way that web handle functions can use
func collectFileChangeEvents(watcher *recwatch.RecursiveWatcher, mut *sync.Mutex, events TimeEventMap, maxAge time.Duration) {
	go func() {
		for {
			select {
			case ev := <-watcher.Events:
				mut.Lock()
				// Remove old events
				removeOldEvents(&events, maxAge)
				// Save the event with the current time
				events[time.Now()] = recwatch.Event(ev)
				mut.Unlock()
				//log.Info(ev)
			case err := <-watcher.Errors:
				log.Error(err)
			}
		}
	}()
}

// Create events whenever a file in the server directory changes
func genFileChangeEvents(events TimeEventMap, mut *sync.Mutex, maxAge time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "text/event-stream;charset=utf-8")
		w.Header().Add("Cache-Control", "no-cache")
		w.Header().Add("Connection", "keep-alive")
		w.Header().Add("Access-Control-Allow-Origin", "*")

		// TODO: Check for CloseNotify, for more graceful timeouts (and mut unlocks?)

		var id uint64

		for {
			mut.Lock()
			if len(events) > 0 {
				// Remove old keys
				removeOldEvents(&events, maxAge)
				// Sort the events by the registered time
				var keys timeKeys
				for k := range events {
					keys = append(keys, k)
				}
				sort.Sort(keys)
				prevname := ""
				for _, k := range keys {
					ev := events[k]
					// Avoid sending several events for the same filename
					if ev.Name != prevname {
						// Send an event to the client
						Event(w, &id, ev.Name, true)
						id++
						prevname = ev.Name
					}
				}
			}
			mut.Unlock()
			// Wait for old events to be gone, and new to appear
			time.Sleep(maxAge)
		}
	}
}

// Serve events on a dedicated port.
// addr is the host address ([host][:port])
// urlPath is the path to handle (ie /fs)
// refresh is how often the event buffer should be checked and cleared.
// The filesystem events are gathered independently of that.
func EventServer(addr, urlPath, path string, refresh time.Duration) {
	// Create a new filesystem watcher
	rw, err := recwatch.NewRecursiveWatcher(path)
	if err != nil {
		log.Fatal(err)
	}

	var mut sync.Mutex
	events := make(TimeEventMap)

	// Collect the events for the last n seconds, repeatedly
	// Runs in the background
	collectFileChangeEvents(rw, &mut, events, refresh)

	// Serve events
	go func() {
		if strings.Contains(addr, ":") {
			fields := strings.Split(addr, ":")
			log.Info("Serving filesystem events on port " + fields[1])
		}
		eventMux := http.NewServeMux()
		// Fire off events whenever a file in the server directory changes
		eventMux.HandleFunc(urlPath, genFileChangeEvents(events, &mut, refresh))
		eventServer := &http.Server{
			Addr:    addr,
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

// Insert JavaScript that refreshes the page when the source files changes.
// Depends on the event server.
func linkToAutoRefresh(htmldata []byte) []byte {
	fullHost := eventAddr
	if strings.HasPrefix(fullHost, ":") {
		fullHost = "localhost" + eventAddr
	}
	if bytes.Contains(htmldata, []byte("<head>")) {
		return bytes.Replace(htmldata, []byte("<head>"), []byte(`<head>
	    <script>
          if (!!window.EventSource) {
            var source = new EventSource(window.location.protocol + '//`+fullHost+`/fs');
            source.addEventListener('message', function(e) {
              const path = '/' + e.data
              if (path.indexOf(window.location.pathname) >= 0) { location.reload() }
            }, false);
          }
	    </script>`), 1)
	}
	// If no <head> were found
	return htmldata
}
