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

func TermWidth() uint {
	ws := &winsize{}
	// Thanks https://stackoverflow.com/a/16576712/131264
	if retCode, _, _ := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws))); int(retCode) != -1 {
		return uint(ws.Col)
	}
	if w := env.Int("COLS", 0); w > 0 {
		return uint(w)
	}
	return uint(env.Int("COLUMNS", 79))
}

func TermHeight() uint {
	ws := &winsize{}
	// Thanks https://stackoverflow.com/a/16576712/131264
	if retCode, _, _ := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws))); int(retCode) != -1 {
		return uint(ws.Row)
	}
	return uint(env.Int("LINES", 25))
}

func TermSize() (uint, uint, error) {
	ws := &winsize{}
	// Thanks https://stackoverflow.com/a/16576712/131264
	if retCode, _, _ := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws))); int(retCode) != -1 {
		return uint(ws.Col), uint(ws.Row), nil
	}
	var w uint = 79
	if cols := env.Int("COLS", 0); cols > 0 {
		w = uint(cols)
	} else if cols := env.Int("COLUMNS", 0); cols > 0 {
		w = uint(cols)
	}
	h := uint(env.Int("LINES", 25))
	if h == 0 || w == 0 {
		return 0, 0, errors.New("columns or lines has been set to 0")
	}
	return w, h, nil
}

// Convenience function
func ScreenWidth() int {
	return int(TermWidth())
}

// Convenience function
func ScreenHeight() int {
	return int(TermHeight())
}

// Convenience function
func ScreenSize() (int, int) {
	w, h, err := TermSize()
	if err != nil {
		return -1, -1
	}
	return int(w), int(h)
}

// Convenience function
func MustTermSize() (uint, uint) {
	ws := &winsize{}
	// Thanks https://stackoverflow.com/a/16576712/131264
	if retCode, _, _ := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws))); int(retCode) != -1 {
		return uint(ws.Col), uint(ws.Row)
	}
	var w uint = 79
	if cols := env.Int("COLS", 0); cols > 0 {
		w = uint(cols)
	} else if cols := env.Int("COLUMNS", 0); cols > 0 {
		w = uint(cols)
	}
	return w, uint(env.Int("LINES", 25))
}
