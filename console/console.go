// Package console provides functions for disabling and disabling output
package console

import (
	"os"
)

type Output struct {
	enabled    bool
	stdout     *os.File
}

// Disable output to stdout. Will close stdout and stderr.
func (o *Output) Disable() {
	os.Stdout.Close()
	os.Stderr.Close()
	o.stdout, _ = os.OpenFile(os.DevNull, os.O_WRONLY, 0644)
	o.enabled = false
}

// Enable output to stdout, if stdout has not been closed
func (o *Output) Enable() {
	o.stdout = os.Stdout
	o.enabled = true
}
