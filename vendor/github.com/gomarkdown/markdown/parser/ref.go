package parser

import (
	"bytes"
	"fmt"

	"github.com/gomarkdown/markdown/ast"
)

// parse a cross reference '(#r, text)', where r can't contain spaces but text
// can (similar to a citation), or an index entry: (!item) or (!item, subitem),
// where (!!item) marks the primary entry.
func maybeShortRefOrIndex(p *Parser, data []byte, offset int) (int, ast.Node) {
	if len(data[offset:]) < 4 {
		return 0, nil
	}
	// short ref first
	data = data[offset:]
	i := 1
	switch data[i] {
	case '#': // cross ref
		i++
	Loop:
		for i < len(data) {
			c := data[i]
			switch {
			case c == ')':
				break Loop
			case !IsAlnum(c):
				if c == '_' || c == '-' || c == ':' || c == ' ' || c == ',' {
					i++
					continue
				}
				i = 0
				break Loop
			}
			i++
		}
		if i >= len(data) {
			return 0, nil
		}
		if data[i] != ')' {
			return 0, nil
		}

		id := data[2:i]
		node := &ast.CrossReference{}
		node.Destination = id
		if c := bytes.Index(id, []byte(",")); c > 0 {
			idpart := id[:c]
			suff := id[c+1:]
			suff = bytes.TrimSpace(suff)
			node.Destination = idpart
			node.Suffix = suff
		}
		// no spaces or commas allowed in id, including a leading one
		for _, c := range []byte(" ,") {
			if bytes.IndexByte(node.Destination, c) >= 0 {
				return 0, nil
			}
		}

		return i + 1, node

	case '!': // index
		i++
		start := i
		if _, _, found := lookupDelimiter(&p.plainParens, data, 0, '(', ')'); !found {
			return 0, nil
		}
		i = skipUntilChar(data, start, ')')

		// did we reach the end of the buffer without a closing marker?
		if i >= len(data) {
			return 0, nil
		}

		if len(data[start:i]) < 1 {
			return 0, nil
		}

		idx := &ast.Index{}

		idx.Primary = data[start] == '!'
		buf := data[start:i]

		if idx.Primary {
			buf = buf[1:]
		}
		items := bytes.Split(buf, []byte(","))
		switch len(items) {
		case 1:
			idx.Item = bytes.TrimSpace(items[0])
		case 2:
			idx.Item = bytes.TrimSpace(items[0])
			idx.Subitem = bytes.TrimSpace(items[1])
		default:
			// Not an index. Leave the id counter for an entry that is emitted.
			return 0, nil
		}
		idx.ID = fmt.Sprintf("idxref:%d", p.indexCnt)
		p.indexCnt++
		return i + 1, idx
	}

	return 0, nil
}
