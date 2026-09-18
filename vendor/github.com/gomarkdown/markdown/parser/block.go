package parser

import (
	"github.com/gomarkdown/markdown/ast"
)

// Parsing block-level elements.

const (
	captionTable  = "Table: "
	captionFigure = "Figure: "
	captionQuote  = "Quote: "
)

func (p *Parser) Block(data []byte) {
	// this is called recursively: enforce a maximum depth
	if p.nesting >= p.maxNesting {
		return
	}
	p.nesting++

	// parse out one block-level construct at a time
	for len(data) > 0 {
		// Attributes may occur before a block or as a kramdown-style IAL after it.
		if p.extensions&Attributes != 0 {
			if n := p.applyAfterBlockAttribute(data); n > 0 {
				data = data[n:]
				continue
			}
			data = p.attribute(data)
		}

		if p.extensions&Includes != 0 {
			if consumed := p.parseInclude(data); consumed > 0 {
				data = data[consumed:]
				continue
			}
		}

		// user supplied parser function
		if p.Opts.ParserHook != nil {
			node, blockdata, consumed := p.Opts.ParserHook(data)
			if consumed > 0 {
				data = data[consumed:]

				if node != nil {
					p.AddBlock(node)
					if blockdata != nil {
						p.Block(blockdata)
						p.Finalize(node)
					}
				}
				continue
			}
		}

		if len(data) == 0 {
			continue
		}
		data = data[p.parseBlock(data):]
	}

	p.nesting--
}

func (p *Parser) AddBlock(n ast.Node) ast.Node {
	p.closeUnmatchedBlocks()

	if p.attr != nil {
		if c := n.AsContainer(); c != nil {
			c.Attribute = p.attr
		} else if l := n.AsLeaf(); l != nil {
			l.Attribute = p.attr
		}
		p.attr = nil
	}
	return p.addChild(n)
}

func (p *Parser) blockMath(data []byte) int {
	if len(data) <= 4 || data[0] != '$' || data[1] != '$' || data[2] == '$' {
		return 0
	}

	// find next $$
	var end int
	for end = 2; end+1 < len(data) && (data[end] != '$' || data[end+1] != '$'); end++ {
	}

	// $$ not match
	if end+1 == len(data) {
		return 0
	}

	p.AddBlock(&ast.MathBlock{Container: ast.Container{Literal: data[2:end]}})

	return end + 2
}
