package main

import (
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xyproto/recwatch"
)

type (
	// TimeEventMap stores filesystem events
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

// Flush can flush the given ResponseWriter.
// Returns false if it wasn't an http.Flusher.
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
func genFileChangeEvents(events TimeEventMap, mut *sync.Mutex, maxAge time.Duration, allowed string) http.HandlerFunc {
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
						if verboseMode {
							log.Info("EVENT " + ev.String())
						}
						// Avoid sending several events for the same filename
						if ev.Name != prevname {
							// Send an event to the client
							Event(w, &id, ev.Name, true)
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

// EventServer serves events on a dedicated port.
// addr is the host address ([host][:port])
// urlPath is the path to handle (ie /fs)
// refresh is how often the event buffer should be checked and cleared.
// The filesystem events are gathered independently of that.
// Allowed can be "*" or a hostname and sets a header in the SSE stream.
func EventServer(addr, urlPath, path string, refresh time.Duration, allowed string) {
	// Create a new filesystem watcher
	rw, err := recwatch.NewRecursiveWatcher(path)
	if err != nil {
		fatalExit(err)
	}

	var mut sync.Mutex
	events := make(TimeEventMap)

	// Collect the events for the last n seconds, repeatedly
	// Runs in the background
	collectFileChangeEvents(rw, &mut, events, refresh)

	if strings.Contains(addr, ":") {
		fields := strings.Split(addr, ":")
		log.Info("Serving filesystem events on port " + fields[1])
	}

	// Serve events
	go func() {
		eventMux := http.NewServeMux()
		// Fire off events whenever a file in the server directory changes
		eventMux.HandleFunc(urlPath, genFileChangeEvents(events, &mut, refresh, allowed))
		eventServer := &http.Server{
			Addr:    addr,
			Handler: eventMux,
			// ReadTimeout:  3600 * time.Second,
			// WriteTimeout: 3600 *	 time.Second,
			// MaxHeaderBytes: 1 << 20,
		}
		if err := eventServer.ListenAndServe(); err != nil {
			// If we can't serve HTTP on this port, give up
			fatalExit(err)
		}
	}()
}

// Insert JavaScript that refreshes the page when the source files changes.
// The JavaScript depends on the event server being available.
// If javascript can not be inserted, return the original data.
func insertAutoRefresh(htmldata []byte) []byte {
	fullHost := eventAddr
	// If the host+port starts with ":", assume it's only the port number
	if strings.HasPrefix(fullHost, ":") {
		// Add the hostname in front
		if serverHost != "" {
			fullHost = serverHost + eventAddr
		} else {
			fullHost = "localhost" + eventAddr
		}
	}
	// Wait 70% of an event duration before starting to listen for events
	multiplier := 0.7
	js := `
    <script>
    if (!!window.EventSource) {
	  window.setTimeout(function() {
        var source = new EventSource(window.location.protocol + '//` + fullHost + defaultEventPath + `');
        source.addEventListener('message', function(e) {
          const path = '/' + e.data;
          if (path.indexOf(window.location.pathname) >= 0) {
            location.reload()
          }
        }, false);
	  }, ` + durationToMS(refreshDuration, multiplier) + `);
	}
    </script>`

	// Reduce the size slightly
	js = strings.TrimSpace(strings.Replace(js, "\n", "", everyInstance))
	// Remove all whitespace that is more than one space
	for strings.Contains(js, "  ") {
		js = strings.Replace(js, "  ", " ", everyInstance)
	}
	// Place the script at the end of the body, if there is a body
	if bytes.Contains(htmldata, []byte("</body>")) {
		return bytes.Replace(htmldata, []byte("</body>"), []byte(js+"</body>"), 1)
	} else if bytes.Contains(htmldata, []byte("<head>")) {
		// If not, place the script in the <head>, if there is a head
		return bytes.Replace(htmldata, []byte("<head>"), []byte("<head>"+js), 1)
	} else if bytes.Contains(htmldata, []byte("<html>")) {
		// If not, place the script in the <html> as a new <head>
		return bytes.Replace(htmldata, []byte("<html>"), []byte("<html><head>"+js+"</head>"), 1)
	}
	// In the unlikely event that no place to insert the JavaScript was found
	return htmldata
}
