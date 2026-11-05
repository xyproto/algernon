package vt

import (
	"os"

	"github.com/xyproto/env/v2"
	"golang.org/x/term"
)

// MustTermSize returns the current terminal width and height.
// The size is read from stdout, stderr, stdin or /dev/tty, whichever is a
// terminal first, so that it is correct also when stdout is redirected.
func MustTermSize() (uint, uint) {
	for _, f := range []*os.File{os.Stdout, os.Stderr, os.Stdin} {
		if w, h, ok := termSize(f); ok {
			return w, h
		}
	}
	if f, err := os.Open("/dev/tty"); err == nil {
		w, h, ok := termSize(f)
		f.Close()
		if ok {
			return w, h
		}
	}

	// Fallback to environment variables
	var w uint = 79
	if cols := env.Int("COLS", 0); cols > 0 {
		w = uint(cols)
	} else if cols := env.Int("COLUMNS", 0); cols > 0 {
		w = uint(cols)
	}
	return w, uint(env.Int("LINES", 25))
}

// termSize returns the size of the terminal that f is connected to, if any
func termSize(f *os.File) (uint, uint, bool) {
	fd := int(f.Fd())
	if !term.IsTerminal(fd) {
		return 0, 0, false
	}
	width, height, err := term.GetSize(fd)
	if err != nil || width <= 0 || height <= 0 {
		return 0, 0, false
	}
	return uint(width), uint(height), true
}
