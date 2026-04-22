//go:build !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly

package vt

// KeyString is not supported on this platform, falls back to ReadKey.
func (tty *TTY) KeyString() string { return tty.ReadKey() }
