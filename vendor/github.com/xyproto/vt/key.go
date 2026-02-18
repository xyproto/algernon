//go:build !windows && !plan9

package vt

import (
	"errors"
	"fmt"
	"strconv"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/sys/unix"
)

var (
	defaultTimeout = 2 * time.Millisecond
	lastKey        int
)

type TTY struct {
	fd      int
	orig    unix.Termios
	timeout time.Duration
}

// clamp restricts v to the range [lo, hi]
func clamp(v, lo, hi int64) int64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// timeoutVals converts d into VMIN and VTIME values, matching the old pkg/term behavior.
// VTIME is in deciseconds (1/10th of a second), so durations under 100ms clamp to 100ms.
// A zero or negative duration means block indefinitely (VMIN=1, VTIME=0).
func timeoutVals(d time.Duration) (uint8, uint8) {
	if d > 0 {
		vtimeDeci := d.Nanoseconds() / 1e6 / 100
		vtime := uint8(clamp(vtimeDeci, 1, 0xff))
		return 0, vtime
	}
	return 1, 0
}

// cfmakeraw sets the termios attributes for raw mode
func cfmakeraw(attr *unix.Termios) {
	attr.Iflag &^= unix.BRKINT | unix.ICRNL | unix.INPCK | unix.ISTRIP | unix.IXON
	attr.Oflag &^= unix.OPOST
	attr.Cflag &^= unix.CSIZE | unix.PARENB
	attr.Cflag |= unix.CS8
	attr.Lflag &^= unix.ECHO | unix.ICANON | unix.IEXTEN | unix.ISIG
	attr.Cc[unix.VMIN] = 1
	attr.Cc[unix.VTIME] = 0
}

// cfmakecbreak sets the termios attributes for cbreak mode
func cfmakecbreak(attr *unix.Termios) {
	attr.Lflag &^= unix.ECHO | unix.ICANON
	attr.Cc[unix.VMIN] = 1
	attr.Cc[unix.VTIME] = 0
}

// tcgetattr gets the current terminal attributes
func tcgetattr(fd int) (unix.Termios, error) {
	t, err := unix.IoctlGetTermios(fd, ioctlGETATTR)
	if err != nil {
		return unix.Termios{}, err
	}
	return *t, nil
}

// tcsetattr sets the terminal attributes
func tcsetattr(fd int, attr *unix.Termios) error {
	return unix.IoctlSetTermios(fd, ioctlSETATTR, attr)
}

// NewTTY opens /dev/tty in raw+cbreak mode with a read timeout
func NewTTY() (*TTY, error) {
	fd, err := unix.Open("/dev/tty", unix.O_NOCTTY|unix.O_CLOEXEC|unix.O_NDELAY|unix.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}

	// Save original terminal state
	orig, err := tcgetattr(fd)
	if err != nil {
		unix.Close(fd)
		return nil, err
	}

	// Apply RawMode, then CBreakMode, then ReadTimeout (same order as old pkg/term)
	var a unix.Termios

	a = orig
	cfmakeraw(&a)
	if err := tcsetattr(fd, &a); err != nil {
		unix.Close(fd)
		return nil, err
	}

	a = orig
	cfmakeraw(&a)
	cfmakecbreak(&a)
	a.Cc[unix.VMIN], a.Cc[unix.VTIME] = timeoutVals(defaultTimeout)
	if err := tcsetattr(fd, &a); err != nil {
		unix.Close(fd)
		return nil, err
	}

	// Clear O_NDELAY
	if err := unix.SetNonblock(fd, false); err != nil {
		unix.Close(fd)
		return nil, err
	}

	return &TTY{fd: fd, orig: orig, timeout: defaultTimeout}, nil
}

// SetTimeout sets the read timeout by updating VMIN/VTIME
func (tty *TTY) SetTimeout(d time.Duration) {
	tty.timeout = d
	a, err := tcgetattr(tty.fd)
	if err != nil {
		return
	}
	a.Cc[unix.VMIN], a.Cc[unix.VTIME] = timeoutVals(d)
	tcsetattr(tty.fd, &a)
}

// Close restores the terminal and closes the file descriptor
func (tty *TTY) Close() {
	tty.Restore()
	unix.Close(tty.fd)
}

