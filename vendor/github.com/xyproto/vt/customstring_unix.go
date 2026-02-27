//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package vt

import (
	"strconv"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/sys/unix"
)

// KeyString reads a keypress and returns it as a string, without flushing pending input
func (tty *TTY) KeyString() string {
	buf := make([]byte, 1)
	savedTimeout := tty.timeout

	tty.RawMode()

	// Block until first byte arrives
	tty.SetTimeout(0)
	n, err := unix.Read(tty.fd, buf)
	if n < 0 {
		n = 0
	}

	// Restore the timeout without flushing pending input
	defer tty.SetTimeout(savedTimeout)

	if err != nil || n == 0 {
		return ""
	}

	b := buf[0]
	escTimeout := 50 * time.Millisecond

	// Non-ESC: single ASCII byte or multi-byte UTF-8
	if b != 27 {
		// Multi-byte UTF-8 sequence
		if b >= 0xC0 {
			var expected int
			switch {
			case b < 0xE0:
				expected = 2
			case b < 0xF0:
				expected = 3
			default:
				expected = 4
			}
			utfBuf := make([]byte, expected)
			utfBuf[0] = b
			one := make([]byte, 1)
			for i := 1; i < expected; i++ {
				tty.SetTimeout(escTimeout)
				numRead, err := unix.Read(tty.fd, one)
				if numRead < 0 {
					numRead = 0
				}
				if err != nil || numRead == 0 {
					break
				}
				utfBuf[i] = one[0]
			}
			r, _ := utf8.DecodeRune(utfBuf)
			if r != utf8.RuneError && unicode.IsPrint(r) {
				return string(r)
			}
			return "c:" + strconv.Itoa(int(b))
		}
		// Single ASCII byte
		r := rune(b)
		if unicode.IsPrint(r) {
			return string(r)
		}
		return "c:" + strconv.Itoa(int(b))
	}

	// ESC byte: collect the rest of the escape sequence one byte at a time
	seq := []byte{b}

	one := make([]byte, 1)
	tty.SetTimeout(escTimeout)
	n, err = unix.Read(tty.fd, one)
	if n < 0 {
		n = 0
	}
	if err != nil || n == 0 {
		return "c:27" // bare ESC
	}
	seq = append(seq, one[0])

	// CSI sequence: ESC [ ... finalByte (0x40-0x7E)
	if one[0] == '[' {
		for {
			tty.SetTimeout(escTimeout)
			n, err = unix.Read(tty.fd, one)
			if n < 0 {
				n = 0
			}
			if err != nil || n == 0 {
				break
			}
			seq = append(seq, one[0])
			if one[0] >= 0x40 && one[0] <= 0x7E {
				break
			}
		}
	}

	// Match against the known lookup tables
	switch len(seq) {
	case 3:
		var key [3]byte
		copy(key[:], seq)
		if s, found := keyStringLookup[key]; found {
			return s
		}
	case 4:
		var key [4]byte
		copy(key[:], seq)
		if s, found := pageStringLookup[key]; found {
			return s
		}
	case 6:
		var key [6]byte
		copy(key[:], seq)
		if s, found := ctrlInsertStringLookup[key]; found {
			return s
		}
	}

	return string(seq)
}
