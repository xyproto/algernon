package parser

import (
	"regexp"

	"github.com/gomarkdown/markdown/ast"
)

// Parsing of inline elements

var (
	urlRe    = `((https?|ftp):\/\/|\/)[-A-Za-z0-9+&@#\/%?=~_|!:,.;\(\)]+`
	anchorRe = regexp.MustCompile(`^(<a\shref="` + urlRe + `"(\stitle="[^"<>]+")?\s?>` + urlRe + `<\/a>)`)
)

// Inline parses text within a block.
// Each function returns the number of consumed chars.
func (p *Parser) Inline(currBlock ast.Node, data []byte) {
	// handlers might call us recursively: enforce a maximum depth
	if p.nesting >= p.maxNesting || len(data) == 0 {
		return
	}
	if p.nesting == 0 {
		p.resetInlineCaches()
	}
	p.nesting++
	prev := p.brackets
	p.brackets = bracketTable{data: data}
	defer func() {
		p.brackets = prev
		p.nesting--
	}()
	beg, end := 0, 0

	n := len(data)
	for end < n {
		handler := p.inlineCallback[data[end]]
		if handler == nil {
			end++
			continue
		}
		consumed, node := handler(p, data, end)
		if consumed == 0 {
			// no action from the callback
			end++
			continue
		}
		// copy inactive chars into the output
		ast.AppendChild(currBlock, newTextNode(data[beg:end]))
		if node != nil {
			ast.AppendChild(currBlock, node)
		}
		beg = end + consumed
		end = beg
	}

	if beg < n {
		if data[end-1] == '\n' {
			end--
		}
		ast.AppendChild(currBlock, newTextNode(data[beg:end]))
	}
}

// resetInlineCaches drops the memo tables that inline callbacks keep for
// the buffer under the cursor. They key on the buffer's address and
// length, so a caller that reuses a Parser on a buffer it has rewritten in
// place must not see entries built from the old contents.
func (p *Parser) resetInlineCaches() {
	p.codeSpans.data = nil
	p.spaces.data = nil
	p.angles.data = nil
	p.nextGt.data = nil
	p.nextCommentEnd.data = nil
}

// runCache remembers the end of the run of one byte value that the inline
// cursor is inside. A callback registered for that byte fires once per
// byte of the run, and without the cache each firing rescanned the run.
type runCache struct {
	data       *byte
	n          int
	start, end int
}

// endOf returns the index just past the run of b that starts at
// data[offset], scanning only when offset is outside the remembered run.
func (c *runCache) endOf(data []byte, offset int, b byte) int {
	if c.data == &data[0] && c.n == len(data) && offset >= c.start && offset < c.end {
		return c.end
	}
	end := skipChar(data, offset, b)
	*c = runCache{data: &data[0], n: len(data), start: offset, end: end}
	return end
}

// newline preceded by two spaces becomes <br>
func maybeLineBreak(p *Parser, data []byte, offset int) (int, ast.Node) {
	origOffset := offset
	offset = p.spaces.endOf(data, offset, ' ')

	if offset < len(data) && data[offset] == '\n' {
		if offset-origOffset >= 2 {
			return offset - origOffset + 1, &ast.Hardbreak{}
		}
		return offset - origOffset, nil
	}
	return 0, nil
}

// newline without two spaces works when HardLineBreak is enabled
func lineBreak(p *Parser, data []byte, offset int) (int, ast.Node) {
	if p.extensions&HardLineBreak != 0 {
		return 1, &ast.Hardbreak{}
	}
	return 0, nil
}

func maybeImage(p *Parser, data []byte, offset int) (int, ast.Node) {
	if offset < len(data)-1 && data[offset+1] == '[' {
		return link(p, data, offset)
	}
	return 0, nil
}

func maybeInlineFootnoteOrSuper(p *Parser, data []byte, offset int) (int, ast.Node) {
	if offset < len(data)-1 && data[offset+1] == '[' {
		return link(p, data, offset)
	}

	if p.extensions&SuperSubscript != 0 {
		ret := skipUntilChar(data[offset:], 1, '^')
		if ret >= len(data)-offset {
			// no closing '^'
			return 0, nil
		}
		for i := offset; i < offset+ret; i++ {
			if IsSpace(data[i]) && !isEscape(data, i) {
				return 0, nil
			}
		}
		sup := &ast.Superscript{Leaf: ast.Leaf{Literal: data[offset+1 : offset+ret]}}
		return ret + 1, sup
	}

	return 0, nil
}

func math(_ *Parser, data []byte, offset int) (int, ast.Node) {
	data = data[offset:]

	// too short, or block math
	if len(data) <= 2 || data[1] == '$' {
		return 0, nil
	}

	// find next '$'
	var end int
	for end = 1; end < len(data) && data[end] != '$'; end++ {
	}

	// $ not match
	if end == len(data) {
		return 0, nil
	}

	return end + 1, &ast.Math{Leaf: ast.Leaf{Literal: data[1:end]}}
}

func newTextNode(d []byte) *ast.Text {
	return &ast.Text{Leaf: ast.Leaf{Literal: d}}
}
