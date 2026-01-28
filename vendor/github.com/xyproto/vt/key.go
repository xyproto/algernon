//go:build !windows

package vt

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/pkg/term"
)

var (
	defaultTimeout    = 2 * time.Millisecond
	defaultESCTimeout = 100 * time.Millisecond
	lastKey           int
)

// Key codes for CSI (ESC [) sequences and SS3 (ESC O) sequences.
var csiKeyLookup = map[string]int{
	"[A":    253, // Up Arrow
	"[B":    255, // Down Arrow
	"[C":    254, // Right Arrow
	"[D":    252, // Left Arrow
	"[H":    1,   // Home
	"[F":    5,   // End
	"[1~":   1,   // Home
	"[4~":   5,   // End
	"[5~":   251, // Page Up
	"[6~":   250, // Page Down
	"[2;5~": 258, // Ctrl-Insert
}

var ss3KeyLookup = map[byte]int{
	'A': 253, // Up Arrow
	'B': 255, // Down Arrow
	'C': 254, // Right Arrow
	'D': 252, // Left Arrow
	'H': 1,   // Home
	'F': 5,   // End
}

var keyCodeToString = map[int]string{
	253: "↑",
	255: "↓",
	254: "→",
	252: "←",
	1:   "⇱",
	5:   "⇲",
	251: "⇞",
	250: "⇟",
	258: "⎘",
}

const (
	esc                   = 0x1b
	bracketedPasteStart   = "\x1b[200~"
	bracketedPasteEnd     = "\x1b[201~"
	enableBracketedPaste  = "\x1b[?2004h"
	disableBracketedPaste = "\x1b[?2004l"
)

type EventKind int

const (
	EventNone EventKind = iota
	EventKey
	EventRune
	EventText
	EventPaste
)

type Event struct {
	Kind EventKind
	Key  int
	Rune rune
	Text string
}

type inputReader struct {
	buf         []byte
	escDeadline time.Time
	escSeqLen   int
	inPaste     bool
}

type TTY struct {
	t          *term.Term
	timeout    time.Duration
	escTimeout time.Duration
	reader     *inputReader
	noBlock    bool
}

// NewTTY opens /dev/tty in raw and cbreak mode as a term.Term
func NewTTY() (*TTY, error) {
	// Apply raw mode last to avoid cbreak overriding raw settings.
	t, err := term.Open("/dev/tty", term.CBreakMode, term.RawMode, term.ReadTimeout(defaultTimeout))
	if err != nil {
		return nil, err
	}
	tty := &TTY{
		t:          t,
		timeout:    defaultTimeout,
		escTimeout: defaultESCTimeout,
		reader:     &inputReader{},
	}
	// Best-effort enable bracketed paste for terminals that support it.
	_, _ = tty.t.Write([]byte(enableBracketedPaste))
	return tty, nil
}

// SetTimeout sets a timeout for reading a key
func (tty *TTY) SetTimeout(d time.Duration) {
	tty.timeout = d
	tty.t.SetReadTimeout(tty.timeout)
}

// SetEscTimeout sets the timeout used to decide if ESC is a standalone key.
func (tty *TTY) SetEscTimeout(d time.Duration) {
	tty.escTimeout = d
}

// Close will restore and close the raw terminal
func (tty *TTY) Close() {
	// Best-effort disable bracketed paste before restoring the terminal.
	_, _ = tty.t.Write([]byte(disableBracketedPaste))
	tty.t.Restore()
	tty.t.Close()
}

// ReadEvent reads and parses a single input event (key, rune, or paste).
// It is designed to feel non-blocking while still assembling escape sequences.
func (tty *TTY) ReadEvent() (Event, error) {
	return tty.readEvent(tty.timeout, tty.escTimeout)
}

// ReadEventBlocking waits until a full input event is available.
func (tty *TTY) ReadEventBlocking() (Event, error) {
	for {
		ev, err := tty.readEvent(0, tty.escTimeout)
		if err != nil {
			return ev, err
		}
		if ev.Kind != EventNone {
			return ev, nil
		}
	}
}

