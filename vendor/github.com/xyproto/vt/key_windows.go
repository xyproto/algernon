//go:build windows

package vt

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
	"unicode"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/term"
)

var (
	defaultTimeout = 2 * time.Millisecond
	lastKey        int
)

type TTY struct {
	fd              int
	orig            *term.State
	timeout         time.Duration
	useConsoleInput bool
	conin           *os.File
	pending         []byte
	escArmed        bool
}

// NewTTY opens the terminal
func NewTTY() (*TTY, error) {
	fd := int(os.Stdin.Fd())
	var conin *os.File

	// Detect console vs PTY first
	handle := windows.Handle(fd)

	// Check if this is a real console before setting raw mode
	var mode uint32
	useConsoleInput := false
	var orig *term.State

	if err := windows.GetConsoleMode(handle, &mode); err == nil {
		// Real Windows console - prefer CONIN$ and use native KEY_EVENT decoding
		if f, err := os.OpenFile("CONIN$", os.O_RDWR, 0); err == nil {
			fd = int(f.Fd())
			conin = f
		}

		useConsoleInput = true
		var err error
		orig, err = term.MakeRaw(fd)
		if err != nil {
			return nil, err
		}

		// Disable VT input for console mode
		const EnableVirtualTerminalInput = 0x0200
		if mode&EnableVirtualTerminalInput != 0 {
			_ = windows.SetConsoleMode(handle, mode&^EnableVirtualTerminalInput)
		}
	} else {
		// PTY mode (Git Bash) - open /dev/tty and use stty for raw mode
		f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err != nil {
			return nil, fmt.Errorf("PTY detected but /dev/tty not accessible: %w", err)
		}
		fd = int(f.Fd())
		conin = f
		orig = nil

		// Use stty to set raw mode on /dev/tty
		cmd := exec.Command("stty", "raw", "-echo", "-ixon", "min", "1", "time", "0")
		cmd.Stdin = f
		cmd.Stdout = f
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			f.Close()
			return nil, fmt.Errorf("stty failed: %w", err)
		}
	}

	return &TTY{
		fd:              fd,
		orig:            orig,
		timeout:         defaultTimeout,
		useConsoleInput: useConsoleInput,
		conin:           conin,
		pending:         make([]byte, 0),
	}, nil
}

// SetTimeout sets a timeout for reading a key.
// Since Windows ReadFile blocks, we might need a workaround for timeouts.
// For now, we store it.
func (tty *TTY) SetTimeout(d time.Duration) {
	tty.timeout = d
}

// Close restores the terminal
func (tty *TTY) Close() {
	tty.Restore()
	if tty.conin != nil {
		_ = tty.conin.Close()
		tty.conin = nil
	}
}

// Key reads the keycode or ASCII code
func (tty *TTY) Key() int {
	ascii, keyCode, err := asciiAndKeyCode(tty)
	if err != nil {
		lastKey = 0
		return 0
	}
	var key int
	if keyCode != 0 {
		key = keyCode
	} else {
		key = ascii
	}
	if key == lastKey {
		lastKey = 0
		return 0
	}
	lastKey = key
	return key
}

