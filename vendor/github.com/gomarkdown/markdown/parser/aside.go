package parser

import (
	"bytes"

	"github.com/gomarkdown/markdown/ast"
)

// returns aisde prefix length
func asidePrefix(data []byte) int {
	return prefixedBlock(data, "A>")
}

// parse a aside fragment
func (p *Parser) aside(data []byte) int {
	var raw bytes.Buffer
	beg, end := 0, 0
	// identical to quote
	for beg < len(data) {
		end = beg
		// Step over whole lines, collecting them. While doing that, check for
		// fenced code and if one's found, incorporate it altogether,
		// irregardless of any contents inside it
		for end < len(data) && data[end] != '\n' {
			if p.extensions&FencedCode != 0 {
				if i := p.fencedCodeBlock(data[end:], false); i > 0 {
					// -1 to compensate for the extra end++ after the loop:
					end += i - 1
					break
				}
			}
			end++
		}
		end = skipCharN(data, end, '\n', 1)
		if pre := asidePrefix(data[beg:]); pre > 0 {
			// skip the prefix
			beg += pre
		} else if terminatesPrefixedBlock(data, beg, end, asidePrefix) {
			break
		}
		// this line is part of the aside
		raw.Write(data[beg:end])
		beg = end
	}

	block := p.AddBlock(&ast.Aside{})
	p.Block(raw.Bytes())
	p.Finalize(block)
	return end
}
