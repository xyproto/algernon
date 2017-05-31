// +build darwin,cgo dragonfly,cgo freebsd,cgo linux,cgo nacl,cgo netbsd,cgo openbsd,cgo solaris,cgo

package platformdep

import (
	"os/signal"
	"syscall"
)

func IgnoreTerminalResizeSignal() {
	signal.Ignore(syscall.SIGWINCH)
}
