// +build !darwin,!dragonfly,!freebsd,!linux,!nacl,!netbsd,!openbsd,!solaris !cgo

package main

import (
	"github.com/xyproto/term"
)

func getInput(prompt string) (string, error) {
	return term.Ask(prompt), nil
}

func ignoreTerminalResizeSignal() {}
