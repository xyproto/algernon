// +build windows, !unix

package consolesize

import (
	"fmt"
	"syscall"
	"unsafe"
)

type (
	short     int16
	word      uint16
	smallRect struct {
		Left   short
		Top    short
		Right  short
		Bottom short
	}
	coord struct {
		X short
		Y short
	}
	consoleScreenBufferInfo struct {
		Size              coord
		CursorPosition    coord
		Attributes        word
		Window            smallRect
		MaximumWindowSize coord
	}
)

var kernel32DLL = syscall.NewLazyDLL("kernel32.dll")
var getConsoleScreenBufferInfoProc = kernel32DLL.NewProc("GetConsoleScreenBufferInfo")

// GetConsoleSize returns the current number of columns and rows in the active console window.
// The return value of this function is in the order of cols, rows.
func GetConsoleSize() (int, int) {
	stdoutHandle := getStdHandle(syscall.STD_OUTPUT_HANDLE)
	var info, err = getConsoleScreenBufferInfo(stdoutHandle)

	if err != nil {
		return 0, 0
	}

	return int(info.Window.Right - info.Window.Left + 1), int(info.Window.Bottom - info.Window.Top + 1)
}

func getError(r1, r2 uintptr, lastErr error) error {
	// If the function fails, the return value is zero.
	if r1 == 0 {
		if lastErr != nil {
			return lastErr
		}
		return syscall.EINVAL
	}
	return nil
}

func getStdHandle(stdhandle int) uintptr {
	handle, err := syscall.GetStdHandle(stdhandle)
	if err != nil {
		panic(fmt.Errorf("could not get standard io handle %d", stdhandle))
	}
	return uintptr(handle)
}

func getConsoleScreenBufferInfo(handle uintptr) (*consoleScreenBufferInfo, error) {
	var info consoleScreenBufferInfo
	if err := getError(getConsoleScreenBufferInfoProc.Call(handle, uintptr(unsafe.Pointer(&info)), 0)); err != nil {
		return nil, err
	}
	return &info, nil
}
