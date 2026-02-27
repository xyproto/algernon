//go:build !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly

package vt

// KeyString is not supported on this platform, falls back to String.
func (tty *TTY) KeyString() string { return tty.String() }