// asciiAndKeyCode processes input into an ASCII code or key code
func asciiAndKeyCode(tty *TTY) (ascii, keyCode int, err error) {
	if tty.useConsoleInput {
		return asciiAndKeyCodeConsole(tty)
	}

	// On Windows, we just read bytes. The terminal should be in raw mode sending VT sequences.
	// We use the same logic as Unix but with our read implementation.

	bytes := make([]byte, 6)

	// Read with timeout
	numRead, err := tty.readWithTimeout(bytes)

	if err != nil {
		return 0, 0, err
	}

	if numRead > 0 {
		tty.pending = append(tty.pending, bytes[:numRead]...)
	}

	if len(tty.pending) == 0 {
		return 0, 0, nil
	}

	// Parse stateful pending input (important for Git Bash / PTY where ESC
	// sequences often arrive split across reads).
	switch tty.pending[0] {
	case 27: // ESC prefix
		if len(tty.pending) >= 3 {
			seq3 := [3]byte{tty.pending[0], tty.pending[1], tty.pending[2]}
			if code, found := keyCodeLookup[seq3]; found {
				tty.pending = tty.pending[3:]
				tty.escArmed = false
				return 0, code, nil
			}
		}
		if len(tty.pending) >= 4 {
			seq4 := [4]byte{tty.pending[0], tty.pending[1], tty.pending[2], tty.pending[3]}
			if code, found := pageNavLookup[seq4]; found {
				tty.pending = tty.pending[4:]
				tty.escArmed = false
				return 0, code, nil
			}
		}
		if len(tty.pending) >= 6 {
			seq6 := [6]byte{tty.pending[0], tty.pending[1], tty.pending[2], tty.pending[3], tty.pending[4], tty.pending[5]}
			if code, found := ctrlInsertLookup[seq6]; found {
				tty.pending = tty.pending[6:]
				tty.escArmed = false
				return 0, code, nil
			}
		}

		// If this looks like an in-progress CSI sequence, wait for more bytes.
		if len(tty.pending) == 1 || (len(tty.pending) == 2 && tty.pending[1] == '[') {
			// First idle poll after ESC: arm. Second idle poll: emit ESC.
			if numRead == 0 {
				if tty.escArmed {
					tty.pending = tty.pending[1:]
					tty.escArmed = false
					return 27, 0, nil
				}
				tty.escArmed = true
			}
			return 0, 0, nil
		}

		// Unknown ESC-prefixed sequence: emit ESC and keep remaining bytes.
		tty.pending = tty.pending[1:]
		tty.escArmed = false
		return 27, 0, nil

	default:
		ascii = int(tty.pending[0])
		tty.pending = tty.pending[1:]
		tty.escArmed = false
		return ascii, 0, nil
	}
}

func asciiAndKeyCodeConsole(tty *TTY) (ascii, keyCode int, err error) {
	handle := windows.Handle(tty.fd)

	waitMS := uint32(windows.INFINITE)
	if tty.timeout > 0 {
		waitMS = uint32(tty.timeout.Milliseconds())
		if waitMS == 0 {
			waitMS = 1
		}
	}

	event, err := windows.WaitForSingleObject(handle, waitMS)
	if err != nil {
		return 0, 0, err
	}
	if event == uint32(windows.WAIT_TIMEOUT) {
		return 0, 0, nil
	}

	for {
		// Check if events are available before reading to avoid blocking
		var numEvents uint32
		if err := windows.GetNumberOfConsoleInputEvents(handle, &numEvents); err != nil {
			return 0, 0, err
		}
		if numEvents == 0 {
			return 0, 0, nil
		}

		rec, n, readErr := readOneConsoleInputRecord(handle)
		if readErr != nil {
			return 0, 0, readErr
		}
		if n == 0 {
			return 0, 0, nil
		}

		const keyEvent = 0x0001
		if rec.EventType != keyEvent {
			continue
		}

		ke := *(*KEY_EVENT_RECORD)(unsafe.Pointer(&rec.Event[0]))
		if ke.bKeyDown == 0 {
			continue
		}

		if ascii, keyCode = decodeConsoleKeyEvent(ke); ascii != 0 || keyCode != 0 {
			return ascii, keyCode, nil
		}
	}
}

type KEY_EVENT_RECORD struct {
	bKeyDown          int32
	wRepeatCount      uint16
	wVirtualKeyCode   uint16
	wVirtualScanCode  uint16
	uChar             [2]byte
	dwControlKeyState uint32
}

type INPUT_RECORD struct {
	EventType uint16
	_         [2]byte // alignment padding
	Event     [16]byte
}

func readOneConsoleInputRecord(handle windows.Handle) (INPUT_RECORD, uint32, error) {
	modkernel32 := windows.NewLazySystemDLL("kernel32.dll")
	procReadConsoleInputW := modkernel32.NewProc("ReadConsoleInputW")

	var rec [1]INPUT_RECORD
	var n uint32
	r1, _, _ := procReadConsoleInputW.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&rec[0])),
		1,
		uintptr(unsafe.Pointer(&n)),
	)
	if r1 == 0 {
		return INPUT_RECORD{}, 0, errors.New("ReadConsoleInputW failed")
	}
	return rec[0], n, nil
}

