//go:build windows

package vt

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

func initTerminal() {
	if handle, ok := consoleOutHandle(); ok {
		_ = enableVT(handle)
		return
	}
}

func consoleOutHandle() (windows.Handle, bool) {
	handle, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE)
	if err != nil || handle == windows.InvalidHandle || handle == 0 {
		return 0, false
	}
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return 0, false
	}
	return handle, true
}

func enableVT(handle windows.Handle) error {
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return err
	}
	const EnableVirtualTerminalProcessing = 0x0004
	if mode&EnableVirtualTerminalProcessing == 0 {
		return windows.SetConsoleMode(handle, mode|EnableVirtualTerminalProcessing)
	}
	return nil
}

func showCursorHelper(enable bool) {
	if handle, ok := consoleOutHandle(); ok {
		setCursorVis(handle, enable)
	}
}

func setCursorVis(handle windows.Handle, enable bool) bool {
	// windows.GetConsoleCursorInfo takes *ConsoleCursorInfo.
	// If the function is not exported, I have to load it myself.
	// "GetConsoleCursorInfo" in kernel32.

	type ConsoleCursorInfo struct {
		Size    uint32
		Visible int32
	}
	var info ConsoleCursorInfo

	modkernel32 := windows.NewLazySystemDLL("kernel32.dll")
	procGetConsoleCursorInfo := modkernel32.NewProc("GetConsoleCursorInfo")
	procSetConsoleCursorInfo := modkernel32.NewProc("SetConsoleCursorInfo")

	r1, _, _ := procGetConsoleCursorInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&info)))
	if r1 == 0 {
		return false
	}

	if enable {
		info.Visible = 1
		info.Size = 100
	} else {
		info.Visible = 0
	}

	r1, _, _ = procSetConsoleCursorInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&info)))
	return r1 != 0
}

func ensureCursorHidden(visible bool) {
	if !visible {
		ShowCursor(false)
	}
}

func echoOffHelper() bool {
	return false
}
