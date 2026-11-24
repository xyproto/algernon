package parser

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gomarkdown/markdown/internal/textutil"
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
	i = textutil.SkipCharN(data, i, '\n', 1)
	return i
}

// nextByteIs reports whether the byte after data[i] is c.
func nextByteIs(data []byte, i int, c byte) bool {
	return textutil.ByteAt(data, i+1, c)
}

// byteIn reports whether c is one of the bytes in set.
func byteIn(c byte, set string) bool {
	return strings.IndexByte(set, c) >= 0
}

// pairAt reports whether data[i] is a and the byte after it is b.
func pairAt(data []byte, i int, a, b byte) bool {
	return textutil.ByteAt(data, i, a) && nextByteIs(data, i, b)
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
	return textutil.SkipWhile(data, i, func(b byte) bool { return b == ' ' || b == '\t' })
}

// lineEnd returns the end of the line holding data[i-1]: the index after its
// '\n', or len(data). It returns i if i is 0 or not before len(data).
func lineEnd(data []byte, i int) int {
	if i <= 0 || i >= len(data) {
		return i
	}
	if j := skipUntilChar(data, i-1, '\n'); j < len(data) {
		return j + 1
	}
	return len(data)
}

// nextGatheredLine finds the next line of a block whose lines are gathered
// one at a time: the line starts at data[start] and ends at or after i+1. It
// returns the end of the line and whether it is blank, and records a blank
// line in *sawBlank.
func nextGatheredLine(data []byte, start, i int, sawBlank *bool) (end int, blank bool) {
	end = lineEnd(data, i+1)
	blank = IsEmpty(data[start:end]) > 0
	if blank {
		*sawBlank = true
	}
	return end, blank
}

// hasUnescapedSpace reports whether data[start:end] has a space that is not
// escaped with a backslash.
func hasUnescapedSpace(data []byte, start, end int) bool {
	for i := start; i < end; i++ {
		if IsSpace(data[i]) && !isEscape(data, i) {
			return true
		}
	}
	return false
}

// backUntilChar moves i back to just after the last c before it, or to 0.
func backUntilChar(data []byte, i int, c byte) int {
	if i <= 0 {
		return i
	}
	return bytes.LastIndexByte(data[:i], c) + 1
}

func prefixedBlock(data []byte, marker string) int {
	i := textutil.SkipCharN(data, 0, ' ', 3)
	if !bytes.HasPrefix(data[i:], []byte(marker)) {
		return 0
	}
	i += len(marker)
	return textutil.SkipCharN(data, i, ' ', 1)
}

func terminatesPrefixedBlock(data []byte, begin, end int, prefix func([]byte) int) bool {
	return IsEmpty(data[begin:]) > 0 &&
		(end >= len(data) || prefix(data[end:]) == 0 && IsEmpty(data[end:]) == 0)
}

// punctuation is the ASCII set IsPunctuation recognizes. A string avoids
// allocating that set on every call; emphasis checks it for each closer.
const punctuation = "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"

// IsPunctuation returns true if c is a punctuation symbol.
func IsPunctuation(c byte) bool {
	return strings.IndexByte(punctuation, c) >= 0
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
