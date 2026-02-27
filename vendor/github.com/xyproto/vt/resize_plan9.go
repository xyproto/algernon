//go:build plan9

package vt

import "os"

// SetupResizeHandler is a no-op on Plan 9
func SetupResizeHandler(sigChan chan os.Signal) {}
