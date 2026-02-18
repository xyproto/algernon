//go:build plan9

package vt

import (
	"errors"
	"fmt"
	"time"
)

var (
	defaultTimeout = 2 * time.Millisecond
	lastKey        int
)

// StubTerm is a stub for term.Term on unsupported platforms
type StubTerm struct{}

// Available returns 0 bytes available (stub)
func (s *StubTerm) Available() (int, error) {
	return 0, errors.New("TTY is not supported on this platform")
}

// Read is a stub that returns an error
func (s *StubTerm) Read(p []byte) (int, error) {
	return 0, errors.New("TTY is not supported on this platform")
}

// TTY represents a terminal device
type TTY struct {
	timeout time.Duration
}

// NewTTY opens the terminal in raw mode (stub for unsupported platforms)
func NewTTY() (*TTY, error) {
	return nil, errors.New("TTY is not supported on this platform")
}

// SetTimeout sets a timeout for reading a key
func (tty *TTY) SetTimeout(d time.Duration) {
	tty.timeout = d
}

// Close will restore and close the raw terminal
func (tty *TTY) Close() {}

// Key reads the keycode or ASCII code
func (tty *TTY) Key() int { return 0 }

// String reads a string from the TTY
func (tty *TTY) String() string { return "" }

// Rune reads a rune from the TTY
func (tty *TTY) Rune() rune { return rune(0) }

// RawMode switches the terminal to raw mode
func (tty *TTY) RawMode() {}

// NoBlock sets the terminal to non-blocking mode
func (tty *TTY) NoBlock() {}

// Restore the terminal to its original state
func (tty *TTY) Restore() {}

// Flush flushes the terminal output
func (tty *TTY) Flush() {}

// Term returns a stub terminal reader
func (tty *TTY) Term() *StubTerm {
	return &StubTerm{}
}

// WriteString writes a string to the terminal
func (tty *TTY) WriteString(s string) error {
	return errors.New("TTY is not supported on this platform")
}

// ReadString reads a string from the TTY
func (tty *TTY) ReadString() (string, error) {
	return "", errors.New("TTY is not supported on this platform")
}

// PrintRawBytes for debugging raw byte sequences
func (tty *TTY) PrintRawBytes() {
	fmt.Println("TTY is not supported on this platform")
}

// ASCII returns the ASCII code of the key pressed
func (tty *TTY) ASCII() int { return 0 }

// KeyCode returns the key code of the key pressed
func (tty *TTY) KeyCode() int { return 0 }

// WaitForKey waits for a key press (stub)
func WaitForKey() {}
