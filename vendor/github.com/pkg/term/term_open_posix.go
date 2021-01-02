// +build !windows,!solaris

package term

import (
	"os"

	"github.com/pkg/term/termios"
	"golang.org/x/sys/unix"
)

// Open opens an asynchronous communications port.
func Open(name string, options ...func(*Term) error) (*Term, error) {
	fd, e := unix.Open(name, unix.O_NOCTTY|unix.O_CLOEXEC|unix.O_NDELAY|unix.O_RDWR, 0666)
	if e != nil {
		return nil, &os.PathError{"open", name, e}
	}

	t := Term{name: name, fd: fd}
	if err := termios.Tcgetattr(uintptr(t.fd), &t.orig); err != nil {
		return nil, err
	}
	if err := t.SetOption(options...); err != nil {
		return nil, err
	}
	return &t, unix.SetNonblock(t.fd, false)
}

// Restore restores the state of the terminal captured at the point that
// the terminal was originally opened.
func (t *Term) Restore() error {
	return termios.Tcsetattr(uintptr(t.fd), termios.TCIOFLUSH, &t.orig)
}
