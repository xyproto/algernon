package parser

import (
	"bytes"

	"github.com/gomarkdown/markdown/ast"
)

// single and double emphasis parsing
func emphasis(p *Parser, data []byte, offset int) (int, ast.Node) {
	// Registering '~' for subscript must not enable double-tilde syntax or
	// reinterpret the suffix of a double-tilde opener as a single opener.
	if data[offset] == '~' && p.extensions&Strikethrough == 0 {
		if offset+1 < len(data) && data[offset+1] == '~' ||
			offset > 0 && data[offset-1] == '~' && !isEscape(data, offset-1) {
			return 0, nil
		}
	}
	data = data[offset:]
	c := data[0]

	n := len(data)
	if n > 2 && data[1] != c {
		// whitespace cannot follow an opening emphasis;
		// strikethrough only takes two characters '~~'
		if IsSpace(data[1]) {
			return 0, nil
		}
		if p.extensions&SuperSubscript != 0 && c == '~' {
			// potential subscript: no spaces, except when escaped. helperEmphasis does
			// not check that for us, so walk the bytes and check.
			ret := skipUntilUnescapedChar(data[1:], 0, c)
			if ret == 0 || ret >= len(data)-1 {
				// empty, or no closing '~'
				return 0, nil
			}
			ret++ // we started with data[1:] above.
			for i := 1; i < ret; i++ {
				if IsSpace(data[i]) && !isEscape(data, i) {
					return 0, nil
				}
			}
			sub := &ast.Subscript{Leaf: ast.Leaf{Literal: data[1:ret]}}
			return ret + 1, sub
		}
		if c == '~' {
			return 0, nil
		}
		ret, node := helperEmphasis(p, data[1:], c)
		if ret == 0 {
			return 0, nil
		}

		return ret + 1, node
	}

	if n > 3 && data[1] == c && data[2] != c {
		if IsSpace(data[2]) {
			return 0, nil
		}
		ret, node := helperDoubleEmphasis(p, data[2:], c)
		if ret == 0 {
			return 0, nil
		}

		return ret + 2, node
	}

	if n > 4 && data[1] == c && data[2] == c && data[3] != c {
		if c == '~' || IsSpace(data[3]) {
			return 0, nil
		}
		ret, node := helperTripleEmphasis(p, data, 3, c)
		if ret == 0 {
			return 0, nil
		}

		return ret + 3, node
	}

	return 0, nil
}

func helperFindEmphChar(p *Parser, data []byte, c byte) int {
	i := 0
	// data is always a suffix of the buffer for the current Inline call.
	// bracketTable indexes that whole buffer, so translate local offsets before
	// looking up a closing bracket.
	base := len(p.brackets.data) - len(data)

	for i < len(data) {
		for i < len(data) && data[i] != c && data[i] != '`' && data[i] != '[' {
			i++
		}
		if i >= len(data) {
			return 0
		}
		// do not count escaped chars; an even run of backslashes escapes
		// itself, not the delimiter
		if isEscape(data, i) {
			i++
			continue
		}
		if data[i] == c {
			return i
		}

		if data[i] == '`' {
			// skip a code span
			tmpI := 0
			i++
			for i < len(data) && data[i] != '`' {
				if tmpI == 0 && data[i] == c {
					tmpI = i
				}
				i++
			}
			if i >= len(data) {
				return tmpI
			}
			i++
		} else if data[i] == '[' {
			// skip a link
			tmpI := 0
			open := i
			closeAt, _, found := p.brackets.lookup(base + open)
			end := len(p.brackets.data)
			if found {
				end = closeAt
			}
			if emphAt, ok := p.brackets.firstEmphasis(base+open+1, end, c); ok {
				tmpI = emphAt - base
			}
			if !found {
				return tmpI
			}
			close := closeAt - base
			i = close + 1
			for i < len(data) && (data[i] == ' ' || data[i] == '\n') {
				i++
			}
			if i >= len(data) {
				return tmpI
			}
			if data[i] != '[' && data[i] != '(' { // not a link
				if tmpI > 0 {
					return tmpI
				}
				continue
			}
			cc := data[i]
			i++
			for i < len(data) && data[i] != cc {
				if tmpI == 0 && data[i] == c {
					return i
				}
				i++
			}
			if i >= len(data) {
				return tmpI
			}
			i++
		}
	}
	return 0
}

