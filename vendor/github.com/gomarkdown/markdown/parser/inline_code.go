package parser

import (
	"bytes"

	"github.com/gomarkdown/markdown/ast"
)

// codeSpanEnd returns the index just past the code span that opens with the
// backtick run at data[0], or 0 when that run has no closing delimiter. The
// closing delimiter is the first later run of at least as many backticks.
// It is the reference form of the search that codeSpanClosers memoises.
func codeSpanEnd(data []byte) int {
	// count the number of backticks in the delimiter
	nb := skipChar(data, 0, '`')

	// find the next delimiter
	i, end := 0, 0
	for end = nb; end < len(data) && i < nb; end++ {
		if data[end] == '`' {
			i++
		} else {
			i = 0
		}
	}

	// no matching delimiter?
	if i < nb && end >= len(data) {
		return 0
	}
	return end
}

// codeSpanCache holds the closer table for one backtick run. When the run
// has no closing delimiter, Inline advances one byte and calls codeSpan on
// the same run minus its first backtick, and so on down to a single
// backtick. Without the cache every retry rescanned to the end of the
// input, which is quadratic in the run length.
type codeSpanCache struct {
	data     *byte // first byte of the slice the entries index into
	n        int
	runStart int // the run the entries describe is data[runStart:runEnd]
	runEnd   int
	closers  []int  // closers[m]: index just past the m-th backtick of the first run of at least m backticks after runEnd, or 0
	small    [8]int // backing store for closers when the run is short, which is nearly always
}

// codeSpanClosers returns the length of the backtick run at data[offset]
// and a table whose entry m is where a delimiter of m backticks closes,
// for every m up to that length. The table is rebuilt only when the run
// changes, so retries from inside the same run cost nothing, not even the
// count of the remaining backticks.
func (p *Parser) codeSpanClosers(data []byte, offset int) (int, []int) {
	c := &p.codeSpans
	if c.data == &data[0] && c.n == len(data) && offset >= c.runStart && offset < c.runEnd {
		return c.runEnd - offset, c.closers
	}

	runEnd := skipChar(data, offset, '`')
	k := runEnd - offset
	closers := c.closers
	if cap(closers) <= k {
		closers = c.small[:]
		if k >= len(c.small) {
			closers = make([]int, 0, k+1)
		}
	}
	closers = append(closers[:0], 0)
	run, filled := 0, 0
	for j := runEnd; j < len(data) && filled < k; j++ {
		if data[j] != '`' {
			run = 0
			continue
		}
		run++
		if run > filled {
			closers = append(closers, j+1)
			filled = run
		}
	}
	for len(closers) <= k {
		closers = append(closers, 0)
	}
	c.data, c.n = &data[0], len(data)
	c.runStart, c.runEnd = offset, runEnd
	c.closers = closers
	return k, closers
}

func codeSpan(p *Parser, data []byte, offset int) (int, ast.Node) {
	nb, closers := p.codeSpanClosers(data, offset)
	if nb == 0 {
		return 0, nil
	}
	end := closers[nb] - offset
	data = data[offset:]
	if end <= 0 {
		return 0, nil
	}
	hasLFBeforeDelimiter := bytes.IndexByte(data[nb:end], '\n') >= 0

	// If there are non-space chars after the ending delimiter and before a '\n',
	// flag that this is not a well formed fenced code block.
	hasCharsAfterDelimiter := false
	for j := end; j < len(data); j++ {
		if data[j] == '\n' {
			break
		}
		if !IsSpace(data[j]) {
			hasCharsAfterDelimiter = true
			break
		}
	}

	// trim outside whitespace
	fBegin := nb
	for fBegin < end && data[fBegin] == ' ' {
		fBegin++
	}

	fEnd := end - nb
	for fEnd > fBegin && data[fEnd-1] == ' ' {
		fEnd--
	}

	if fBegin == fEnd {
		return end, nil
	}

	// if delimiter has 3 backticks
	if nb == 3 {
		i := fBegin
		syntaxStart, syntaxLen := syntaxRange(data, &i)

		// If we found a '\n' before the end marker and there are only spaces
		// after the end marker, then this is a code block.
		if hasLFBeforeDelimiter && !hasCharsAfterDelimiter {
			codeblock := &ast.CodeBlock{
				IsFenced: true,
				Info:     data[syntaxStart : syntaxStart+syntaxLen],
			}
			codeblock.Literal = data[i:fEnd]
			return end, codeblock
		}
	}

	// render the code span
	code := &ast.Code{}
	code.Literal = data[fBegin:fEnd]
	return end, code
}
