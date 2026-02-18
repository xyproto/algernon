//go:build !windows && !plan9

package vt

import (
	"os"
	"os/signal"
	"syscall"
)

// SetupResizeHandler sets up terminal resize signal handling on Unix systems
func SetupResizeHandler(sigChan chan os.Signal) {
	signal.Notify(sigChan, syscall.SIGWINCH)
}
