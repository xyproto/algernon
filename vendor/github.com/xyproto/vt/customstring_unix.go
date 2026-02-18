//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package vt

import (
	"strconv"
	"time"
	"unicode"

	"golang.org/x/sys/unix"
)

// CustomString reads a key string like String(), but preserves any pending
// input in the kernel's tty buffer. This is critical for operations like
// shift-insert paste, where the key escape sequence is immediately followed
// by paste data — flushing the buffer after reading the key would lose it.
//
// Differences from String():
//   - Saves and restores timeout via SetTimeout instead of Restore()+Flush(),
//     so the terminal stays in raw mode and pending input is preserved.
//   - When the first byte is ESC, reads the escape sequence byte-by-byte
//     to avoid consuming any trailing paste data.
func (tty *TTY) CustomString() string {
	buf := make([]byte, 1)

	// Save timeout before SetTimeout(0) overwrites it
	savedTimeout := tty.timeout

	tty.RawMode()

	// Blocking read for first byte (VMIN=1, VTIME=0)
	tty.SetTimeout(0)
	n, err := unix.Read(tty.fd, buf)
	if n < 0 {
		n = 0
	}

	// Restore the saved timeout without flushing pending input
	defer tty.SetTimeout(savedTimeout)

	if err != nil || n == 0 {
		return ""
	}

	b := buf[0]

	// Non-ESC single byte: return immediately
	if b != 27 {
		r := rune(b)
		if unicode.IsPrint(r) {
			return string(r)
		}
		return "c:" + strconv.Itoa(int(b))
	}

	// ESC byte received — collect the rest of the escape sequence
	// one byte at a time so we stop exactly at the sequence boundary.
	var seq []byte
	seq = append(seq, b)

	escTimeout := 50 * time.Millisecond

	// Read the next byte with a short timeout; if nothing follows ESC, it's a bare ESC
	one := make([]byte, 1)
	tty.SetTimeout(escTimeout)
	n, err = unix.Read(tty.fd, one)
	if n < 0 {
		n = 0
	}
	if err != nil || n == 0 {
		return "c:27"
	}
	seq = append(seq, one[0])

	// If second byte is '[', this is a CSI sequence (ESC [ ... finalByte)
	if one[0] == '[' {
		// Read parameter/intermediate bytes until a final byte (0x40-0x7E) arrives
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
			// Final byte of a CSI sequence is in the range 0x40–0x7E (@ through ~)
			if one[0] >= 0x40 && one[0] <= 0x7E {
				break
			}
		}
	}

	// Now match against the known lookup tables
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
