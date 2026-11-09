package parser

import (
	"bytes"
	"sort"
)

// blockIndex stores line-oriented facts that would otherwise be rediscovered
// while paragraph parsing retries a block parser at each successive line.
type blockIndex struct {
	data []byte
	// fenceClosers maps a block prefix such as ">" (or "" for none) to the
	// starts of lines that can close a fence with a given marker once that
	// prefix, if the line has it, is removed.
	fenceClosers map[string]map[string][]int
	// commentEnds holds the start of each "-->", once looked for
	commentEnds []int
	hasComments bool
}

// commentEnd returns the index of the first "-->" in data at or after from,
// or -1 if there is none.
func (x *blockIndex) commentEnd(data []byte, from int) int {
	base, ok := sliceOffset(x.data, data)
	if !ok {
		if i := bytes.Index(data[from:], commentEnd); i >= 0 {
			return from + i
		}
		return -1
	}
	if !x.hasComments {
		x.hasComments = true
		for i := 0; ; i++ {
			j := bytes.Index(x.data[i:], commentEnd)
			if j < 0 {
				break
			}
			i += j
			x.commentEnds = append(x.commentEnds, i)
		}
	}
	i := sort.SearchInts(x.commentEnds, base+from)
	if i == len(x.commentEnds) {
		return -1
	}
	return x.commentEnds[i] - base
}

func (x *blockIndex) ensureFences(prefix string) map[string][]int {
	if closers, ok := x.fenceClosers[prefix]; ok {
		return closers
	}
	if x.fenceClosers == nil {
		x.fenceClosers = make(map[string]map[string][]int)
	}
	closers := make(map[string][]int)
	x.fenceClosers[prefix] = closers
	for start := 0; start < len(x.data); {
		_, n := commonMarkLine(x.data[start:])
		end := start + n
		line := x.data[start:end]
		if prefix != "" {
			line = line[prefixedBlock(line, prefix):]
		}
		if marker := closingFence(line); marker != "" {
			closers[marker] = append(closers[marker], start)
		}
		start = end
	}
	return closers
}

// closingFence returns the marker of line if it can close a fence, or "".
func closingFence(line []byte) string {
	if _, marker := isFenceLine(line, nil, ""); marker != "" {
		if consumed, _ := isFenceLine(line, nil, marker); consumed > 0 {
			return marker
		}
	}
	return ""
}

// hasFenceCloser reports whether data[beg:] contains a closing line for
// marker. If data is not part of the current Block buffer, it conservatively
// returns true and lets the caller use its ordinary scan.
func (x *blockIndex) hasFenceCloser(data []byte, beg int, marker string) bool {
	return x.hasPrefixedFenceCloser(data, beg, marker, "")
}

// hasPrefixedFenceCloser is like hasFenceCloser for a fence inside a block
// whose lines start with prefix, such as a quote. The closing line may or may
// not have the prefix.
func (x *blockIndex) hasPrefixedFenceCloser(data []byte, beg int, marker, prefix string) bool {
	base, ok := sliceOffset(x.data, data)
	if !ok {
		return true
	}
	positions := x.ensureFences(prefix)[marker]
	i := sort.SearchInts(positions, base+beg)
	return i < len(positions)
}
