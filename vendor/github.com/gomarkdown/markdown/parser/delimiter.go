package parser

// delimiterTable maps opening delimiters to their matching closing delimiter
// in one input buffer. Matching is nested and can ignore escaped delimiters.
type delimiterTable struct {
	data           []byte
	open, close    byte
	honorEscapes   bool
	escapePrevious bool
	trackNested    bool
	closeAt        []int
	nested         []bool
}

func (t *delimiterTable) lookup(open int) (closeAt int, nested, ok bool) {
	t.ensure()
	if open < 0 || open >= len(t.closeAt) || t.closeAt[open] < 0 {
		return 0, false, false
	}
	nested = t.trackNested && t.nested[open]
	return t.closeAt[open], nested, true
}

func (t *delimiterTable) ensure() {
	if t.closeAt != nil {
		return
	}
	n := len(t.data)
	t.closeAt = make([]int, n)
	if t.trackNested {
		t.nested = make([]bool, n)
	}
	for i := range t.closeAt {
		t.closeAt[i] = -1
	}
	stack := make([]int, 0, 16)
	backslashes := 0
	for i, c := range t.data {
		if t.escapePrevious && i > 0 && t.data[i-1] == '\\' {
			continue
		}
		if t.honorEscapes && c == '\\' {
			backslashes++
			continue
		}
		escaped := t.honorEscapes && backslashes%2 != 0
		backslashes = 0
		if escaped {
			continue
		}
		switch c {
		case t.open:
			stack = append(stack, i)
		case t.close:
			if len(stack) == 0 {
				continue
			}
			opening := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			t.closeAt[opening] = i
			if t.trackNested && len(stack) > 0 {
				t.nested[stack[len(stack)-1]] = true
			}
		}
	}
}

// lookupDelimiter uses the Inline-scoped table when data is one of its
// suffixes. Direct helper calls fall back to a temporary table over data.
func lookupDelimiter(t *delimiterTable, data []byte, offset int, open, close byte) (int, bool, bool) {
	if base, ok := sliceOffset(t.data, data); ok && t.open == open && t.close == close {
		closeAt, nested, found := t.lookup(base + offset)
		if !found {
			return 0, false, false
		}
		return closeAt - base, nested, found
	}
	tmp := delimiterTable{
		data: data, open: open, close: close,
		honorEscapes: t.honorEscapes, escapePrevious: t.escapePrevious,
		trackNested: t.trackNested,
	}
	return tmp.lookup(offset)
}

// closerBalanceCache incrementally determines whether a closing delimiter has
// an unmatched opener earlier on the same line.
type closerBalanceCache struct {
	data                    *byte
	n, scanned              int
	parens, squares, braces int
	lastMatched             bool
}

func (c *closerBalanceCache) hasOpener(data []byte, closeAt int) bool {
	if len(data) == 0 || closeAt < 0 || closeAt >= len(data) {
		return false
	}
	if c.data != &data[0] || c.n != len(data) || closeAt < c.scanned {
		*c = closerBalanceCache{data: &data[0], n: len(data), scanned: -1}
	}
	if closeAt == c.scanned {
		return c.lastMatched
	}
	matched := false
	for i := c.scanned + 1; i <= closeAt; i++ {
		switch data[i] {
		case '\n':
			c.parens, c.squares, c.braces = 0, 0, 0
		case '(':
			c.parens++
		case ')':
			matched = c.parens > 0
			if matched {
				c.parens--
			}
		case '[':
			c.squares++
		case ']':
			matched = c.squares > 0
			if matched {
				c.squares--
			}
		case '{':
			c.braces++
		case '}':
			matched = c.braces > 0
			if matched {
				c.braces--
			}
		}
	}
	c.scanned = closeAt
	c.lastMatched = matched
	return matched
}
