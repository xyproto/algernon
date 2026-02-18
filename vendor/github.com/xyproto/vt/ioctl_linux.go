//go:build linux || solaris || aix

package vt

import "golang.org/x/sys/unix"

const (
	ioctlGETATTR  = unix.TCGETS
	ioctlSETATTR  = unix.TCSETS
	ioctlFLUSHSET = unix.TCSETSF
)

// tcflush discards pending input/output
func tcflush(fd int) error {
	return unix.IoctlSetInt(fd, unix.TCFLSH, unix.TCIOFLUSH)
}
