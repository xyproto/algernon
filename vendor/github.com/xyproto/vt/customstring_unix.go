//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package vt

import (
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// KeyString reads a keypress and returns it as a string, without flushing pending input
func (tty *TTY) KeyString() string {
	buf := make([]byte, 1)
	savedTimeout := tty.timeout

	tty.RawMode()

	// Block until first byte arrives
	tty.SetTimeout(0)
	n, err := tty.readBytes(buf)
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
				numRead, err := tty.readBytes(one)
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

	// ESC byte: collect the rest of the escape sequence one byte at a time,
	// until parseFirstKey recognises a complete key
	seq := []byte{b}
	one := make([]byte, 1)
	for {
		tty.SetTimeout(escTimeout)
		n, err = tty.readBytes(one)
		if err != nil || n <= 0 {
			break
		}
		seq = append(seq, one[0])
		if _, consumed := parseFirstKey(seq); consumed > 0 {
			break
		}
	}
	if len(seq) == 1 {
		return "c:27" // bare ESC
	}
	if key, consumed := parseFirstKey(seq); consumed == len(seq) {
		if rest, ok := strings.CutPrefix(key, altMarker); ok {
			key, _ = parseFirstKey([]byte(rest))
		}
		return key
	}
	return string(seq)
}
