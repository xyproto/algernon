//go:build windows

package vt

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"
	"unsafe"

	"golang.org/x/term"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode = kernel32.NewProc("SetConsoleMode")
	procGetStdHandle   = kernel32.NewProc("GetStdHandle")
)

const (
	STD_INPUT_HANDLE                   = ^uintptr(10) // -11
	STD_OUTPUT_HANDLE                  = ^uintptr(11) // -12
	ENABLE_VIRTUAL_TERMINAL_PROCESSING = 0x0004
	ENABLE_VIRTUAL_TERMINAL_INPUT      = 0x0200
	DISABLE_NEWLINE_AUTO_RETURN        = 0x0008
)

var (
	defaultTimeout = 50 * time.Millisecond
	lastKey        int
)

// Key codes for 3-byte sequences (arrows, Home, End)
var keyCodeLookup = map[[3]byte]int{
	{27, 91, 65}:  253, // Up Arrow
	{27, 91, 66}:  255, // Down Arrow
	{27, 91, 67}:  254, // Right Arrow
	{27, 91, 68}:  252, // Left Arrow
	{27, 91, 'H'}: 1,   // Home (Ctrl-A)
	{27, 91, 'F'}: 5,   // End (Ctrl-E)
}

// Key codes for 4-byte sequences (Page Up, Page Down, Home, End)
var pageNavLookup = map[[4]byte]int{
	{27, 91, 49, 126}: 1,   // Home (ESC [1~)
	{27, 91, 52, 126}: 5,   // End (ESC [4~)
	{27, 91, 53, 126}: 251, // Page Up (custom code)
	{27, 91, 54, 126}: 250, // Page Down (custom code)
}

// Key codes for 6-byte sequences (Ctrl-Insert)
var ctrlInsertLookup = map[[6]byte]int{
	{27, 91, 50, 59, 53, 126}: 258, // Ctrl-Insert (ESC [2;5~)
}

// String representations for 3-byte sequences
var keyStringLookup = map[[3]byte]string{
	{27, 91, 65}:  "↑", // Up Arrow
	{27, 91, 66}:  "↓", // Down Arrow
	{27, 91, 67}:  "→", // Right Arrow
	{27, 91, 68}:  "←", // Left Arrow
	{27, 91, 'H'}: "⇱", // Home
	{27, 91, 'F'}: "⇲", // End
}

// String representations for 4-byte sequences
var pageStringLookup = map[[4]byte]string{
	{27, 91, 49, 126}: "⇱", // Home
	{27, 91, 52, 126}: "⇲", // End
	{27, 91, 53, 126}: "⇞", // Page Up
	{27, 91, 54, 126}: "⇟", // Page Down
}

// String representations for 6-byte sequences (Ctrl-Insert)
var ctrlInsertStringLookup = map[[6]byte]string{
	{27, 91, 50, 59, 53, 126}: "⎘", // Ctrl-Insert (Copy)
}

type TTY struct {
	fd                 int
	originalState      *term.State
	timeout            time.Duration
	originalInputMode  uint32
	originalOutputMode uint32
}

// enableVTMode enables VT100/ANSI escape sequence processing on Windows
func enableVTMode() error {
	// Enable VT processing for stdout
	stdout, _, _ := procGetStdHandle.Call(STD_OUTPUT_HANDLE)
	if stdout == 0 {
		return errors.New("could not get stdout handle")
	}

	var outputMode uint32
	ret, _, _ := procGetConsoleMode.Call(stdout, uintptr(unsafe.Pointer(&outputMode)))
	if ret == 0 {
		return errors.New("could not get console output mode")
	}

	outputMode |= ENABLE_VIRTUAL_TERMINAL_PROCESSING | DISABLE_NEWLINE_AUTO_RETURN
	ret, _, _ = procSetConsoleMode.Call(stdout, uintptr(outputMode))
	if ret == 0 {
		return errors.New("could not set console output mode")
	}

	// Enable VT processing for stdin
	stdin, _, _ := procGetStdHandle.Call(STD_INPUT_HANDLE)
	if stdin == 0 {
		return errors.New("could not get stdin handle")
	}

	var inputMode uint32
	ret, _, _ = procGetConsoleMode.Call(stdin, uintptr(unsafe.Pointer(&inputMode)))
	if ret == 0 {
		return errors.New("could not get console input mode")
	}

	inputMode |= ENABLE_VIRTUAL_TERMINAL_INPUT
	ret, _, _ = procSetConsoleMode.Call(stdin, uintptr(inputMode))
	if ret == 0 {
		return errors.New("could not set console input mode")
	}

	return nil
}

