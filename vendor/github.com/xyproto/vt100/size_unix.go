// +build darwin,cgo dragonfly,cgo freebsd,cgo linux,cgo nacl,cgo netbsd,cgo openbsd,cgo solaris,cgo

package vt100

import (
	"errors"
	"syscall"
	"unsafe"
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
