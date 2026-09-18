package parser

import (
	"bytes"

	"github.com/gomarkdown/markdown/ast"
)

// attribute parses a (potential) block attribute and adds it to p.
func (p *Parser) attribute(data []byte) []byte {
	attr, n := parseAttributeList(data, true)
	if n == 0 {
		return data
	}
	p.attr = attr
	return data[n:]
}

// parseAttributeList parses {#id .class key="value"} or {: key="value"}.
// If requireEOL is true, '}' must be the last character on the line
// (block IAL). Returns the attribute and bytes consumed, or (nil, 0).
func parseAttributeList(data []byte, requireEOL bool) (*ast.Attribute, int) {
	if len(data) < 3 || data[0] != '{' {
		return nil, 0
	}
	if requireEOL {
		end := skipUntilChar(data, 1, '\n')
		if data[end-1] != '}' {
			return nil, 0
		}
	}

	i := 1
	if i < len(data) && data[i] == ':' {
		i++
	}
	for i < len(data) && (data[i] == ' ' || data[i] == '\t' || data[i] == '\f' || data[i] == '\v') {
		i++
	}

	b := &ast.Attribute{Attrs: make(map[string][]byte)}
	esc := false
	quote := false
	trail := i - 1
	for ; i < len(data); i++ {
		switch data[i] {
		case '\n', '\r':
			return nil, 0
		case ' ', '\t', '\f', '\v':
			if quote {
				continue
			}
			chunk := data[trail+1 : i]
			if len(chunk) > 0 && !addAttrChunk(b, chunk) {
				return nil, 0
			}
			trail = i
		case '"':
			if esc {
				esc = false
				continue
			}
			quote = !quote
		case '\\':
			esc = !esc
		case '}':
			if esc {
				esc = false
				continue
			}
			chunk := data[trail+1 : i]
			if len(chunk) == 0 || !addAttrChunk(b, chunk) {
				return nil, 0
			}
			i++
			return b, i
		default:
			esc = false
		}
	}
	return nil, 0
}

func addAttrChunk(b *ast.Attribute, chunk []byte) bool {
	switch {
	case chunk[0] == '.':
		b.Classes = append(b.Classes, chunk[1:])
	case chunk[0] == '#':
		b.ID = chunk[1:]
	default:
		k, v := keyValue(chunk)
		if k == nil || v == nil {
			return false
		}
		b.Attrs[string(k)] = v
	}
	return true
}

func mergeAttribute(dst, src *ast.Attribute) *ast.Attribute {
	if src == nil {
		return dst
	}
	if dst == nil {
		return src
	}
	if len(src.ID) > 0 {
		dst.ID = src.ID
	}
	dst.Classes = append(dst.Classes, src.Classes...)
	if dst.Attrs == nil {
		dst.Attrs = make(map[string][]byte)
	}
	for k, v := range src.Attrs {
		dst.Attrs[k] = v
	}
	return dst
}

func applyAttribute(n ast.Node, attr *ast.Attribute) {
	if n == nil || attr == nil {
		return
	}
	if c := n.AsContainer(); c != nil {
		c.Attribute = mergeAttribute(c.Attribute, attr)
		return
	}
	if l := n.AsLeaf(); l != nil {
		l.Attribute = mergeAttribute(l.Attribute, attr)
	}
}

// applyAfterBlockAttribute parses a kramdown-style IAL on the line after
// a block (`{: key="value"}`) and merges it onto the preceding block.
// `{#id}` without a colon is left for prefix attributes so existing
// Attributes documents keep applying those to the next block.
func (p *Parser) applyAfterBlockAttribute(data []byte) int {
	prev := p.lastBlock()
	if prev == nil {
		return 0
	}
	attr, consumed := parseAfterBlockAttribute(data)
	if consumed == 0 {
		return 0
	}
	applyAttribute(prev, attr)
	return consumed
}

func parseAfterBlockAttribute(data []byte) (*ast.Attribute, int) {
	i := 0
	if len(data) > 0 && data[0] == '\n' {
		i++
	}
	i = skipCharN(data, i, ' ', 3)
	if i+1 >= len(data) || data[i] != '{' || data[i+1] != ':' {
		return nil, 0
	}
	attr, n := parseAttributeList(data[i:], true)
	if n == 0 {
		return nil, 0
	}
	consumed := i + n
	if consumed < len(data) && data[consumed] == '\n' {
		consumed++
	}
	return attr, consumed
}

func isAfterBlockIAL(data []byte) bool {
	_, consumed := parseAfterBlockAttribute(data)
	return consumed > 0
}

func (p *Parser) lastBlock() ast.Node {
	n := p.tip
	if n == nil {
		return nil
	}
	if _, ok := n.(*ast.Document); ok {
		return ast.GetLastChild(n)
	}
	return n
}

// key="value" quotes are mandatory.
func keyValue(data []byte) ([]byte, []byte) {
	separator := bytes.IndexByte(data, '=')
	if separator <= 0 {
		return nil, nil
	}
	key, value := data[:separator], data[separator+1:]

	if len(value) < 3 || value[0] != '"' || value[len(value)-1] != '"' {
		return key, nil
	}
	return key, value[1 : len(value)-1]
}
