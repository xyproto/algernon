// +build !darwin,!dragonfly,!freebsd,!linux,!nacl,!netbsd,!openbsd,!solaris !cgo

package platformdep

import (
	"github.com/xyproto/term"
)

// GetInput asks the user for keyboard input
func GetInput(prompt string) (string, error) {
	return term.Ask(prompt), nil
}

// IgnoreTerminalResizeSignal does nothing for non-UNIX-related platforms
func IgnoreTerminalResizeSignal() {}
