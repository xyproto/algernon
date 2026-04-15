//go:build !windows && !plan9

package vt

import (
	"os"
	"os/signal"
	"syscall"
)

func initTerminal() {
	// No-op on Unix
}

func showCursorHelper(enable bool) {
	// No-op on Unix, handled by ANSI codes
}

func echoOffHelper() bool {
	return true
}

// SetupResizeHandler sets up a terminal resize signal handler
func SetupResizeHandler(sigChan chan os.Signal) {
	signal.Notify(sigChan, syscall.SIGWINCH)
}
