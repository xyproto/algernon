//go:build windows

package vt

import (
	"os"
)

// SetupResizeHandler is a no-op on Windows
func SetupResizeHandler(sigChan chan os.Signal) {
	// No-op on Windows
}
