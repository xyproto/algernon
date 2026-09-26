package parser

import "sort"

// bracketTable maps each '[' in one Inline() buffer to its matching ']'.
// Built once per buffer so a run of unmatched '[' is O(n) (GHSA-85vw-wvf9-r522).
type bracketTable struct {
	data  []byte
	pairs delimiterTable
	emph  [3][]int
}

func (t *bracketTable) lookup(open int) (closeAt int, nested, ok bool) {
	t.ensure()
	return t.pairs.lookup(open)
}

// firstEmphasis returns the first emphasis delimiter in [start, end). The
// positions are collected while building the bracket table, so an unmatched
// '[' does not require another scan to the end of the inline buffer.
func (t *bracketTable) firstEmphasis(start, end int, c byte) (int, bool) {
	t.ensure()
	var slot int
	switch c {
	case '*':
		slot = 0
	case '_':
		slot = 1
	case '~':
		slot = 2
	default:
		return 0, false
	}
	positions := t.emph[slot]
	i := sort.SearchInts(positions, start)
	if i == len(positions) || positions[i] >= end {
		return 0, false
	}
	return positions[i], true
}

func (t *bracketTable) ensure() {
	if t.pairs.closeAt != nil {
		return
	}
	t.pairs = delimiterTable{
		data: t.data, open: '[', close: ']',
		honorEscapes: true, trackNested: true,
	}
	t.pairs.ensure()
	for i, c := range t.data {
		switch c {
		case '*':
			t.emph[0] = append(t.emph[0], i)
		case '_':
			t.emph[1] = append(t.emph[1], i)
		case '~':
			t.emph[2] = append(t.emph[2], i)
		}
	}
}
