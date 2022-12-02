package fgtrace

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/DataDog/gostackparse"
	"github.com/felixge/fgtrace/internal"
)

const (
	defaultFile         = "fgtrace.json"
	defaultHz           = 99
	defaultHTTPDuration = 30 * time.Second
	defaultStateFrames  = StateFramesRoot
)

// Config configures the capturing of traces as well as serving them via http.
// The zero value is a valid configuration.
type Config struct {
	// Hz determines how often the stack traces of all goroutines are captured
	// per second. WithDefaults() sets it to 99 Hz if it is 0.
	Hz int
	// IncludeSelf controls if the trace contains its own internal goroutines.
	// It's disabled by default because they are usually not of interest.
	IncludeSelf bool
	// StateFrames allows adding the state of goroutines as a virtual frame when
	// their stack traces are captured. WithDefaults() sets it to StateFramesRoot
	// if it is "".
	StateFrames StateFrames
	// Dst is the destination for traces created by calling Trace().
	// WithDefaults() sets it to File("fgtrace.json") if it is nil. Also see
	// Writer().
	Dst io.WriteCloser
	// HTTPDuration is the default duration for traces served via ServeHTTP().
	// WithDefaults() sets it to 30s if it is 0. It is ignored by Trace().
	HTTPDuration time.Duration
}

// StateFrames describes if and where virtual goroutine state frames are added.
type StateFrames string

const (
	// StateFramesRoot causes virtual goroutine state frames to be added at the
	// root of stack traces (e.g. above main).
	StateFramesRoot StateFrames = "root"
	// StateFramesLeaf causes virtual goroutine state frames to be added at the
	// leaf of stack traces.
	StateFramesLeaf StateFrames = "leaf"
	// StateFramesNo casuses no virtual goroutine state frames to be added to
	// stack traces.
	StateFramesNo StateFrames = "no"
)

// assert interface implementation
var _ http.Handler = Config{}

// WithDefaults returns a copy of c with default values applied as described in
// the type documentation. This is done automatically by Trace() and
// ServeHTTP(), but can be useful to log the effective configuration.
func (c Config) WithDefaults() Config {
	if c.Dst == nil {
		c.Dst = File(defaultFile)
	}
	if c.Hz == 0 {
		c.Hz = defaultHz
	}
	if c.HTTPDuration == 0 {
		c.HTTPDuration = defaultHTTPDuration
	}
	if c.StateFrames == "" {
		c.StateFrames = defaultStateFrames
	}
	return c
}

// Trace applies WithDefaults to c and starts capturing a trace at c.Hz to
// c.Dst. Callers are responsible for calling Trace.Stop() to finish the trace.
func (c Config) Trace() *Trace {
	t := &Trace{
		c:       c.WithDefaults(),
		stop:    make(chan struct{}),
		stopped: make(chan error),
	}
	t.start()
	return t
}

// ServeHTTP applies WithDefaults to c and serves a trace. The query
// parameters "hz" and "seconds" can be used to overwrite the defaults.
func (c Config) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c = c.WithDefaults()
	c.Dst = Writer(w)

	params := []struct {
		Name string
		Fn   func(val string) error
	}{
		{
			Name: "seconds",
			Fn: func(val string) error {
				seconds, err := strconv.ParseFloat(val, 64)
				if err != nil {
					return err
				} else if seconds <= 0 {
					return errors.New("invalid value")
				}
				c.HTTPDuration = time.Duration(float64(time.Second) * seconds)
				return nil
			},
		},
		{
			Name: "hz",
			Fn: func(val string) error {
				hz, err := strconv.Atoi(val)
				if err != nil {
					return err
				} else if hz <= 0 {
					return errors.New("invalid value")
				}
				c.Hz = hz
				return nil
			},
		},
	}

	for _, p := range params {
		val := r.URL.Query().Get(p.Name)
		if val == "" {
			continue
		} else if err := p.Fn(val); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "bad %s: %q: %s\n", p.Name, val, err)
			return
		}
	}

	defer c.Trace().Stop()
	time.Sleep(c.HTTPDuration)

}

// File is a helper for Config.Dst that returns an io.WriteCloser that creates
// and writes to the file with the given name.
func File(name string) io.WriteCloser {
	return internal.NewFileWriter(name)
}

// Writer is a helper for for Config.Dst that returns an io.WriteCloser that
// writes to w and does nothing when Close() is called.
func Writer(w io.Writer) io.WriteCloser {
	return internal.WriteNopCloser(w)
}

