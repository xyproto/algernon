// +build !darwin,!dragonfly,!freebsd,!linux,!nacl,!netbsd,!openbsd,!solaris

package main

// UNIX-like systems uses open_unix.go instead.

const (
	defaultOpenExecutable = "start"
)
