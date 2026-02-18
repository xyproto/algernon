//go:build darwin || freebsd || netbsd || openbsd || dragonfly

package vt

import "golang.org/x/sys/unix"

const (
	ioctlGETATTR  = unix.TIOCGETA
	ioctlSETATTR  = unix.TIOCSETA
	ioctlFLUSHSET = unix.TIOCSETAF
)

// tcflush discards pending input/output (BSD uses TIOCFLUSH with pointer to queue selector)
func tcflush(fd int) error {
	return unix.IoctlSetPointerInt(fd, unix.TIOCFLUSH, unix.TCIOFLUSH)
}
