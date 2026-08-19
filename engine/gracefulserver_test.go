package engine

import (
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

// waitForPort waits for a listener to accept connections on the given address
func waitForPort(t *testing.T, addr string) {
	t.Helper()
	for range 100 {
		if isPortListening(addr, 100*time.Millisecond) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("nothing is listening on %s", addr)
}

// TestGracefulServerFinishesOngoingRequests checks that a request that is
// already being handled is allowed to finish when the server is stopped, and
// that a deliberate shutdown is not reported as an error.
func TestGracefulServerFinishesOngoingRequests(t *testing.T) {
	addr := findFreePort(t)

	handling := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		close(handling)
		// Still busy when the shutdown starts
		time.Sleep(200 * time.Millisecond)
		w.Write([]byte("done"))
	})

	var initiated atomic.Bool
	gs := &GracefulServer{
		Server:            &http.Server{Addr: addr, Handler: mux},
		ShutdownInitiated: func() { initiated.Store(true) },
		Timeout:           5 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- gs.ListenAndServe()
	}()
	waitForPort(t, addr)

	body := make(chan string, 1)
	go func() {
		resp, err := http.Get("http://" + addr + "/")
		if err != nil {
			body <- "error: " + err.Error()
			return
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		body <- string(data)
	}()

	<-handling
	gs.interrupt()

	if got := <-body; got != "done" {
		t.Errorf("the ongoing request got %q, want %q", got, "done")
	}
	select {
	case err := <-serveErr:
		if err != nil {
			t.Errorf("ListenAndServe returned %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("ListenAndServe did not return after the shutdown")
	}
	if !gs.Interrupted() {
		t.Error("Interrupted() = false, want true")
	}
	if !initiated.Load() {
		t.Error("ShutdownInitiated was not called")
	}
	if isPortListening(addr, 100*time.Millisecond) {
		t.Errorf("%s is still listening after the shutdown", addr)
	}
}

// TestGracefulServerClosesSlowRequests checks that requests that take longer
// than Timeout do not keep the server from stopping.
func TestGracefulServerClosesSlowRequests(t *testing.T) {
	addr := findFreePort(t)

	handling := make(chan struct{})
	release := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		close(handling)
		<-release
	})

	gs := &GracefulServer{
		Server:  &http.Server{Addr: addr, Handler: mux},
		Timeout: 100 * time.Millisecond,
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- gs.ListenAndServe()
	}()
	waitForPort(t, addr)

	go http.Get("http://" + addr + "/")

	<-handling
	stopped := make(chan struct{})
	go func() {
		gs.interrupt()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Error("the shutdown did not give up on the slow request")
	}
	close(release)

	select {
	case err := <-serveErr:
		if err != nil {
			t.Errorf("ListenAndServe returned %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("ListenAndServe did not return after the shutdown")
	}
}
