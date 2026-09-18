package parser

import "bytes"

// nextMatchCache remembers the next match at or after the inline cursor.
// Failed callbacks therefore do not repeatedly scan the rest of the buffer.
type nextMatchCache struct {
	data *byte
	n    int
	from int // no match starts in data[from:at]
	at   int // next match, or n when there is none
}

func (c *nextMatchCache) next(data []byte, offset int, find func([]byte) int) int {
	if c.data == &data[0] && c.n == len(data) && offset >= c.from && offset <= c.at {
		return c.at
	}
	at := len(data)
	if j := find(data[offset:]); j >= 0 {
		at = offset + j
	}
	*c = nextMatchCache{data: &data[0], n: len(data), from: offset, at: at}
	return at
}

// lastByteCache incrementally tracks the last occurrence of a byte before the
// inline cursor.
type lastByteCache struct {
	data *byte
	n    int
	upTo int
	last int
}

func (c *lastByteCache) lastAt(data []byte, offset int, b byte) int {
	if c.data != &data[0] || c.n != len(data) || offset < c.upTo {
		*c = lastByteCache{data: &data[0], n: len(data), upTo: -1, last: -1}
	}
	if j := bytes.LastIndexByte(data[c.upTo+1:offset+1], b); j >= 0 {
		c.last = c.upTo + 1 + j
	}
	c.upTo = offset
	return c.last
}
