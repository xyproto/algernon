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
		var n int
		if data, n = p.blockAttributes(data); n == 0 {
			n = firstConsumed(
				func() int {
					if p.extensions&Includes == 0 {
						return 0
					}
					return p.parseInclude(data)
				},
				// user supplied parser function
				func() int { return p.runParserHook(data) },
				func() int {
					if len(data) == 0 {
						return 0
					}
					return p.parseBlock(data)
				},
			)
		}
		data = data[n:]
	}

	p.nesting--
}

// runParserHook runs Opts.ParserHook at the start of a block, adds the node
// it returns and parses that node's block data, and returns how much of
// data the hook consumed.
// firstConsumed calls parsers in order and returns the size consumed by the
// first one that consumes input, or 0 if none does.
func firstConsumed(parsers ...func() int) int {
	for _, parse := range parsers {
		if n := parse(); n > 0 {
			return n
		}
	}
	return 0
}

// blockAttributes handles the attribute lists that may occur before a block
// or as a kramdown-style IAL after it. It returns data without a leading
// attribute list, and the size of an IAL it applied to the previous block.
func (p *Parser) blockAttributes(data []byte) ([]byte, int) {
	if p.extensions&Attributes == 0 {
		return data, 0
	}
	if n := p.applyAfterBlockAttribute(data); n > 0 {
		return data, n
	}
	return p.attribute(data), 0
}

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
		if field := ast.AttributeField(n); field != nil {
			*field = p.attr
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
