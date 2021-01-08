// +build !windows

// Package term manages POSIX terminals. As POSIX terminals are connected to,
// or emulate, a UART, this package also provides control over the various
// UART and serial line parameters.
package term

import (
	"io"
	"os"

	"github.com/pkg/term/termios"
	"golang.org/x/sys/unix"
)

const (
	NONE     = iota // flow control off
	XONXOFF         // software flow control
	HARDWARE        // hardware flow control
)

// Read reads up to len(b) bytes from the terminal. It returns the number of
// bytes read and an error, if any. EOF is signaled by a zero count with
// err set to io.EOF.
func (t *Term) Read(b []byte) (int, error) {
	n, e := unix.Read(t.fd, b)
	if n < 0 {
		n = 0
	}
	if n == 0 && len(b) > 0 && e == nil {
		return 0, io.EOF
	}
	if e != nil {
		return n, &os.PathError{
			Op:   "read",
			Path: t.name,
			Err:  e}
	}
	return n, nil
}

// SetOption takes one or more option function and applies them in order to Term.
func (t *Term) SetOption(options ...func(*Term) error) error {
	for _, opt := range options {
		if err := opt(t); err != nil {
			return err
		}
	}
	return nil
}

// Write writes len(b) bytes to the terminal. It returns the number of bytes
// written and an error, if any. Write returns a non-nil error when n !=
// len(b).
func (t *Term) Write(b []byte) (int, error) {
	n, e := unix.Write(t.fd, b)
	if n < 0 {
		n = 0
	}
	if n != len(b) {
		return n, io.ErrShortWrite
	}
	if e != nil {
		return n, &os.PathError{
			Op:   "write",
			Path: t.name,
			Err:  e,
		}
	}
	return n, nil
}

// Available returns how many bytes are unused in the buffer.
func (t *Term) Available() (int, error) {
	return termios.Tiocinq(uintptr(t.fd))
}

// Buffered returns the number of bytes that have been written into the current buffer.
func (t *Term) Buffered() (int, error) {
	return termios.Tiocoutq(uintptr(t.fd))
}