// asciiAndKeyCode processes input into an ASCII code or key code
func asciiAndKeyCode(tty *TTY) (ascii, keyCode int, err error) {
	bytes := make([]byte, 6)

	// Set raw mode, cbreak, and timeout before each read
	tty.RawMode()
	tty.NoBlock()
	tty.SetTimeout(tty.timeout)

	// Read bytes from the terminal
	numRead, err := unix.Read(tty.fd, bytes)
	if numRead < 0 {
		numRead = 0
	}

	if err != nil {
		tty.Restore()
		tty.Flush()
		return
	}

	if numRead == 0 {
		tty.Restore()
		tty.Flush()
		return
	}

	// Handle multi-byte sequences
	switch {
	case numRead == 1:
		ascii = int(bytes[0])
	case numRead == 3:
		seq := [3]byte{bytes[0], bytes[1], bytes[2]}
		if code, found := keyCodeLookup[seq]; found {
			keyCode = code
			tty.Restore()
			tty.Flush()
			return
		}
		r, _ := utf8.DecodeRune(bytes[:numRead])
		if unicode.IsPrint(r) {
			ascii = int(r)
		}
	case numRead == 4:
		seq := [4]byte{bytes[0], bytes[1], bytes[2], bytes[3]}
		if code, found := pageNavLookup[seq]; found {
			keyCode = code
			tty.Restore()
			tty.Flush()
			return
		}
	case numRead == 6:
		seq := [6]byte{bytes[0], bytes[1], bytes[2], bytes[3], bytes[4], bytes[5]}
		if code, found := ctrlInsertLookup[seq]; found {
			keyCode = code
			tty.Restore()
			tty.Flush()
			return
		}
	default:
		r, _ := utf8.DecodeRune(bytes[:numRead])
		if unicode.IsPrint(r) {
			ascii = int(r)
		}
	}

	tty.Restore()
	tty.Flush()
	return
}

// Key reads the keycode or ASCII code and avoids repeated keys
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

// String reads a string, handling key sequences and printable characters
func (tty *TTY) String() string {
	bytes := make([]byte, 6)
	tty.RawMode()
	tty.SetTimeout(0) // block until data
	numRead, err := unix.Read(tty.fd, bytes)
	if numRead < 0 {
		numRead = 0
	}
	defer func() {
		tty.Restore()
		tty.Flush()
	}()
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
		// Try to read any remaining bytes with a short timeout
		tty.SetTimeout(defaultTimeout)
		bytes2 := make([]byte, 256)
		numRead2, err := unix.Read(tty.fd, bytes2)
		if numRead2 < 0 {
			numRead2 = 0
		}
		if err != nil || numRead2 == 0 {
			return string(bytes[:numRead])
		}
		return string(append(bytes[:numRead], bytes2[:numRead2]...))
	}
	return string(bytes[:numRead])
}

// Rune reads a rune, handling special sequences for arrows, Home, End, etc.
func (tty *TTY) Rune() rune {
	bytes := make([]byte, 6)
	tty.RawMode()
	tty.SetTimeout(0) // block until data
	numRead, err := unix.Read(tty.fd, bytes)
	if numRead < 0 {
		numRead = 0
	}
	tty.Restore()
	tty.Flush()

	if err != nil || numRead == 0 {
		return rune(0)
	}

	switch {
	case numRead == 1:
		return rune(bytes[0])
	case numRead == 3:
		seq := [3]byte{bytes[0], bytes[1], bytes[2]}
		if str, found := keyStringLookup[seq]; found {
			return []rune(str)[0]
		}
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
		r, _ := utf8.DecodeRune(bytes[:numRead])
		return r
	}
}

// RawMode switches the terminal to raw mode
func (tty *TTY) RawMode() {
	a, err := tcgetattr(tty.fd)
	if err != nil {
		return
	}
	cfmakeraw(&a)
	tcsetattr(tty.fd, &a)
}

// NoBlock sets the terminal to cbreak mode
func (tty *TTY) NoBlock() {
	a, err := tcgetattr(tty.fd)
	if err != nil {
		return
	}
	cfmakecbreak(&a)
	tcsetattr(tty.fd, &a)
}

// Restore the terminal to its original state
func (tty *TTY) Restore() {
	unix.IoctlSetTermios(tty.fd, ioctlFLUSHSET, &tty.orig)
}

// Flush discards pending input/output
func (tty *TTY) Flush() {
	tcflush(tty.fd)
}

// WriteString writes a string to the terminal
func (tty *TTY) WriteString(s string) error {
	n, err := unix.Write(tty.fd, []byte(s))
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("no bytes written to the TTY")
	}
	return nil
}

// ReadString reads all available data from the TTY
func (tty *TTY) ReadString() (string, error) {
	var result []byte
	buf := make([]byte, 128)
	// Temporarily set a short read timeout
	tty.SetTimeout(100 * time.Millisecond)
	defer tty.SetTimeout(tty.timeout)
	for {
		n, err := unix.Read(tty.fd, buf)
		if n < 0 {
			n = 0
		}
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

// PrintRawBytes for debugging raw byte sequences
func (tty *TTY) PrintRawBytes() {
	bytes := make([]byte, 6)
	tty.RawMode()
	tty.SetTimeout(0)
	numRead, err := unix.Read(tty.fd, bytes)
	if numRead < 0 {
		numRead = 0
	}
	tty.Restore()
	tty.Flush()

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
