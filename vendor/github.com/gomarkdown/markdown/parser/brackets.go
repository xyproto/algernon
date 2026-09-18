package parser

// bracketTable maps each '[' in one Inline() buffer to its matching ']'.
// Built once per buffer so a run of unmatched '[' is O(n) (GHSA-85vw-wvf9-r522).
type bracketTable struct {
	data   []byte
	close  []int
	nested []bool
}

func (t *bracketTable) lookup(open int) (closeAt int, nested, ok bool) {
	t.ensure()
	if open < 0 || open >= len(t.close) {
		return 0, false, false
	}
	closeAt = t.close[open]
	if closeAt < 0 {
		return 0, false, false
	}
	return closeAt, t.nested[open], true
}

func (t *bracketTable) ensure() {
	if t.close != nil {
		return
	}
	n := len(t.data)
	closeAt := make([]int, n)
	nested := make([]bool, n)
	for i := 0; i < n; i++ {
		closeAt[i] = -1
	}
	stack := make([]int, 0, 16)
	for i := 0; i < n; i++ {
		// an odd run of backslashes escapes the bracket; an even run is
		// escaped backslashes and leaves the bracket live
		if isEscape(t.data, i) {
			continue
		}
		switch t.data[i] {
		case '[':
			stack = append(stack, i)
		case ']':
			if len(stack) == 0 {
				continue
			}
			open := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			closeAt[open] = i
			if len(stack) > 0 {
				nested[stack[len(stack)-1]] = true
			}
		}
	}
	t.close = closeAt
	t.nested = nested
}
