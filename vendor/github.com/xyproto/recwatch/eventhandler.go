package recwatch

import (
	"errors"
	"net/http"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// GenRelFileChangeEvents is like GenFileChangeEvents but emits paths
// relative to watchDir instead of absolute filesystem paths.
func GenRelFileChangeEvents(events TimeEventMap, mut *sync.Mutex, maxAge time.Duration, allowed, watchDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream;charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		if allowed != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowed)
		}

		var id uint64

		for {
			func() {
				mut.Lock()
				defer mut.Unlock()
				if len(events) > 0 {
					RemoveOldEvents(&events, maxAge)
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
						name := ev.Name
						if rel, err := filepath.Rel(watchDir, name); err == nil {
							name = rel
						}
						if name != prevname {
							WriteEvent(w, &id, name, true)
							id++
							prevname = name
						}
					}
				}
			}()
			time.Sleep(maxAge)
		}
	}
}

// EventHandler returns an http.Handler that serves SSE file-change events
// for the given path. Unlike EventServer, it does not start its own
// http.Server — the caller can mount the handler on any mux.
// Event data contains absolute file paths (same as EventServer).
func EventHandler(path, allowed, eventPath string, refreshDuration time.Duration) (http.Handler, error) {
	if !Exists(path) {
		return nil, errors.New(path + " does not exist, can't watch")
	}

	rw, err := NewRecursiveWatcher(path)
	if err != nil {
		return nil, err
	}

	var mut sync.Mutex
	events := make(TimeEventMap)
	CollectFileChangeEvents(rw, &mut, events, refreshDuration)

	mux := http.NewServeMux()
	mux.HandleFunc(eventPath, GenFileChangeEvents(events, &mut, refreshDuration, allowed))
	return mux, nil
}

// RelEventHandler is like EventHandler but emits paths relative to the
// watched directory instead of absolute filesystem paths.
func RelEventHandler(path, allowed, eventPath string, refreshDuration time.Duration) (http.Handler, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	if !Exists(absPath) {
		return nil, errors.New(absPath + " does not exist, can't watch")
	}

	rw, err := NewRecursiveWatcher(absPath)
	if err != nil {
		return nil, err
	}

	var mut sync.Mutex
	events := make(TimeEventMap)
	CollectFileChangeEvents(rw, &mut, events, refreshDuration)

	mux := http.NewServeMux()
	mux.HandleFunc(eventPath, GenRelFileChangeEvents(events, &mut, refreshDuration, allowed, absPath))
	return mux, nil
}

// EventServerHandler is like EventServer but returns the http.Handler
// instead of starting a server. It also emits relative paths. This is
// the recommended replacement for EventServer when embedding the SSE
// endpoint into an existing server.
func EventServerHandler(path, allowed, eventPath string, refreshDuration time.Duration) (http.Handler, error) {
	return RelEventHandler(path, allowed, eventPath, refreshDuration)
}
