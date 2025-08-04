package vt

import (
	"os"

	"github.com/xyproto/env/v2"
	"golang.org/x/term"
)

// MustTermSize returns the terminal size using golang.org/x/term for cross-platform compatibility
func MustTermSize() (uint, uint) {
	fd := int(os.Stdout.Fd())
	if term.IsTerminal(fd) {
		width, height, err := term.GetSize(fd)
		if err == nil {
			return uint(width), uint(height)
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
