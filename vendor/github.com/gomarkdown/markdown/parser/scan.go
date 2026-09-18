package parser

import (
	"bytes"
	"unicode"
	"unicode/utf8"
)

// IsEmpty returns the length of a blank line at the start of data,
// or 0 if the line is not blank. It is okay to call on an empty buffer.
func IsEmpty(data []byte) int {
	if len(data) == 0 {
		return 0
	}

	var i int
	for i = 0; i < len(data) && data[i] != '\n'; i++ {
		if data[i] != ' ' && data[i] != '\t' {
			return 0
		}
	}
	i = skipCharN(data, i, '\n', 1)
	return i
}

// skipChar advances i as long as data[i] == c
func skipChar(data []byte, i int, c byte) int {
	for i < len(data) && data[i] == c {
		i++
	}
	return i
}

// like skipChar but only skips up to max characters
func skipCharN(data []byte, i int, c byte, max int) int {
	for i < len(data) && max > 0 && data[i] == c {
		i++
		max--
	}
	return i
}

// skipUntilChar advances i as long as data[i] != c
func skipUntilChar(data []byte, i int, c byte) int {
	for i < len(data) && data[i] != c {
		i++
	}
	return i
}

func skipAlnum(data []byte, i int) int {
	for i < len(data) && IsAlnum(data[i]) {
		i++
	}
	return i
}

func skipSpace(data []byte, i int) int {
	for i < len(data) && IsSpace(data[i]) {
		i++
	}
	return i
}

func skipHSpace(data []byte, i int) int {
	for i < len(data) && (data[i] == ' ' || data[i] == '\t') {
		i++
	}
	return i
}

func backChar(data []byte, i int, c byte) int {
	for i > 0 && data[i-1] == c {
		i--
	}
	return i
}

func backUntilChar(data []byte, i int, c byte) int {
	for i > 0 && data[i-1] != c {
		i--
	}
	return i
}

func prefixedBlock(data []byte, marker string) int {
	i := skipCharN(data, 0, ' ', 3)
	if !bytes.HasPrefix(data[i:], []byte(marker)) {
		return 0
	}
	i += len(marker)
	if i < len(data) && data[i] == ' ' {
		i++
	}
	return i
}

func terminatesPrefixedBlock(data []byte, begin, end int, prefix func([]byte) int) bool {
	return IsEmpty(data[begin:]) > 0 &&
		(end >= len(data) || prefix(data[end:]) == 0 && IsEmpty(data[end:]) == 0)
}

// IsPunctuation returns true if c is a punctuation symbol.
func IsPunctuation(c byte) bool {
	return bytes.IndexByte([]byte("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"), c) >= 0
}

func IsPunctuation2(d []byte) bool {
	if len(d) == 0 {
		return false
	}
	if IsPunctuation(d[0]) {
		return true
	}
	r, _ := utf8.DecodeRune(d)
	if r == utf8.RuneError {
		return false
	}
	return unicode.IsPunct(r)
}

// IsSpace returns true if c is a white-space character
func IsSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v'
}

// IsLetter returns true if c is ascii letter
func IsLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

// IsAlnum returns true if c is an ASCII digit or letter.
func IsAlnum(c byte) bool { return c >= '0' && c <= '9' || IsLetter(c) }

func NormalizeNewlines(d []byte) []byte {
	if bytes.IndexByte(d, '\r') < 0 {
		return d
	}
	res := make([]byte, len(d))
	copy(res, d)
	d = res
	wi := 0
	n := len(d)
	for i := 0; i < n; i++ {
		c := d[i]
		// 13 is CR
		if c != 13 {
			d[wi] = c
			wi++
			continue
		}
		// replace CR (mac / win) with LF (unix)
		d[wi] = 10
		wi++
		if i < n-1 && d[i+1] == 10 {
			// this was CRLF, so skip the LF
			i++
		}

	}
	return d[:wi]
}
