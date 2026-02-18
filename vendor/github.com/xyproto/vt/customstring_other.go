//go:build !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly

package vt

// CustomString is not supported on this platform. Returns an empty string.
func (tty *TTY) CustomString() string { return "" }
