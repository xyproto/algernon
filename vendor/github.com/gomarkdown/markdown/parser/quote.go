package parser

import (
	"bytes"

	"github.com/gomarkdown/markdown/ast"
)

// quotePrefix returns the blockquote prefix length.
func quotePrefix(data []byte) int {
	return prefixedBlock(data, ">")
}

// prefixedBlockContent returns the content of a block whose lines start with
// prefix, such as a quote, without the prefix, and the number of bytes the
// block takes in data. Lines inside a fenced code block may omit the prefix.
func (p *Parser) prefixedBlockContent(data []byte, prefix string) ([]byte, int) {
	hasPrefix := func(data []byte) int { return prefixedBlock(data, prefix) }
	var raw bytes.Buffer
	beg, end := 0, 0
	fenceMarker := ""
	for beg < len(data) {
		end = beg
		for end < len(data) && data[end] != '\n' {
			end++
		}
		end = skipCharN(data, end, '\n', 1)
		contentBeg := beg
		if pre := hasPrefix(data[beg:]); pre > 0 {
			contentBeg += pre
		} else if fenceMarker == "" && terminatesPrefixedBlock(data, beg, end, hasPrefix) {
			break
		}
		raw.Write(data[contentBeg:end])
		if p.extensions&FencedCode != 0 {
			if _, marker := isFenceLine(data[contentBeg:end], nil, fenceMarker); marker != "" {
				if fenceMarker != "" {
					fenceMarker = ""
				} else if p.blockIndex.hasPrefixedFenceCloser(data, end, marker, prefix) {
					// an unclosed fence is not code, so it doesn't keep lines
					// without the prefix in the block
					fenceMarker = marker
				}
			}
		}
		beg = end
	}
	return raw.Bytes(), end
}

// parse a blockquote fragment
func (p *Parser) quote(data []byte) int {
	raw, end := p.prefixedBlockContent(data, ">")

	if p.extensions&Mmark != 0 {
		if captionContent, id, consumed := parseCaption(data[end:], []byte(captionQuote)); consumed > 0 {
			figure := &ast.CaptionFigure{HeadingID: id}
			caption := &ast.Caption{}
			p.Inline(caption, captionContent)

			p.AddBlock(figure)
			block := &ast.BlockQuote{Container: ast.Container{Attribute: figure.Attribute}}
			p.addChild(block)
			p.Block(raw)
			p.Finalize(block)

			p.addChild(caption)
			p.Finalize(figure)
			return end + consumed
		}
	}

	block := p.AddBlock(&ast.BlockQuote{})
	p.Block(raw)
	p.Finalize(block)

	return end
}
