// +build darwin,cgo dragonfly,cgo freebsd,cgo linux,cgo nacl,cgo netbsd,cgo openbsd,cgo solaris,cgo

package platformdep

import (
	"os/signal"
	"syscall"
)

// IgnoreTerminalResizeSignal configures UNIX related platforms to ignore SIGWINCH
func IgnoreTerminalResizeSignal() {
	signal.Ignore(syscall.SIGWINCH)
}
