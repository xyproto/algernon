//go:build windows

package vt

import (
	"os"
)

// SetupResizeHandler sets up terminal resize signal handling on Windows
// Note: Windows Console doesn't have SIGWINCH. Resize detection would require
// polling or Win32 API events. For now, this is a no-op.
func SetupResizeHandler(sigChan chan os.Signal) {
	// No-op on Windows - console resize events are not exposed via signals
	// Apps can poll for size changes if needed
}
