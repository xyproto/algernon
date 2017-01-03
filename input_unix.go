// +build darwin,cgo dragonfly,cgo freebsd,cgo linux,cgo nacl,cgo netbsd,cgo openbsd,cgo solaris,cgo

package main

import (
	"os/signal"
	"syscall"
)

func ignoreTerminalResizeSignal() {
	signal.Ignore(syscall.SIGWINCH)
}