// NewTTY opens stdin/stdout for terminal input/output on Windows
func NewTTY() (*TTY, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return nil, errors.New("not a terminal")
	}

	// Enable VT100/ANSI mode for Windows Console
	err := enableVTMode()
	if err != nil {
		// Don't fail if VT mode can't be enabled, just warn
		fmt.Fprintf(os.Stderr, "Warning: Could not enable VT100 mode: %v\n", err)
	}

	originalState, err := term.GetState(fd)
	if err != nil {
		return nil, err
	}

	return &TTY{
		fd:            fd,
		originalState: originalState,
		timeout:       defaultTimeout,
	}, nil
}

// SetTimeout sets a timeout for reading a key
func (tty *TTY) SetTimeout(d time.Duration) {
	tty.timeout = d
}

// Close will restore the terminal state
func (tty *TTY) Close() {
	if tty.originalState != nil {
		term.Restore(tty.fd, tty.originalState)
	}
}

// asciiAndKeyCode processes input into an ASCII code or key code, handling multi-byte sequences like Ctrl-Insert
func asciiAndKeyCode(tty *TTY) (ascii, keyCode int, err error) {
	bytes := make([]byte, 6) // Use 6 bytes to cover longer sequences like Ctrl-Insert
	var numRead int

	// Set the terminal into raw mode
	rawState, err := term.MakeRaw(tty.fd)
	if err != nil {
		return 0, 0, err
	}
	defer term.Restore(tty.fd, rawState)

	// Read bytes from stdin with timeout
	done := make(chan bool, 1)
	go func() {
		numRead, err = os.Stdin.Read(bytes)
		done <- true
	}()

	select {
	case <-done:
		if err != nil {
			return 0, 0, err
		}
	case <-time.After(tty.timeout):
		return 0, 0, errors.New("read timeout")
	}

	// Handle multi-byte sequences
	switch {
	case numRead == 1:
		ascii = int(bytes[0])
	case numRead == 3:
		seq := [3]byte{bytes[0], bytes[1], bytes[2]}
		if code, found := keyCodeLookup[seq]; found {
			keyCode = code
			return
		}
		// Not found, check if it's a printable character
		r, _ := utf8.DecodeRune(bytes[:numRead])
		if unicode.IsPrint(r) {
			ascii = int(r)
		}
	case numRead == 4:
		seq := [4]byte{bytes[0], bytes[1], bytes[2], bytes[3]}
		if code, found := pageNavLookup[seq]; found {
			keyCode = code
			return
		}
	case numRead == 6:
		seq := [6]byte{bytes[0], bytes[1], bytes[2], bytes[3], bytes[4], bytes[5]}
		if code, found := ctrlInsertLookup[seq]; found {
			keyCode = code
			return
		}
	default:
		// Attempt to decode as UTF-8
		r, _ := utf8.DecodeRune(bytes[:numRead])
		if unicode.IsPrint(r) {
			ascii = int(r)
		}
	}

	return
}

// Key reads the keycode or ASCII code and avoids repeated keys
func (tty *TTY) Key() int {
	ascii, keyCode, err := asciiAndKeyCode(tty)
	if err != nil {
		return 0
	}
	var key int
	if keyCode != 0 {
		key = keyCode
	} else {
		key = ascii
	}
	// Don't filter repeated keys - let the application handle key repeats
	return key
}

// String reads a string, handling key sequences and printable characters
func (tty *TTY) String() string {
	bytes := make([]byte, 6)
	var numRead int
	var err error

	// Set the terminal into raw mode
	rawState, err := term.MakeRaw(tty.fd)
	if err != nil {
		return ""
	}
	defer term.Restore(tty.fd, rawState)

	// Read bytes from stdin with timeout
	done := make(chan bool, 1)
	go func() {
		numRead, err = os.Stdin.Read(bytes)
		done <- true
	}()

	select {
	case <-done:
		if err != nil || numRead == 0 {
			return ""
		}
	case <-time.After(tty.timeout):
		return "" // timeout
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
		// Attempt to interpret as UTF-8 string
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
		// For simplicity, just return what we read
		return string(bytes[:numRead])
	}
	return string(bytes[:numRead])
}

