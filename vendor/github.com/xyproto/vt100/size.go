//go:build !windows
// +build !windows

package vt100

import (
	"errors"
	"syscall"
	"unsafe"

	"github.com/xyproto/env/v2"
)

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

// Thanks https://stackoverflow.com/a/16576712/131264
func TermSize() (uint, uint, error) {
	ws := &winsize{}
	if retCode, _, _ := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws))); int(retCode) != -1 {
		return uint(ws.Col), uint(ws.Row), nil
	}
	if w, h := env.Int("COLUMNS", 0), env.Int("LINES", 0); w > 0 && h > 0 {
		return uint(w), uint(h), nil
	}
	return 0, 0, errors.New("could not get terminal size")
}

// Convenience function
func ScreenWidth() int {
	w, _, err := TermSize()
	if err != nil {
		return -1
	}
	return int(w)
}

// Convenience function
func ScreenHeight() int {
	_, h, err := TermSize()
	if err != nil {
		return -1
	}
	return int(h)
}
