package parser

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"
)

// sliceOffset reports where part starts in whole when part is a suffix backed
// by the same byte slice.
func sliceOffset(whole, part []byte) (int, bool) {
	if len(part) == 0 {
		return len(whole), len(whole) == 0
	}
	base := len(whole) - len(part)
	return base, base >= 0 && &whole[base] == &part[0]
}

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

// skipWhile advances i as long as keep(data[i]) holds.
func skipWhile(data []byte, i int, keep func(byte) bool) int {
	for i < len(data) && keep(data[i]) {
		i++
	}
	return i
}

// skipChar advances i as long as data[i] == c
func skipChar(data []byte, i int, c byte) int {
	return skipWhile(data, i, func(b byte) bool { return b == c })
}

// nextByteIs reports whether the byte after data[i] is c.
func nextByteIs(data []byte, i int, c byte) bool {
	return i+1 < len(data) && data[i+1] == c
}

// byteIn reports whether c is one of the bytes in set.
func byteIn(c byte, set string) bool {
	return strings.IndexByte(set, c) >= 0
}

// pairAt reports whether data[i] is a and the byte after it is b.
func pairAt(data []byte, i int, a, b byte) bool {
	return i >= 0 && i < len(data) && data[i] == a && nextByteIs(data, i, b)
}

// enclosedBy reports whether s starts with open and ends with a separate
// close.
func enclosedBy(s []byte, open, close byte) bool {
	return len(s) >= 2 && s[0] == open && s[len(s)-1] == close
}

// skipUntilAny advances i to the first byte at or after it that is in set,
// or to len(data). A set of more than one byte must be ASCII.
func skipUntilAny(data []byte, i int, set string) int {
	if i >= len(data) {
		return i
	}
	var j int
	if len(set) == 1 {
		j = bytes.IndexByte(data[i:], set[0])
	} else {
		j = bytes.IndexAny(data[i:], set)
	}
	if j < 0 {
		return len(data)
	}
	return i + j
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
	return skipUntilAny(data, i, string([]byte{c}))
}

// skipUntilUnescapedChar is like skipUntilChar for a c that is not escaped
// with a backslash.
func skipUntilUnescapedChar(data []byte, i int, c byte) int {
	for {
		i = skipUntilChar(data, i, c)
		if i >= len(data) || !isEscape(data, i) {
			return i
		}
		i++
	}
}

func skipHSpace(data []byte, i int) int {
	return skipWhile(data, i, func(b byte) bool { return b == ' ' || b == '\t' })
}

func backChar(data []byte, i int, c byte) int {
	for i > 0 && data[i-1] == c {
		i--
	}
	return i
}

// backUntilChar moves i back to just after the last c before it, or to 0.
func backUntilChar(data []byte, i int, c byte) int {
	if i <= 0 {
		return i
	}
	return bytes.LastIndexByte(data[:i], c) + 1
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

// IsLetter returns true if c is an ASCII letter
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
