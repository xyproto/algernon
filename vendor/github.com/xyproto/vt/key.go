//go:build !windows && !plan9

package vt

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/sys/unix"
)

var (
	defaultTimeout = 100 * time.Millisecond // VTIME resolution is 1 decisecond; anything less clamps to 100ms
)

type TTY struct {
	fd      int
	orig    unix.Termios
	timeout time.Duration
	lastKey int
	// pending holds input bytes that were read from the terminal but not yet
	// consumed by a String()/Key() call. When a user holds down a key (or
	// pastes text), a single unix.Read can return multiple queued key
	// sequences; we parse the first one and stash the rest here so the next
	// call returns them as separate keys instead of concatenating everything
	// into one giant literal string.
	pending []byte
	// reader, when non-nil, replaces the terminal file descriptor as the
	// source of input bytes. Set via NewTTYFromReader for scripted / test
	// input. While non-nil, all termios-touching methods (RawMode, NoBlock,
	// Restore, Flush, SetTimeout, Poll, Close, ...) become no-ops and byte
	// reads go through readBytes instead of unix.Read.
	reader io.Reader
}

// readBytes is the single byte-read entry point used by ReadKey, Rune and
// asciiAndKeyCode. When a mock reader has been installed via
// NewTTYFromReader it is used instead of the terminal file descriptor.
func (tty *TTY) readBytes(buf []byte) (int, error) {
	if tty.reader != nil {
		return tty.reader.Read(buf)
	}
	return unix.Read(tty.fd, buf)
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

// timeoutVals converts d into VMIN and VTIME values.
// VTIME is in deciseconds (1/10th of a second).
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

// SetTimeout sets the read timeout.
// Returns the previous timeout.
func (tty *TTY) SetTimeout(d time.Duration) (time.Duration, error) {
	if d == tty.timeout {
		return d, nil
	}
	savedTimeout := tty.timeout
	if tty.reader != nil {
		tty.timeout = d
		return savedTimeout, nil
	}
	if err := tty.SetTimeoutNoSave(d); err != nil {
		return 0, err
	}
	tty.timeout = d
	return savedTimeout, nil
}

// SetTimeoutNoSave sets the read timeout without saving the previous value
func (tty *TTY) SetTimeoutNoSave(d time.Duration) error {
	if tty.reader != nil {
		return nil
	}
	a, err := tcgetattr(tty.fd)
	if err != nil {
		return err
	}
	a.Cc[unix.VMIN], a.Cc[unix.VTIME] = timeoutVals(d)
	tcsetattr(tty.fd, &a)
	return nil
}

// Close restores the terminal and closes the file descriptor
func (tty *TTY) Close() {
	if tty.reader != nil {
		if c, ok := tty.reader.(io.Closer); ok {
			_ = c.Close()
		}
		return
	}
	tty.Restore()
	unix.Close(tty.fd)
}

// HasPendingInput reports whether ReadKey would return another key without
// having to wait — either bytes are already buffered inside the TTY or more
// bytes are readable from the file descriptor right now. Useful for frame
// skipping: if more input is pending, the current frame can be dropped in
// favour of the next one.
func (tty *TTY) HasPendingInput() bool {
	if len(tty.pending) > 0 {
		return true
	}
	if tty.reader != nil {
		return false
	}
	ok, err := tty.Poll(0)
	if err != nil {
		return false
	}
	return ok
}

// Poll checks if there is data available to read from the TTY within the given timeout.
// Returns true if data is available, false if the timeout was reached.
func (tty *TTY) Poll(d time.Duration) (bool, error) {
	if tty.reader != nil {
		// No cross-platform way to peek an arbitrary io.Reader; assume data
		// is available (the reader will block on its own if not).
		return true, nil
	}
	var readfds unix.FdSet
	readfds.Bits[tty.fd/64] |= 1 << (uint(tty.fd) % 64)

	var tv *unix.Timeval
	if d >= 0 {
		t := unix.NsecToTimeval(d.Nanoseconds())
		tv = &t
	}

	for {
		n, err := unix.Select(tty.fd+1, &readfds, nil, nil, tv)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return false, err
		}
		return n > 0, nil
	}
}

// asciiAndKeyCode processes input into an ASCII or key code
func asciiAndKeyCode(tty *TTY) (ascii, keyCode int, err error) {
	bytes := make([]byte, 6)

	// Set raw mode, cbreak, and timeout before each read
	tty.RawMode()
	tty.NoBlock()
	tty.SetTimeoutNoSave(tty.timeout)

	// Read bytes from the terminal
	numRead, err := tty.readBytes(bytes)
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
	case numRead == 5:
		seq := [5]byte{bytes[0], bytes[1], bytes[2], bytes[3], bytes[4]}
		if code, found := fKeyLookup[seq]; found {
			keyCode = code
			tty.Restore()
			tty.Flush()
			return
		}
	case numRead == 6:
		seq := [6]byte{bytes[0], bytes[1], bytes[2], bytes[3], bytes[4], bytes[5]}
		if code, found := modKeyLookup[seq]; found {
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
		tty.lastKey = 0
		return 0
	}
	var key int
	if keyCode != 0 {
		key = keyCode
	} else {
		key = ascii
	}
	if key == tty.lastKey {
		tty.lastKey = 0
		return 0
	}
	tty.lastKey = key
	return key
}

