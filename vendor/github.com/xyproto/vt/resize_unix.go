//go:build !windows && !plan9

package vt

import (
	"os"
	"os/signal"
	"syscall"
)

// SetupResizeHandler sets up a terminal resize signal handler
func SetupResizeHandler(sigChan chan os.Signal) {
	signal.Notify(sigChan, syscall.SIGWINCH)
}
