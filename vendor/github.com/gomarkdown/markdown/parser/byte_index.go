package parser

import (
	"bytes"
	"sort"
)

// byteIndex stores positions of a small set of bytes in one input buffer.
// It is useful when many parser callbacks need the next occurrence without
// rescanning overlapping suffixes.
type byteIndex struct {
	data         []byte
	chars        string
	honorEscapes bool
	positions    map[byte][]int
}

func (x *byteIndex) ensure() {
	if x.positions != nil {
		return
	}
	x.positions = make(map[byte][]int, len(x.chars))
	for i := 0; i < len(x.data); i++ {
		j := bytes.IndexAny(x.data[i:], x.chars)
		if j < 0 {
			break
		}
		i += j
		if x.honorEscapes && isEscape(x.data, i) {
			continue
		}
		c := x.data[i]
		x.positions[c] = append(x.positions[c], i)
	}
}

func (x *byteIndex) next(data []byte, start int, c byte) (int, bool) {
	base, ok := sliceOffset(x.data, data)
	if !ok {
		tmp := byteIndex{data: data, chars: x.chars, honorEscapes: x.honorEscapes}
		return tmp.next(data, start, c)
	}
	x.ensure()
	positions := x.positions[c]
	i := sort.SearchInts(positions, base+start)
	if i == len(positions) {
		return 0, false
	}
	return positions[i] - base, true
}