func decodeConsoleKeyEvent(ke KEY_EVENT_RECORD) (ascii, keyCode int) {
	vk := ke.wVirtualKeyCode

	// Ignore modifier-only events.
	if vk == 0x10 || vk == 0x11 || vk == 0x12 {
		return 0, 0
	}

	// Decode arrows/home/end/page keys to package key codes.
	switch vk {
	case 0x26: // VK_UP
		return 0, 253
	case 0x28: // VK_DOWN
		return 0, 255
	case 0x27: // VK_RIGHT
		return 0, 254
	case 0x25: // VK_LEFT
		return 0, 252
	case 0x24: // VK_HOME
		return 0, 1
	case 0x23: // VK_END
		return 0, 5
	case 0x21: // VK_PRIOR / Page Up
		return 0, 251
	case 0x22: // VK_NEXT / Page Down
		return 0, 250
	}

	const (
		leftAltPressed   = 0x0002
		rightAltPressed  = 0x0001
		leftCtrlPressed  = 0x0008
		rightCtrlPressed = 0x0004
	)
	ctrlPressed := ke.dwControlKeyState&(leftCtrlPressed|rightCtrlPressed) != 0
	altPressed := ke.dwControlKeyState&(leftAltPressed|rightAltPressed) != 0

	if ctrlPressed && !altPressed {
		if vk >= 'A' && vk <= 'Z' {
			return int(vk-'A') + 1, 0
		}
	}

	r := rune(ke.uChar[0]) | rune(ke.uChar[1])<<8
	if r != 0 {
		return int(r), 0
	}

	return 0, 0
}

// readWithTimeout implements reading with timeout on Windows
func (tty *TTY) readWithTimeout(b []byte) (int, error) {
	if tty.useConsoleInput {
		return tty.readWithTimeoutConsole(b)
	}
	return tty.readWithTimeoutPTY(b)
}

// readWithTimeoutPTY reads from PTY (Git Bash) - simple ReadFile
func (tty *TTY) readWithTimeoutPTY(b []byte) (int, error) {
	if tty.timeout <= 0 {
		var n uint32
		err := windows.ReadFile(windows.Handle(tty.fd), b, &n, nil)
		return int(n), err
	}

	handle := windows.Handle(tty.fd)
	event, err := windows.WaitForSingleObject(handle, uint32(tty.timeout.Milliseconds()))
	if err != nil {
		return 0, err
	}
	if event == uint32(windows.WAIT_TIMEOUT) {
		return 0, nil
	}

	var n uint32
	err = windows.ReadFile(handle, b, &n, nil)
	return int(n), err
}

// readWithTimeoutConsole reads from Windows console with event filtering
func (tty *TTY) readWithTimeoutConsole(b []byte) (int, error) {
	if tty.timeout <= 0 {
		var n uint32
		err := windows.ReadFile(windows.Handle(tty.fd), b, &n, nil)
		return int(n), err
	}

	handle := windows.Handle(tty.fd)
	event, err := windows.WaitForSingleObject(handle, uint32(tty.timeout.Milliseconds()))
	if err != nil {
		return 0, err
	}
	if event == uint32(windows.WAIT_TIMEOUT) {
		return 0, nil
	}

	type KEY_EVENT_RECORD struct {
		bKeyDown          int32
		wRepeatCount      uint16
		wVirtualKeyCode   uint16
		wVirtualScanCode  uint16
		uChar             [2]byte
		dwControlKeyState uint32
	}
	type INPUT_RECORD struct {
		EventType uint16
		_         [2]byte
		Event     [16]byte
	}

	modkernel32 := windows.NewLazySystemDLL("kernel32.dll")
	procPeekConsoleInputW := modkernel32.NewProc("PeekConsoleInputW")
	procReadConsoleInputW := modkernel32.NewProc("ReadConsoleInputW")

	for {
		var numEvents uint32
		err = windows.GetNumberOfConsoleInputEvents(handle, &numEvents)
		if err != nil {
			break
		}
		if numEvents == 0 {
			return 0, nil
		}

		var events [1]INPUT_RECORD
		var numRead uint32

		r1, _, _ := procPeekConsoleInputW.Call(uintptr(handle), uintptr(unsafe.Pointer(&events[0])), 1, uintptr(unsafe.Pointer(&numRead)))
		if r1 == 0 {
			break
		}
		if numRead == 0 {
			return 0, nil
		}

		first := events[0]
		shouldConsume := false

		const KEY_EVENT = 0x0001

		if first.EventType == KEY_EVENT {
			ke := *(*KEY_EVENT_RECORD)(unsafe.Pointer(&first.Event[0]))

			if ke.bKeyDown == 0 {
				shouldConsume = true
			} else {
				vk := ke.wVirtualKeyCode
				if vk == 0x10 || vk == 0x11 || vk == 0x12 {
					if ke.uChar[0] == 0 && ke.uChar[1] == 0 {
						shouldConsume = true
					}
				}
			}
		} else {
			shouldConsume = true
		}

		if shouldConsume {
			var dummy [1]INPUT_RECORD
			var n uint32
			procReadConsoleInputW.Call(uintptr(handle), uintptr(unsafe.Pointer(&dummy[0])), 1, uintptr(unsafe.Pointer(&n)))
			continue
		}

		break
	}

	var n uint32
	err = windows.ReadFile(handle, b, &n, nil)
	return int(n), err
}

