package parser

import (
	"github.com/gomarkdown/markdown/ast"
)

// returns aside prefix length
func asidePrefix(data []byte) int {
	return prefixedBlock(data, "A>")
}

// parse an aside fragment
func (p *Parser) aside(data []byte) int {
	raw, end := p.prefixedBlockContent(data, "A>")
	block := p.AddBlock(&ast.Aside{})
	p.Block(raw)
	p.Finalize(block)
	return end
}
