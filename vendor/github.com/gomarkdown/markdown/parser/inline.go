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

// Inline parses data as inline content and appends the nodes to currBlock.
func (p *Parser) Inline(currBlock ast.Node, data []byte) {
	if p.Opts.Flags&CommonMark != 0 {
		p.commonMarkInline(currBlock, data)
		return
	}
	// handlers might call us recursively: enforce a maximum depth
	if p.nesting >= p.maxNesting || len(data) == 0 {
		return
	}
	if p.nesting == 0 {
		p.resetInlineCaches()
	}
	p.nesting++
	prevBrackets := p.brackets
	prevCitationBrackets := p.citationBrackets
	prevParens := p.parens
	prevLinkStops := p.linkStops
	prevPlainParens := p.plainParens
	prevCloserBalance := p.closerBalance
	p.brackets = bracketTable{data: data}
	p.citationBrackets = delimiterTable{data: data, open: '[', close: ']', escapePrevious: true}
	p.parens = delimiterTable{data: data, open: '(', close: ')', honorEscapes: true}
	p.linkStops = byteIndex{data: data, chars: "'\")", honorEscapes: true}
	p.plainParens = delimiterTable{data: data, open: '(', close: ')'}
	p.closerBalance = closerBalanceCache{scanned: -1}
	defer func() {
		p.brackets = prevBrackets
		p.citationBrackets = prevCitationBrackets
		p.parens = prevParens
		p.linkStops = prevLinkStops
		p.plainParens = prevPlainParens
		p.closerBalance = prevCloserBalance
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
// the buffer under the cursor. It runs at the start of each top-level Inline
// call. The tables key on the buffer's address and length, so a later call on
// a buffer that was rewritten in place must not see entries built from the
// old contents.
func (p *Parser) resetInlineCaches() {
	p.codeSpans.buf = bufferID{}
	p.spaces.buf = bufferID{}
	p.angles.buf = bufferID{}
	p.nextGt.buf = bufferID{}
	p.nextCommentEnd.buf = bufferID{}
}

// runCache remembers the end of the run of one byte value that the inline
// cursor is inside. A callback registered for that byte fires once per
// byte of the run, and without the cache each firing rescanned the run.
type runCache struct {
	buf        bufferID
	start, end int
}

// endOf returns the index just past the run of b that starts at
// data[offset], scanning only when offset is outside the remembered run.
func (c *runCache) endOf(data []byte, offset int, b byte) int {
	if c.buf.covers(data, offset, c.start, c.end) {
		return c.end
	}
	end := skipChar(data, offset, b)
	*c = runCache{buf: idOf(data), start: offset, end: end}
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

// linkAfterMarker parses a link when a '[' follows the marker at
// data[offset], as in ![alt] and ^[note]. found reports whether it did.
func linkAfterMarker(p *Parser, data []byte, offset int) (consumed int, node ast.Node, found bool) {
	if !nextByteIs(data, offset, '[') {
		return 0, nil, false
	}
	consumed, node = link(p, data, offset)
	return consumed, node, true
}

func maybeImage(p *Parser, data []byte, offset int) (int, ast.Node) {
	consumed, node, _ := linkAfterMarker(p, data, offset)
	return consumed, node
}

func maybeInlineFootnoteOrSuper(p *Parser, data []byte, offset int) (int, ast.Node) {
	if consumed, node, found := linkAfterMarker(p, data, offset); found {
		return consumed, node
	}

	if p.extensions&SuperSubscript != 0 {
		ret := skipUntilUnescapedChar(data[offset:], 1, '^')
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

	// no matching $
	if end == len(data) {
		return 0, nil
	}

	return end + 1, &ast.Math{Leaf: ast.Leaf{Literal: data[1:end]}}
}

func newTextNode(d []byte) *ast.Text {
	return &ast.Text{Leaf: ast.Leaf{Literal: d}}
}
