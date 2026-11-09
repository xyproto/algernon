package parser

import "bytes"

// bufferID identifies the buffer a cache's entries index into: the address
// of its first byte and its length. The zero value matches no buffer.
type bufferID struct {
	data *byte
	n    int
}

// idOf returns the bufferID of data, which must not be empty.
func idOf(data []byte) bufferID { return bufferID{&data[0], len(data)} }

// covers reports whether data is the buffer b identifies and offset is in
// [lo, hi), the range a cache entry is valid for.
func (b bufferID) covers(data []byte, offset, lo, hi int) bool {
	return b == idOf(data) && offset >= lo && offset < hi
}

// nextMatchCache remembers the next match at or after the inline cursor.
// Failed callbacks therefore do not repeatedly scan the rest of the buffer.
type nextMatchCache struct {
	buf  bufferID
	from int // no match starts in data[from:at]
	at   int // next match, or len(data) when there is none
}

func (c *nextMatchCache) next(data []byte, offset int, find func([]byte) int) int {
	if c.buf.covers(data, offset, c.from, c.at+1) {
		return c.at
	}
	at := len(data)
	if j := find(data[offset:]); j >= 0 {
		at = offset + j
	}
	*c = nextMatchCache{buf: idOf(data), from: offset, at: at}
	return at
}

// lastByteCache incrementally tracks the last occurrence of a byte before the
// inline cursor.
type lastByteCache struct {
	buf  bufferID
	upTo int
	last int
}

func (c *lastByteCache) lastAt(data []byte, offset int, b byte) int {
	if c.buf != idOf(data) || offset < c.upTo {
		*c = lastByteCache{buf: idOf(data), upTo: -1, last: -1}
	}
	if j := bytes.LastIndexByte(data[c.upTo+1:offset+1], b); j >= 0 {
		c.last = c.upTo + 1 + j
	}
	c.upTo = offset
	return c.last
}
