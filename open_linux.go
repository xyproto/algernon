// +build linux nacl solaris

package main

const (
	// There might be better default launchers for some of the above platforms.
	// Pull requests are welcome.
	defaultOpenExecutable = "xdg-open"
)