// Rune reads a rune, handling special sequences for arrows, Home, End, etc.
func (tty *TTY) Rune() rune {
	bytes := make([]byte, 6)
	var numRead int
	var err error

	// Set the terminal into raw mode
	rawState, err := term.MakeRaw(tty.fd)
	if err != nil {
		return rune(0)
	}
	defer term.Restore(tty.fd, rawState)

	// Read bytes from stdin with timeout
	done := make(chan bool, 1)
	go func() {
		numRead, err = os.Stdin.Read(bytes)
		done <- true
	}()

	select {
	case <-done:
		if err != nil || numRead == 0 {
			return rune(0)
		}
	case <-time.After(tty.timeout):
		return rune(0) // timeout
	}

	switch {
	case numRead == 1:
		return rune(bytes[0])
	case numRead == 3:
		seq := [3]byte{bytes[0], bytes[1], bytes[2]}
		if str, found := keyStringLookup[seq]; found {
			return []rune(str)[0]
		}
		// Attempt to interpret as UTF-8 rune
		r, _ := utf8.DecodeRune(bytes[:numRead])
		return r
	case numRead == 4:
		seq := [4]byte{bytes[0], bytes[1], bytes[2], bytes[3]}
		if str, found := pageStringLookup[seq]; found {
			return []rune(str)[0]
		}
		r, _ := utf8.DecodeRune(bytes[:numRead])
		return r
	case numRead == 6:
		seq := [6]byte{bytes[0], bytes[1], bytes[2], bytes[3], bytes[4], bytes[5]}
		if str, found := ctrlInsertStringLookup[seq]; found {
			return []rune(str)[0]
		}
		r, _ := utf8.DecodeRune(bytes[:numRead])
		return r
	default:
		// Attempt to interpret as UTF-8 rune
		r, _ := utf8.DecodeRune(bytes[:numRead])
		return r
	}
}

// RawMode switches the terminal to raw mode
func (tty *TTY) RawMode() {
	_, _ = term.MakeRaw(tty.fd)
}

// NoBlock sets the terminal to cbreak mode (no-op for golang.org/x/term)
func (tty *TTY) NoBlock() {
	// No-op for golang.org/x/term - raw mode handles this
}

// Restore the terminal to its original state
func (tty *TTY) Restore() {
	if tty.originalState != nil {
		term.Restore(tty.fd, tty.originalState)
	}
}

// Flush flushes the terminal output (no-op)
func (tty *TTY) Flush() {
	// No-op for golang.org/x/term
}

// WriteString writes a string to stdout
func (tty *TTY) WriteString(s string) error {
	_, err := os.Stdout.WriteString(s)
	return err
}

// ReadString reads a string from the TTY with timeout
func (tty *TTY) ReadString() (string, error) {
	// Set up a timeout channel
	timeout := time.After(100 * time.Millisecond) // Short timeout for terminal responses
	resultChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	go func() {
		// Set raw mode temporarily
		_, err := term.MakeRaw(tty.fd)
		if err != nil {
			errorChan <- err
			return
		}
		defer term.Restore(tty.fd, tty.originalState)

		var result []byte
		buffer := make([]byte, 1)

		for {
			n, err := os.Stdin.Read(buffer)
			if err != nil {
				errorChan <- err
				return
			}
			if n > 0 {
				// For terminal responses, look for bell character (0x07) which terminates OSC sequences
				if buffer[0] == 0x07 || buffer[0] == '\a' {
					resultChan <- string(result)
					return
				}
				// Also break on ESC sequence end for some terminals
				if len(result) > 0 && buffer[0] == '\\' && result[len(result)-1] == 0x1b {
					resultChan <- string(result)
					return
				}
				result = append(result, buffer[0])

				// Prevent infinite reading - limit response size
				if len(result) > 512 {
					resultChan <- string(result)
					return
				}
			}
		}
	}()

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return "", err
	case <-timeout:
		// Timeout - return empty string (no error, just no response from terminal)
		return "", nil
	}
}

// PrintRawBytes for debugging raw byte sequences
func (tty *TTY) PrintRawBytes() {
	bytes := make([]byte, 6)
	var numRead int

	// Set the terminal into raw mode
	_, err := term.MakeRaw(tty.fd)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer term.Restore(tty.fd, tty.originalState)

	// Read bytes from stdin
	numRead, err = os.Stdin.Read(bytes)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Raw bytes: %v\n", bytes[:numRead])
}

// ASCII returns the ASCII code of the key pressed
func (tty *TTY) ASCII() int {
	ascii, _, err := asciiAndKeyCode(tty)
	if err != nil {
		return 0
	}
	return ascii
}

// KeyCode returns the key code of the key pressed
func (tty *TTY) KeyCode() int {
	_, keyCode, err := asciiAndKeyCode(tty)
	if err != nil {
		return 0
	}
	return keyCode
}

// WaitForKey waits for ctrl-c, Return, Esc, Space, or 'q' to be pressed
func WaitForKey() {
	// Get a new TTY and start reading keypresses in a loop
	r, err := NewTTY()
	if err != nil {
		panic(err)
	}
	defer r.Close()
	for {
		switch r.Key() {
		case 3, 13, 27, 32, 113:
			return
		}
	}
}