// Trace represents a trace that is being captured.
type Trace struct {
	c       Config            // config for the trace
	err     error             // error that caused the tracer to stop
	stop    chan struct{}     // closed to initiate stop
	stopped chan error        // messaged to confirm stop completed
	enc     *internal.Encoder // trace event format encoder
}

func (t *Trace) start() {
	if t.enc, t.err = internal.NewEncoder(t.c.Dst); t.err != nil {
		return
	} else if t.err = t.enc.CustomMeta("hz", t.c.Hz); t.err != nil {
		return
	}

	go func() { t.stopped <- t.trace() }()
}

// Stop stops the trace, calls Close() on the configured dst and returns nil on
// success. Calling Stop() more than once returns the previous error or an
// error indicating that the tracer has already been stopped.
func (t *Trace) Stop() error {
	if t.err != nil {
		return t.err
	}

	close(t.stop)
	err := <-t.stopped
	// TODO(fg) does the trace format support writing error messages? if yes,
	// we should probably attempt to write the error to the file as well.
	if finishErr := t.enc.Finish(); finishErr != nil && err == nil {
		err = finishErr
	}

	if err != nil {
		t.err = err
	} else {
		// To be returned if Stop() is called more than once.
		t.err = errors.New("tracer is already stopped")
	}

	return err
}

// trace is the background goroutine that takes goroutine profiles and
// converts them to trace events.
func (t *Trace) trace() error {
	var (
		tick           = time.NewTicker(time.Second / time.Duration(t.c.Hz))
		start          = time.Now()
		now            = start
		prevGoroutines = make(map[int]*gostackparse.Goroutine)
		prof           goroutineProfiler
	)
	defer tick.Stop()

	for {
		ts := now.Sub(start).Seconds() * 1e6
		goroutines, err := prof.Goroutines()
		if err != nil {
			return err
		}
		if !t.c.IncludeSelf {
			goroutines = excludeSelf(goroutines)
		}
		addVirualStateFrames(goroutines, t.c.StateFrames)
		currentGoroutines := make(map[int]*gostackparse.Goroutine, len(prevGoroutines))
		for _, current := range goroutines {
			currentGoroutines[current.ID] = current
			prev := prevGoroutines[current.ID]
			if err := t.enc.Encode(ts, prev, current); err != nil {
				return err
			}
		}
		for _, prev := range prevGoroutines {
			if _, ok := currentGoroutines[prev.ID]; ok {
				continue
			}
			if err := t.enc.Encode(ts, prev, nil); err != nil {
				return err
			}
		}
		prevGoroutines = currentGoroutines

		// Sleep until next tick comes up or the tracer is stopped.
		select {
		case now = <-tick.C:
		case <-t.stop:
			ts := time.Since(start).Seconds() * 1e6
			for _, prev := range prevGoroutines {
				if err := t.enc.Encode(ts, prev, nil); err != nil {
					return err
				}
			}
			return nil
		}
	}
}

type goroutineProfiler struct {
	buf []byte
}

func (g *goroutineProfiler) Goroutines() ([]*gostackparse.Goroutine, error) {
	if g.buf == nil {
		g.buf = make([]byte, 16*1024)
	}
	for {
		n := runtime.Stack(g.buf, true)
		if n < len(g.buf) {
			gs, errs := gostackparse.Parse(bytes.NewReader(g.buf[:n]))
			if len(errs) > 0 {
				return gs, errs[0]
			}
			return gs, nil
		}
		g.buf = make([]byte, 2*len(g.buf))
	}
}

func excludeSelf(gs []*gostackparse.Goroutine) []*gostackparse.Goroutine {
	newGS := make([]*gostackparse.Goroutine, 0, len(gs))
	for _, g := range gs {
		include := true
		for _, f := range g.Stack {
			if strings.HasPrefix(f.Func, internal.ModulePath()) {
				include = false
				break
			}
		}
		if include {
			newGS = append(newGS, g)
		}
	}
	return newGS
}

func addVirualStateFrames(gs []*gostackparse.Goroutine, f StateFrames) {
	if f == StateFramesNo {
		return
	}

	for _, g := range gs {
		state := g.State
		if state == "runnable" {
			// Taking a goroutine profile puts all running goroutines into runnable
			// state. So let's indicate that we can't be sure of their real state,
			// but that it's most likely running instead of runnable.
			state = "running/runnable"
		}

		vFrame := &gostackparse.Frame{Func: state, File: "runtime", Line: 1}
		switch f {
		case StateFramesRoot:
			g.Stack = append(g.Stack, vFrame)
		case StateFramesLeaf:
			g.Stack = append([]*gostackparse.Frame{vFrame}, g.Stack...)
		}
	}
}
