package parser

import (
	"bytes"

	"github.com/gomarkdown/markdown/ast"
)

// Parsing block-level elements.

const (
	captionTable  = "Table: "
	captionFigure = "Figure: "
	captionQuote  = "Quote: "
)

func (p *Parser) Block(data []byte) {
	if p.Opts.Flags&CommonMark != 0 {
		p.commonMarkBlocks(data)
		return
	}
	// this is called recursively: enforce a maximum depth
	if p.nesting >= p.maxNesting {
		return
	}
	p.nesting++
	prevBlockIndex := p.blockIndex
	p.blockIndex = blockIndex{data: data}
	defer func() {
		p.blockIndex = prevBlockIndex
	}()

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
		if consumed := p.runParserHook(data); consumed > 0 {
			data = data[consumed:]
			continue
		}

		if len(data) == 0 {
			continue
		}
		data = data[p.parseBlock(data):]
	}

	p.nesting--
}

// runParserHook runs Opts.ParserHook at the start of a block, adds the node
// it returns and parses that node's block data, and returns how much of
// data the hook consumed.
func (p *Parser) runParserHook(data []byte) int {
	if p.Opts.ParserHook == nil {
		return 0
	}
	node, blockData, consumed := p.Opts.ParserHook(data)
	if consumed <= 0 {
		return 0
	}
	if node != nil {
		p.AddBlock(node)
		if blockData != nil {
			p.Block(blockData)
			p.Finalize(node)
		}
	}
	return consumed
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
	if len(data) <= 4 || !bytes.HasPrefix(data, []byte("$$")) || data[2] == '$' {
		return 0
	}

	// find next $$
	var end int
	for end = 2; end+1 < len(data) && (data[end] != '$' || data[end+1] != '$'); end++ {
	}

	// no matching $$
	if end+1 == len(data) {
		return 0
	}

	p.AddBlock(&ast.MathBlock{Container: ast.Container{Literal: data[2:end]}})

	return end + 2
}
