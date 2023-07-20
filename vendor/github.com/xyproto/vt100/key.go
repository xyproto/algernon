package vt100

import (
	"errors"
	"io/ioutil"
	"strconv"
	"time"
	"unicode"

	"github.com/pkg/term"
)

var (
	defaultTimeout = 2 * time.Millisecond
	lastKey        int
)

type TTY struct {
	t       *term.Term
	timeout time.Duration
}

// NewTTY opens /dev/tty in raw and cbreak mode as a term.Term
func NewTTY() (*TTY, error) {
	t, err := term.Open("/dev/tty", term.RawMode, term.CBreakMode, term.ReadTimeout(defaultTimeout))
	if err != nil {
		return nil, err
	}
	return &TTY{t, defaultTimeout}, nil
}

// Term will return the underlying term.Term
func (tty *TTY) Term() *term.Term {
	return tty.t
}

// RawMode will switch the terminal to raw mode
func (tty *TTY) RawMode() {
	term.RawMode(tty.t)
}

// NoBlock leaves "cooked" mode and enters "cbreak" mode
func (tty *TTY) NoBlock() {
	tty.t.SetCbreak()
}

// SetTimeout sets a timeout for reading a key
func (tty *TTY) SetTimeout(d time.Duration) {
	tty.timeout = d
	tty.t.SetReadTimeout(tty.timeout)
}

// Restore will restore the terminal
func (tty *TTY) Restore() {
	tty.t.Restore()
}

// Close will Restore and close the raw terminal
func (tty *TTY) Close() {
	t := tty.Term()
	t.Restore()
	t.Close()
}

// Thanks https://stackoverflow.com/a/32018700/131264
// Returns either an ascii code, or (if input is an arrow) a Javascript key code.
func asciiAndKeyCode(tty *TTY) (ascii, keyCode int, err error) {
	bytes := make([]byte, 3)
	var numRead int
	tty.RawMode()
	tty.NoBlock()
	tty.SetTimeout(tty.timeout)
	numRead, err = tty.t.Read(bytes)
	tty.Restore()
	tty.t.Flush()
	if err != nil {
		return
	}
	if numRead == 3 && bytes[0] == 27 && bytes[1] == 91 {
		// Three-character control sequence, beginning with "ESC-[".

		// Since there are no ASCII codes for arrow keys, we use
		// Javascript key codes.
		if bytes[2] == 65 {
			// Up
			keyCode = 38
		} else if bytes[2] == 66 {
			// Down
			keyCode = 40
		} else if bytes[2] == 67 {
			// Right
			keyCode = 39
		} else if bytes[2] == 68 {
			// Left
			keyCode = 37
		}
	} else if numRead == 1 {
		ascii = int(bytes[0])
		//} else {
		// TWo characters read??
	}
	return
}

// Don't use the "JavaScript key codes" for the arrow keys
func asciiAndKeyCodeNoJavascript(tty *TTY) (ascii, keyCode int, err error) {
	bytes := make([]byte, 3)
	var numRead int
	tty.RawMode()
	tty.NoBlock()
	tty.SetTimeout(tty.timeout)
	numRead, err = tty.t.Read(bytes)
	tty.Restore()
	tty.t.Flush()
	if err != nil {
		return
	}
	if numRead == 3 && bytes[0] == 27 && bytes[1] == 91 {
		// Three-character control sequence, beginning with "ESC-[".

		// Since there are no ASCII codes for arrow keys, we use
		// the last 4 values of a byte
		if bytes[2] == 65 {
			// Up
			keyCode = 253
		} else if bytes[2] == 66 {
			// Down
			keyCode = 255
		} else if bytes[2] == 67 {
			// Right
			keyCode = 254
		} else if bytes[2] == 68 {
			// Left
			keyCode = 252
		}
	} else if numRead == 1 {
		ascii = int(bytes[0])
		//} else {
		// Two characters read??
	}
	return
}

// Returns either an ascii code, or (if input is an arrow) a Javascript key code.
func asciiAndKeyCodeOnce() (ascii, keyCode int, err error) {
	t, err := NewTTY()
	if err != nil {
		return 0, 0, err
	}
	a, kc, err := asciiAndKeyCode(t)
	t.Close()
	return a, kc, err
}

func (tty *TTY) ASCII() int {
	ascii, _, err := asciiAndKeyCode(tty)
	if err != nil {
		return 0
	}
	return ascii
}

