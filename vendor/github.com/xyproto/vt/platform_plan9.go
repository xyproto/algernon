//go:build plan9

package vt

import "os"

func initTerminal() {
	// No-op on Plan 9
}

func showCursorHelper(enable bool) {
	// No-op on Plan 9
}

func echoOffHelper() bool {
	return true
}

// SetupResizeHandler is a no-op on Plan 9
func SetupResizeHandler(sigChan chan os.Signal) {}
