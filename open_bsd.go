// +build darwin dragonfly freebsd netbsd openbsd

package main

const (
	// There might be better default launchers for some of the above platforms.
	// Pull requests are welcome.
	defaultOpenExecutable = "open"
)
