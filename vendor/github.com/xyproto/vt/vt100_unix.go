//go:build !windows && !plan9

package vt

func initTerminal() {
	// No-op on Unix
}

func showCursorHelper(enable bool) {
	// No-op on Unix, handled by ANSI codes
}

func ensureCursorHidden(visible bool) {
	// No-op on Unix
}

func echoOffHelper() bool {
	return true
}