// parseFirstKey parses the first key sequence from buf and returns its string
// representation plus the number of bytes consumed. When the buffer starts
// with an incomplete sequence (e.g. only ESC), consumed == 0 signals the
// caller to try reading more bytes before classifying. A return of
// (key, consumed) with consumed > 0 means a complete key has been recognised.
func parseFirstKey(buf []byte) (string, int) {
	n := len(buf)
	if n == 0 {
		return "", 0
	}
	// Non-ESC single byte: plain character or control code.
	if buf[0] != 27 {
		r, size := utf8.DecodeRune(buf)
		if r == utf8.RuneError && size <= 1 {
			return "c:" + strconv.Itoa(int(buf[0])), 1
		}
		if unicode.IsPrint(r) {
			return string(r), size
		}
		return "c:" + strconv.Itoa(int(buf[0])), 1
	}
	// ESC alone: need more bytes to decide (might be start of CSI/SS3).
	if n < 2 {
		return "", 0
	}
	// Lone ESC followed by something that's not [ or O: it's the Escape key
	// (or Alt+key) — for orbiton's purposes return it as c:27 and keep the
	// next byte for the following call.
	if buf[1] != '[' && buf[1] != 'O' {
		// Alt-Return is reported as ESC + CR (or ESC + LF) on most terminals.
		// When both bytes have already arrived in the same buffer the user
		// pressed them together — a real Escape would have been consumed
		// before the next key arrived — so treat the pair as a single key.
		if buf[1] == 0x0D || buf[1] == 0x0A {
			return "alt⏎", 2
		}
		return "c:27", 1
	}
	// 3-byte sequences: ESC [ X   or   ESC O X
	if n >= 3 {
		seq3 := [3]byte{buf[0], buf[1], buf[2]}
		if str, ok := keyStringLookup[seq3]; ok {
			return str, 3
		}
	}
	// 4-byte sequences: ESC [ N ~
	if n >= 4 {
		seq4 := [4]byte{buf[0], buf[1], buf[2], buf[3]}
		if str, ok := pageStringLookup[seq4]; ok {
			return str, 4
		}
	}
	// 5-byte sequences: ESC [ N N ~
	if n >= 5 {
		seq5 := [5]byte{buf[0], buf[1], buf[2], buf[3], buf[4]}
		if str, ok := fKeyStringLookup[seq5]; ok {
			return str, 5
		}
	}
	// 6-byte modifier sequences: ESC [ 1 ; M X
	if n >= 6 {
		seq6 := [6]byte{buf[0], buf[1], buf[2], buf[3], buf[4], buf[5]}
		if str, ok := modKeyStringLookup[seq6]; ok {
			return str, 6
		}
	}
	// Unknown CSI sequence. Consume up to the terminator so stray bytes don't
	// get re-emitted as literal "^[[..." text. A CSI/SS3 final byte is in the
	// range 0x40-0x7E (or '~' for page-type sequences).
	if buf[1] == '[' || buf[1] == 'O' {
		for i := 2; i < n; i++ {
			b := buf[i]
			if (b >= 0x40 && b <= 0x7E) || b == '~' {
				seq := string(buf[:i+1])
				// Recognise long CSI sequences (kitty CSI-u, xterm
				// modifyOtherKeys=2) that report modified keys not
				// covered by the fixed-size lookups above.
				if str, ok := longCSILookup[seq]; ok {
					return str, i + 1
				}
				return seq, i + 1
			}
		}
		// Terminator not yet in buffer — wait for more bytes.
		return "", 0
	}
	// Fallback: consume one byte.
	return string(buf[:1]), 1
}

