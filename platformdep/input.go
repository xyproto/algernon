// +build !darwin,!dragonfly,!freebsd,!linux,!netbsd,!openbsd,!solaris !cgo

package platformdep

import (
	"github.com/xyproto/ask"
)

// GetInput asks the user for keyboard input
func GetInput(prompt string) (string, error) {
	return ask.Ask(prompt), nil
}

// IgnoreTerminalResizeSignal does nothing for non-UNIX-related platforms
func IgnoreTerminalResizeSignal() {}