func helperEmphasis(p *Parser, data []byte, c byte) (int, ast.Node) {
	i := 0

	// skip two symbols if coming from emph3, as it detected a double emphasis case
	if len(data) > 1 && data[0] == c && data[1] == c {
		i = 2
	}

	searches := p.emphSearches(data, c, false)
	defer searches.done()
	for i < len(data) {
		if searches.failedAt(i) {
			return 0, nil
		}
		length := helperFindEmphChar(p, data[i:], c)
		i += length
		if i >= len(data) {
			return 0, nil
		}
		// helperFindEmphChar returns 0 both for a delimiter at data[i] and for
		// no delimiter at all
		if length == 0 && data[i] != c {
			return 0, nil
		}

		if i+1 < len(data) && data[i+1] == c {
			i += 2
			continue
		}

		if data[i] == c && !IsSpace(data[i-1]) {
			if p.extensions&NoIntraEmphasis != 0 {
				rest := data[i+1:]
				if !(len(rest) == 0 || IsSpace(rest[0]) || IsPunctuation2(rest)) {
					if length == 0 {
						return 0, nil
					}
					continue
				}
			}

			searches.found()
			emph := &ast.Emph{}
			p.Inline(emph, data[:i])
			return i + 1, emph
		}

		// We have to check this at the end, otherwise the scenario where we find repeated c's will get skipped
		if length == 0 {
			return 0, nil
		}
	}

	return 0, nil
}

func helperDoubleEmphasis(p *Parser, data []byte, c byte) (int, ast.Node) {
	i := 0

	searches := p.emphSearches(data, c, true)
	defer searches.done()
	for i < len(data) {
		if searches.failedAt(i) {
			return 0, nil
		}
		length := helperFindEmphChar(p, data[i:], c)
		if length == 0 {
			return 0, nil
		}
		i += length

		if i+1 < len(data) && data[i] == c && data[i+1] == c && i > 0 && !IsSpace(data[i-1]) {
			// A triple closer may close an inner single emphasis before the
			// strong emphasis (for example, **bold *ital***).
			contentEnd := i
			if i+2 < len(data) && data[i+2] == c && c != '~' {
				if hasTrailingEmphOpener(data[:i], c) {
					contentEnd = i + 1
				}
			}

			searches.found()
			var node ast.Node = &ast.Strong{}
			if c == '~' {
				node = &ast.Del{}
			}
			p.Inline(node, data[:contentEnd])
			return contentEnd + 2, node
		}
		i++
	}
	return 0, nil
}

// emphSearchLog records where a search for the closer of emphasis has been,
// so that if it fails, later searches that get to the same places stop there.
type emphSearchLog struct {
	table   *bracketTable
	base    int
	c       byte
	double  bool
	visited []emphSearch
	success bool
}

// emphSearches starts a search in data, which must be a suffix of the buffer
// of the current Inline call for failures to be remembered.
func (p *Parser) emphSearches(data []byte, c byte, double bool) *emphSearchLog {
	log := &emphSearchLog{c: c, double: double}
	if base, ok := sliceOffset(p.brackets.data, data); ok {
		log.table, log.base = &p.brackets, base
	}
	return log
}

// failedAt reports whether a search from data[i] is known to fail, and
// otherwise records that the search got there.
func (l *emphSearchLog) failedAt(i int) bool {
	if l.table == nil {
		return false
	}
	s := emphSearch{pos: l.base + i, c: l.c, double: l.double}
	if l.table.emphFailed(s) {
		return true
	}
	l.visited = append(l.visited, s)
	return false
}

func (l *emphSearchLog) found() { l.success = true }

func (l *emphSearchLog) done() {
	if l.table != nil && !l.success {
		l.table.markEmphFailed(l.visited)
	}
}

// hasTrailingEmphOpener reports whether the final c is preceded by a boundary
// and followed by text, making it an unmatched opener.
func hasTrailingEmphOpener(data []byte, c byte) bool {
	last := bytes.LastIndexByte(data, c)
	if last < 0 {
		return false
	}
	return (last == 0 || IsSpace(data[last-1])) &&
		last+1 < len(data) && !IsSpace(data[last+1])
}

func helperTripleEmphasis(p *Parser, data []byte, offset int, c byte) (int, ast.Node) {
	i := 0
	origData := data
	data = data[offset:]

	for i < len(data) {
		length := helperFindEmphChar(p, data[i:], c)
		if length == 0 {
			return 0, nil
		}
		i += length

		// skip symbols preceded by whitespace
		if data[i] != c || IsSpace(data[i-1]) {
			continue
		}

		switch {
		case i+2 < len(data) && data[i+1] == c && data[i+2] == c:
			// triple symbol found
			strong := &ast.Strong{}
			em := &ast.Emph{}
			ast.AppendChild(strong, em)
			p.Inline(em, data[:i])
			return i + 3, strong
		case i+1 < len(data) && data[i+1] == c:
			// double symbol found, hand over to emph1
			length, node := helperEmphasis(p, origData[offset-2:], c)
			if length == 0 {
				return 0, nil
			}
			return length - 2, node
		default:
			// single symbol found, hand over to emph2
			length, node := helperDoubleEmphasis(p, origData[offset-1:], c)
			if length == 0 {
				return 0, nil
			}
			return length - 1, node
		}
	}
	return 0, nil
}