// ReadKey reads a key sequence (or printable character) from the TTY.
// When multiple key sequences arrive in one read (for example a held-down
// arrow key during a slow redraw), they are returned one by one on
// successive calls via a pending byte buffer — this prevents queued arrow
// escapes from leaking into the document as literal "^[[..." text.
func (tty *TTY) ReadKey() string {
	// Note: we deliberately do NOT restore the original terminal state or
	// flush the input queue on exit. Restoring would re-enable echo between
	// keystrokes (causing raw escape sequences like "\x1b[A" to be echoed
	// onto the screen while the editor is busy redrawing — visible as
	// literal "^[[A" and, in graphical book mode, as flicker/jumping).
	// Flushing would discard keystrokes the user typed while a redraw was
	// in progress. The outer editor loop restores the terminal on exit.
	tty.RawMode()

	// Try to return a key already sitting in the pending buffer first.
	if key, consumed := parseFirstKey(tty.pending); consumed > 0 {
		tty.pending = tty.pending[consumed:]
		return key
	}

	// Need more bytes. Use a generous read buffer so bursts of queued input
	// (e.g. every \x1b[C from a held Right-arrow) are not split across reads.
	// Block until at least one byte arrives.
	savedTimeout, err := tty.SetTimeout(0)
	if err != nil {
		return ""
	}
	defer tty.SetTimeout(savedTimeout)

	readBuf := make([]byte, 256)
	numRead, err := tty.readBytes(readBuf)
	if numRead < 0 {
		numRead = 0
	}
	if err != nil && numRead == 0 {
		return ""
	}
	tty.pending = append(tty.pending, readBuf[:numRead]...)

	// If the pending buffer currently holds only an incomplete escape
	// sequence (e.g. lone ESC or ESC [ without a terminator), do one short
	// follow-up read to let the rest arrive before classifying.
	if key, consumed := parseFirstKey(tty.pending); consumed > 0 {
		tty.pending = tty.pending[consumed:]
		return key
	}
	// Incomplete: wait briefly for the tail of the escape sequence.
	tty.SetTimeoutNoSave(defaultTimeout)
	numRead2, _ := tty.readBytes(readBuf)
	if numRead2 > 0 {
		tty.pending = append(tty.pending, readBuf[:numRead2]...)
	}
	if key, consumed := parseFirstKey(tty.pending); consumed > 0 {
		tty.pending = tty.pending[consumed:]
		return key
	}
	// Still nothing parseable (shouldn't normally happen); flush the pending
	// bytes as-is so we don't deadlock on them. A lone ESC byte that never
	// got a continuation is the Escape key itself — return it as "c:27" so
	// callers that compare against the canonical key string (e.g. menu
	// dismissal) continue to work.
	if len(tty.pending) == 1 && tty.pending[0] == 27 {
		tty.pending = tty.pending[:0]
		return "c:27"
	}
	s := string(tty.pending)
	tty.pending = tty.pending[:0]
	return s
}

// Rune reads a rune, handling special sequences for arrows, Home, End, etc.
func (tty *TTY) Rune() rune {
	bytes := make([]byte, 6)
	tty.RawMode()

	savedTimeout, err := tty.SetTimeout(0) // block until data
	if err != nil {
		return rune(0)
	}
	defer tty.SetTimeout(savedTimeout)

	numRead, err := tty.readBytes(bytes)
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
	case numRead == 5:
		seq := [5]byte{bytes[0], bytes[1], bytes[2], bytes[3], bytes[4]}
		if str, found := fKeyStringLookup[seq]; found {
			return []rune(str)[0]
		}
		r, _ := utf8.DecodeRune(bytes[:numRead])
		return r
	case numRead == 6:
		seq := [6]byte{bytes[0], bytes[1], bytes[2], bytes[3], bytes[4], bytes[5]}
		if str, found := modKeyStringLookup[seq]; found {
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
	if tty.reader != nil {
		return
	}
	a, err := tcgetattr(tty.fd)
	if err != nil {
		return
	}
	cfmakeraw(&a)
	tcsetattr(tty.fd, &a)
}

// NoBlock sets the terminal to cbreak mode
func (tty *TTY) NoBlock() {
	if tty.reader != nil {
		return
	}
	a, err := tcgetattr(tty.fd)
	if err != nil {
		return
	}
	cfmakecbreak(&a)
	tcsetattr(tty.fd, &a)
}

// Restore the terminal to its original state (flushes pending input)
func (tty *TTY) Restore() {
	if tty.reader != nil {
		return
	}
	unix.IoctlSetTermios(tty.fd, ioctlFLUSHSET, &tty.orig)
}

// RestoreNoFlush restores the terminal to its original state without flushing pending input
func (tty *TTY) RestoreNoFlush() {
	if tty.reader != nil {
		return
	}
	unix.IoctlSetTermios(tty.fd, ioctlSETATTR, &tty.orig)
}

// Flush discards pending input/output
func (tty *TTY) Flush() {
	if tty.reader != nil {
		return
	}
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

	// Set a long read timeout
	d := 100 * time.Millisecond
	_, err := tty.SetTimeout(d) // block until data
	if err != nil {
		return "", err
	}

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

// ReadStringKeepTiming reads all available data from the TTY,
// preserving the caller's timeout value
func (tty *TTY) ReadStringKeepTiming() (string, error) {
	var result []byte
	buf := make([]byte, 128)

	// Set a long read timeout
	d := 100 * time.Millisecond
	savedTimeout, err := tty.SetTimeout(d) // block until data
	if err != nil {
		return "", err
	}
	defer tty.SetTimeout(savedTimeout)

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

	// block until data
	if savedTimeout, err := tty.SetTimeout(0); err == nil { // success
		defer tty.SetTimeout(savedTimeout)
	}

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
