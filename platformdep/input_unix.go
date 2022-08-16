//go:build unix && cgo

package platformdep

import (
	"os/signal"
	"syscall"
)

// IgnoreTerminalResizeSignal configures UNIX related platforms to ignore SIGWINCH
func IgnoreTerminalResizeSignal() {
	signal.Ignore(syscall.SIGWINCH)
}
