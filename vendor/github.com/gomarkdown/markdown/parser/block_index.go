package parser

import (
	"bytes"
	"sort"
)

// blockIndex stores line-oriented facts that would otherwise be rediscovered
// while paragraph parsing retries a block parser at each successive line.
type blockIndex struct {
	data         []byte
	fenceClosers map[string][]int
}

func (x *blockIndex) ensureFences() {
	if x.fenceClosers != nil {
		return
	}
	x.fenceClosers = make(map[string][]int)
	for start := 0; start < len(x.data); {
		end := bytes.IndexByte(x.data[start:], '\n')
		if end < 0 {
			end = len(x.data)
		} else {
			end += start + 1
		}
		line := x.data[start:end]
		if _, marker := isFenceLine(line, nil, ""); marker != "" {
			if consumed, _ := isFenceLine(line, nil, marker); consumed > 0 {
				x.fenceClosers[marker] = append(x.fenceClosers[marker], start)
			}
		}
		start = end
	}
}

// hasFenceCloser reports whether data[beg:] contains a closing line for
// marker. If data is not part of the current Block buffer, it conservatively
// returns true and lets the caller use its ordinary scan.
func (x *blockIndex) hasFenceCloser(data []byte, beg int, marker string) bool {
	base, ok := sliceOffset(x.data, data)
	if !ok {
		return true
	}
	x.ensureFences()
	positions := x.fenceClosers[marker]
	i := sort.SearchInts(positions, base+beg)
	return i < len(positions)
}
