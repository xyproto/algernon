package vt

import (
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