// String reads a string
func (tty *TTY) String() string {
	bytes := make([]byte, 6)
	tty.SetTimeout(0)
	numRead, err := tty.readWithTimeout(bytes)
	if err != nil || numRead == 0 {
		return ""
	}

	switch {
	case numRead == 1:
		r := rune(bytes[0])
		if unicode.IsPrint(r) {
			return string(r)
		}
		return "c:" + strconv.Itoa(int(r))
	case numRead == 3:
		seq := [3]byte{bytes[0], bytes[1], bytes[2]}
		if str, found := keyStringLookup[seq]; found {
			return str
		}
		return string(bytes[:numRead])
	case numRead == 4:
		seq := [4]byte{bytes[0], bytes[1], bytes[2], bytes[3]}
		if str, found := pageStringLookup[seq]; found {
			return str
		}
		return string(bytes[:numRead])
	case numRead == 6:
		seq := [6]byte{bytes[0], bytes[1], bytes[2], bytes[3], bytes[4], bytes[5]}
		if str, found := ctrlInsertStringLookup[seq]; found {
			return str
		}
		fallthrough
	default:
		return string(bytes[:numRead])
	}
}

// Rune reads a rune
func (tty *TTY) Rune() rune {
	ascii, keyCode, err := asciiAndKeyCode(tty)
	if err != nil {
		return rune(0)
	}
	if ascii != 0 {
		return rune(ascii)
	}
	if keyCode != 0 {
		switch keyCode {
		case 253:
			return '↑'
		case 255:
			return '↓'
		case 254:
			return '→'
		case 252:
			return '←'
		case 1:
			return '⇱'
		case 5:
			return '⇲'
		case 251:
			return '⇞'
		case 250:
			return '⇟'
		}
	}
	return rune(0)
}

// RawMode switches the terminal to raw mode
func (tty *TTY) RawMode() {
	if tty.useConsoleInput {
		term.MakeRaw(tty.fd)
	}
	// For PTY mode, raw mode was already set in NewTTY
}

// NoBlock - Windows doesn't easily support non-blocking ReadFile without overlapped IO.
// But we use WaitForSingleObject in readWithTimeout, so we can simulate it by setting timeout to very small.
func (tty *TTY) NoBlock() {
	tty.SetTimeout(1 * time.Millisecond)
}

// Restore the terminal to its original state
func (tty *TTY) Restore() {
	if tty.orig != nil {
		term.Restore(tty.fd, tty.orig)
	}
}

// Flush discards pending input/output
func (tty *TTY) Flush() {
	// Windows FlushConsoleInputBuffer
	windows.FlushConsoleInputBuffer(windows.Handle(tty.fd))
}

// WriteString writes a string to the terminal
func (tty *TTY) WriteString(s string) error {
	_, err := os.Stdout.WriteString(s)
	return err
}

// ReadString reads all available data
func (tty *TTY) ReadString() (string, error) {
	var result []byte
	buf := make([]byte, 128)
	// Temporarily set a short read timeout
	tty.SetTimeout(100 * time.Millisecond)
	defer tty.SetTimeout(tty.timeout)
	for {
		n, err := tty.readWithTimeout(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}
		if err != nil || n == 0 {
			break
		}
	}
	if len(result) == 0 {
		return "", errors.New("no data read from TTY")
	}
	return string(result), nil
}

// PrintRawBytes ...
func (tty *TTY) PrintRawBytes() {}

// ASCII ...
func (tty *TTY) ASCII() int {
	ascii, _, err := asciiAndKeyCode(tty)
	if err != nil {
		return 0
	}
	return ascii
}

// KeyCode ...
func (tty *TTY) KeyCode() int {
	_, keyCode, err := asciiAndKeyCode(tty)
	if err != nil {
		return 0
	}
	return keyCode
}

// WaitForKey ...
func WaitForKey() {
	tty, _ := NewTTY()
	if tty != nil {
		defer tty.Close()
		for {
			k := tty.Key()
			if k == 3 || k == 13 || k == 27 || k == 32 || k == 113 {
				return
			}
		}
	}
}