func ASCIIOnce() int {
	ascii, _, err := asciiAndKeyCodeOnce()
	if err != nil {
		return 0
	}
	return ascii
}

func (tty *TTY) KeyCode() int {
	_, keyCode, err := asciiAndKeyCode(tty)
	if err != nil {
		return 0
	}
	return keyCode
}

func KeyCodeOnce() int {
	_, keyCode, err := asciiAndKeyCodeOnce()
	if err != nil {
		return 0
	}
	return keyCode
}

// Return the keyCode or ascii, but ignore repeated keys
func (tty *TTY) Key() int {
	ascii, keyCode, err := asciiAndKeyCodeNoJavascript(tty)
	if err != nil {
		lastKey = 0
		return 0
	}
	if keyCode != 0 {
		if keyCode == lastKey {
			lastKey = 0
			return 0
		}
		lastKey = keyCode
		return keyCode
	}
	if ascii == lastKey {
		lastKey = 0
		return 0
	}
	lastKey = ascii
	return ascii
}

func KeyOnce() int {
	ascii, keyCode, err := asciiAndKeyCodeOnce()
	if err != nil {
		return 0
	}
	if keyCode != 0 {
		return keyCode
	}
	return ascii
}

// Wait for Return, Esc, Space or q to be pressed
func WaitForKey() {
	// Get a new TTY and start reading keypresses in a loop
	r, err := NewTTY()
	if err != nil {
		r.Close()
		panic(err)
	}
	//r.SetTimeout(10 * time.Millisecond)
	for {
		switch r.Key() {
		case 13, 27, 32, 113:
			r.Close()
			return
		}
	}
}

// String will block and then return a string
// Arrow keys are returned as ←, →, ↑ or ↓
// returns an empty string if the pressed key could not be interpreted
func (tty *TTY) String() string {
	bytes := make([]byte, 3)
	tty.RawMode()
	//tty.NoBlock()
	tty.SetTimeout(0)
	numRead, err := tty.t.Read(bytes)
	if err != nil {
		return ""
	}
	tty.Restore()
	tty.t.Flush()
	if numRead == 3 && bytes[0] == 27 && bytes[1] == 91 {
		// Three-character control sequence, beginning with "ESC-[".

		// Since there are no ASCII codes for arrow keys, we use
		// the last 4 values of a byte
		if bytes[2] == 65 {
			// Up
			return "↑"
		} else if bytes[2] == 66 {
			// Down
			return "↓"
		} else if bytes[2] == 67 {
			// Right
			return "→"
		} else if bytes[2] == 68 {
			// Left
			return "←"
		}
	} else if numRead == 1 {
		r := rune(bytes[0])
		if unicode.IsPrint(r) {
			return string(r)
		}
		return "c:" + strconv.Itoa(int(r))
	} else {
		// Two or more bytes, a unicode character (or mashing several keys)
		return string([]rune(string(bytes))[0])
	}
	return ""
}

// Rune will block and then return a rune.
// Arrow keys are returned as ←, →, ↑ or ↓
// returns a rune(0) if the pressed key could not be interpreted
func (tty *TTY) Rune() rune {
	bytes := make([]byte, 3)
	tty.RawMode()
	//tty.NoBlock()
	tty.SetTimeout(0)
	numRead, err := tty.t.Read(bytes)
	if err != nil {
		return rune(0)
	}
	tty.Restore()
	tty.t.Flush()
	if numRead == 3 && bytes[0] == 27 && bytes[1] == 91 {
		// Three-character control sequence, beginning with "ESC-[".

		// Since there are no ASCII codes for arrow keys, we use
		// the last 4 values of a byte
		if bytes[2] == 65 {
			// Up
			return '↑'
		} else if bytes[2] == 66 {
			// Down
			return '↓'
		} else if bytes[2] == 67 {
			// Right
			return '→'
		} else if bytes[2] == 68 {
			// Left
			return '←'
		}
	} else if numRead == 1 {
		return rune(bytes[0])
	} else {
		// Two or more bytes, a unicode character (or mashing several keys)
		return []rune(string(bytes))[0]
	}
	return rune(0)
}

// Write a string to the TTY
func (tty *TTY) WriteString(s string) error {
	n, err := tty.t.Write([]byte(s))
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("no bytes written to the TTY")
	}
	return nil
}

// Read a string from the TTY
func (tty *TTY) ReadString() (string, error) {
	b, err := ioutil.ReadAll(tty.t)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
