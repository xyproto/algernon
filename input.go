// +build !darwin,!dragonfly,!freebsd,!linux,!nacl,!netbsd,!openbsd,!solaris !cgo

package main

import (
	"github.com/xyproto/term"
)

func getInput(prompt string) (string, error) {
	return term.Ask(prompt), nil
}

// TODO: Implement for non-unix systems
func saveHistory(historyFilename string) error { return nil }
func loadHistory(historyFilename string) error { return nil }
func addHistory(line string)                   {}

func ignoreTerminalResizeSignal() {}
