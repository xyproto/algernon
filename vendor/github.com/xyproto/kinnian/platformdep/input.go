// +build !darwin,!dragonfly,!freebsd,!linux,!nacl,!netbsd,!openbsd,!solaris !cgo

package platformdep

import (
	"github.com/xyproto/term"
)

func GetInput(prompt string) (string, error) {
	return term.Ask(prompt), nil
}

func IgnoreTerminalResizeSignal() {}
