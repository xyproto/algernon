// Package console provides functions for disabling and disabling output
package console

import (
	"os"
)

// Output can be used for temporarily silencing stdout by redirecting to os.DevNull ("/dev/null" or "NUL")
type Output struct {
	stdout  *os.File
	enabled bool
}

// Disable output to stdout
func (o *Output) Disable() {
	if !o.enabled {
		o.stdout = os.Stdout
		os.Stdout, _ = os.OpenFile(os.DevNull, os.O_WRONLY, 0644)
		o.enabled = true
	}
}

// Enable output to stdout
func (o *Output) Enable() {
	if o.enabled {
		os.Stdout = o.stdout
	}
}