func (tty *TTY) readEvent(poll, escWait time.Duration) (Event, error) {
	for {
		// Try to parse what's already in the buffer.
		ev, ready, needMore := tty.reader.parse(time.Now(), escWait)
		if ready {
			return ev, nil
		} else if !needMore && len(tty.reader.buf) == 0 {
			// No buffered input; read from terminal.
			if poll > 0 {
				// Wait briefly in raw mode to avoid key echoing in cooked mode.
				deadline := time.Now().Add(poll)
				for {
					avail, err := tty.t.Available()
					if err == nil && avail > 0 {
						break
					}
					if time.Now().After(deadline) {
						return Event{Kind: EventNone}, nil
					}
					time.Sleep(1 * time.Millisecond)
				}
				// Read immediately without termios timeout quantization.
				if err := tty.readIntoBuffer(0); err != nil {
					return Event{Kind: EventNone}, err
				}
			} else {
				readTimeout := poll
				if poll == 0 && len(tty.reader.buf) > 0 {
					readTimeout = tty.timeout
				}
				if err := tty.readIntoBuffer(readTimeout); err != nil {
					return Event{Kind: EventNone}, err
				}
			}
			if len(tty.reader.buf) == 0 {
				return Event{Kind: EventNone}, nil
			}
			continue
		}

		// Kilo-style: after ESC, wait a little for the rest of the sequence.
		if needMore && len(tty.reader.buf) > 0 && tty.reader.buf[0] == esc {
			if err := tty.readIntoBuffer(escWait); err != nil {
				return Event{Kind: EventNone}, err
			}
			continue
		}

		readTimeout := poll
		if poll == 0 && len(tty.reader.buf) > 0 {
			readTimeout = tty.timeout
		}
		if err := tty.readIntoBuffer(readTimeout); err != nil {
			return Event{Kind: EventNone}, err
		}
	}
}

