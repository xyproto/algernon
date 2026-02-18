//go:build plan9

package vt

func initTerminal() {
	// No-op on Plan 9
}

func showCursorHelper(enable bool) {
	// No-op
}

func ensureCursorHidden(visible bool) {
	// No-op
}

func echoOffHelper() bool {
	return true
}