func (tty *TTY) readIntoBuffer(timeout time.Duration) error {
	_ = tty.t.SetReadTimeout(timeout)
	tmp := make([]byte, 256)
	n, err := tty.t.Read(tmp)
	if n > 0 {
		tty.reader.buf = append(tty.reader.buf, tmp[:n]...)
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	for {
		avail, err := tty.t.Available()
		if err != nil || avail <= 0 {
			break
		}
		if avail > len(tmp) {
			if avail > 4096 {
				avail = 4096
			}
			tmp = make([]byte, avail)
		}
		n, err = tty.t.Read(tmp[:avail])
		if n > 0 {
			tty.reader.buf = append(tty.reader.buf, tmp[:n]...)
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		if n == 0 {
			break
		}
	}
	return nil
}

func (r *inputReader) parse(now time.Time, escWait time.Duration) (Event, bool, bool) {
	if len(r.buf) == 0 {
		r.escDeadline = time.Time{}
		r.escSeqLen = 0
		return Event{Kind: EventNone}, false, false
	}

	if r.inPaste {
		if idx := bytes.Index(r.buf, []byte(bracketedPasteEnd)); idx >= 0 {
			text := string(r.buf[:idx])
			r.buf = r.buf[idx+len(bracketedPasteEnd):]
			r.inPaste = false
			r.escSeqLen = 0
			return Event{Kind: EventPaste, Text: text}, true, false
		}
		return Event{Kind: EventNone}, false, true
	}

	if r.buf[0] == esc {
		if bytes.HasPrefix(r.buf, []byte(bracketedPasteStart)) {
			r.buf = r.buf[len(bracketedPasteStart):]
			r.inPaste = true
			if idx := bytes.Index(r.buf, []byte(bracketedPasteEnd)); idx >= 0 {
				text := string(r.buf[:idx])
				r.buf = r.buf[idx+len(bracketedPasteEnd):]
				r.inPaste = false
				r.escSeqLen = 0
				return Event{Kind: EventPaste, Text: text}, true, false
			}
			return Event{Kind: EventNone}, false, true
		}

		if ev, consumed, ok, complete := parseCSI(r.buf); ok {
			r.buf = r.buf[consumed:]
			r.escDeadline = time.Time{}
			r.escSeqLen = 0
			return ev, true, false
		} else if complete && consumed > 0 {
			seq := string(r.buf[:consumed])
			r.buf = r.buf[consumed:]
			r.escDeadline = time.Time{}
			r.escSeqLen = 0
			return Event{Kind: EventText, Text: seq}, true, false
		}

		if ev, consumed, ok, complete := parseSS3(r.buf); ok {
			r.buf = r.buf[consumed:]
			r.escDeadline = time.Time{}
			r.escSeqLen = 0
			return ev, true, false
		} else if complete && consumed > 0 {
			seq := string(r.buf[:consumed])
			r.buf = r.buf[consumed:]
			r.escDeadline = time.Time{}
			r.escSeqLen = 0
			return Event{Kind: EventText, Text: seq}, true, false
		}

		if r.escDeadline.IsZero() || len(r.buf) > r.escSeqLen {
			r.escDeadline = now.Add(escWait)
			r.escSeqLen = len(r.buf)
		}
		if now.Before(r.escDeadline) {
			return Event{Kind: EventNone}, false, true
		}

		r.buf = r.buf[1:]
		r.escDeadline = time.Time{}
		r.escSeqLen = 0
		return Event{Kind: EventKey, Key: int(esc)}, true, false
	}

	r.escDeadline = time.Time{}
	r.escSeqLen = 0
	r0, size := utf8.DecodeRune(r.buf)
	if r0 == utf8.RuneError && size == 1 {
		return Event{Kind: EventNone}, false, true
	}
	r.buf = r.buf[size:]
	return Event{Kind: EventRune, Rune: r0}, true, false
}

func parseCSI(buf []byte) (Event, int, bool, bool) {
	if len(buf) < 2 || buf[0] != esc || buf[1] != '[' {
		return Event{}, 0, false, false
	}
	for i := 2; i < len(buf); i++ {
		b := buf[i]
		if b >= 0x40 && b <= 0x7e {
			if code, ok := csiKeyLookup[string(buf[1:i+1])]; ok {
				return Event{Kind: EventKey, Key: code}, i + 1, true, true
			}
			if ev, ok := parseCSIFallback(buf[2:i], b); ok {
				return ev, i + 1, true, true
			}
			return Event{}, i + 1, false, true
		}
	}
	return Event{}, 0, false, false
}

func parseSS3(buf []byte) (Event, int, bool, bool) {
	if len(buf) < 2 || buf[0] != esc || buf[1] != 'O' {
		return Event{}, 0, false, false
	}
	if len(buf) < 3 {
		return Event{}, 0, false, false
	}
	if code, ok := ss3KeyLookup[buf[2]]; ok {
		return Event{Kind: EventKey, Key: code}, 3, true, true
	}
	return Event{}, 3, false, true
}

// Key reads the keycode or ASCII code and avoids repeated keys
func (tty *TTY) Key() int {
	if !tty.noBlock {
		tty.RawMode()
	}
	ev, err := tty.ReadEvent()
	if !tty.noBlock {
		tty.Restore()
	}
	if ev.Kind != EventNone {
		tty.t.Flush()
	}
	if err != nil {
		lastKey = 0
		return 0
	}
	var key int
	switch ev.Kind {
	case EventKey:
		key = ev.Key
	case EventRune:
		key = int(ev.Rune)
	default:
		key = 0
	}
	if key == lastKey {
		lastKey = 0
		return 0
	}
	lastKey = key
	return key
}

// KeyRaw reads a key without toggling raw mode or flushing input.
// Callers should manage tty.RawMode() / tty.Restore() themselves.
func (tty *TTY) KeyRaw() int {
	// Ensure raw mode is active to avoid echoing escape sequences.
	tty.RawMode()
	ev, err := tty.ReadEvent()
	if err != nil {
		return 0
	}
	var key int
	switch ev.Kind {
	case EventKey:
		key = ev.Key
	case EventRune:
		key = int(ev.Rune)
	default:
		key = 0
	}
	return key
}

// String reads a string, handling key sequences and printable characters
func (tty *TTY) String() string {
	tty.RawMode()
	ev, err := tty.ReadEventBlocking()
	tty.Restore()
	if ev.Kind != EventNone {
		tty.t.Flush()
	}
	if err != nil {
		return ""
	}
	switch ev.Kind {
	case EventPaste:
		return ev.Text
	case EventText:
		return ev.Text
	case EventKey:
		if s, ok := keyCodeToString[ev.Key]; ok {
			return s
		}
		return "c:" + strconv.Itoa(ev.Key)
	case EventRune:
		if unicode.IsPrint(ev.Rune) {
			return string(ev.Rune)
		}
		return "c:" + strconv.Itoa(int(ev.Rune))
	default:
		return ""
	}
}

// StringRaw reads a string without toggling raw mode or flushing input.
// Callers should manage tty.RawMode() / tty.Restore() themselves.
func (tty *TTY) StringRaw() string {
	// Ensure raw mode is active to avoid echoing escape sequences.
	tty.RawMode()
	ev, err := tty.ReadEventBlocking()
	if err != nil {
		return ""
	}
	switch ev.Kind {
	case EventPaste:
		return ev.Text
	case EventText:
		return ev.Text
	case EventKey:
		if s, ok := keyCodeToString[ev.Key]; ok {
			return s
		}
		return "c:" + strconv.Itoa(ev.Key)
	case EventRune:
		if unicode.IsPrint(ev.Rune) {
			return string(ev.Rune)
		}
		return "c:" + strconv.Itoa(int(ev.Rune))
	default:
		return ""
	}
}

// Rune reads a rune, handling special sequences for arrows, Home, End, etc.
func (tty *TTY) Rune() rune {
	tty.RawMode()
	ev, err := tty.ReadEventBlocking()
	tty.Restore()
	if ev.Kind != EventNone {
		tty.t.Flush()
	}
	if err != nil {
		return rune(0)
	}
	switch ev.Kind {
	case EventRune:
		return ev.Rune
	case EventKey:
		if s, ok := keyCodeToString[ev.Key]; ok {
			return []rune(s)[0]
		}
	case EventText:
		if ev.Text != "" {
			return []rune(ev.Text)[0]
		}
	case EventPaste:
		if ev.Text != "" {
			return []rune(ev.Text)[0]
		}
	}
	return rune(0)
}

// RuneRaw reads a rune without toggling raw mode or flushing input.
// Callers should manage tty.RawMode() / tty.Restore() themselves.
func (tty *TTY) RuneRaw() rune {
	// Ensure raw mode is active to avoid echoing escape sequences.
	tty.RawMode()
	ev, err := tty.ReadEventBlocking()
	if err != nil {
		return rune(0)
	}
	switch ev.Kind {
	case EventRune:
		return ev.Rune
	case EventKey:
		if s, ok := keyCodeToString[ev.Key]; ok {
			return []rune(s)[0]
		}
	case EventText:
		if ev.Text != "" {
			return []rune(ev.Text)[0]
		}
	case EventPaste:
		if ev.Text != "" {
			return []rune(ev.Text)[0]
		}
	}
	return rune(0)
}

// RawMode switches the terminal to raw mode
func (tty *TTY) RawMode() {
	tty.t.SetRaw()
}

// NoBlock prevents Key() from toggling terminal modes.
// Use this in game loops to prevent escape sequence characters from being echoed.
func (tty *TTY) NoBlock() {
	tty.noBlock = true
	tty.RawMode()
}

// Restore the terminal to its original state
func (tty *TTY) Restore() {
	tty.t.Restore()
}

// Flush flushes the terminal output
func (tty *TTY) Flush() {
	tty.t.Flush()
}

// WriteString writes a string to the terminal
func (tty *TTY) WriteString(s string) error {
	if n, err := tty.t.Write([]byte(s)); err != nil || n == 0 {
		return errors.New("no bytes written to the TTY")
	}
	return nil
}

// ReadString reads a string from the TTY with timeout
func (tty *TTY) ReadString() (string, error) {
	// Set up a timeout channel
	timeout := time.After(100 * time.Millisecond)
	resultChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	go func() {
		// Set raw mode temporarily
		tty.RawMode()
		defer tty.Restore()
		defer tty.Flush()

		var result []byte
		buffer := make([]byte, 1)

		for {
			n, err := tty.t.Read(buffer)
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

	// Set the terminal into raw mode with a timeout
	tty.RawMode()
	tty.SetTimeout(0)
	// Read bytes from the terminal
	numRead, err := tty.t.Read(bytes)
	// Restore the terminal settings
	tty.Restore()
	tty.t.Flush()

	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Raw bytes: %v\n", bytes[:numRead])
}

// Term will return the underlying term.Term
func (tty *TTY) Term() *term.Term {
	return tty.t
}

// ASCII returns the ASCII code of the key pressed
func (tty *TTY) ASCII() int {
	tty.RawMode()
	defer func() {
		tty.Restore()
		tty.t.Flush()
	}()
	ev, err := tty.ReadEvent()
	if err != nil {
		return 0
	}
	if ev.Kind == EventRune {
		return int(ev.Rune)
	}
	return 0
}

// KeyCode returns the key code of the key pressed
func (tty *TTY) KeyCode() int {
	tty.RawMode()
	defer func() {
		tty.Restore()
		tty.t.Flush()
	}()
	ev, err := tty.ReadEvent()
	if err != nil {
		return 0
	}
	if ev.Kind == EventKey {
		return ev.Key
	}
	return 0
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
