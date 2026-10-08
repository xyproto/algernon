package parser

import (
	"bytes"

	"github.com/gomarkdown/markdown/ast"
)

// quotePrefix returns the blockquote prefix length.
func quotePrefix(data []byte) int {
	return prefixedBlock(data, ">")
}

// parse a blockquote fragment
func (p *Parser) quote(data []byte) int {
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
		if pre := quotePrefix(data[beg:]); pre > 0 {
			// skip the prefix
			contentBeg += pre
		} else if fenceMarker != "" {
			// Lines inside a quoted fenced code block may omit the quote
			// prefix. Keep them in the quote until the fence closes.
		} else if terminatesPrefixedBlock(data, beg, end, quotePrefix) {
			break
		}
		// this line is part of the blockquote
		raw.Write(data[contentBeg:end])
		if p.extensions&FencedCode != 0 {
			if _, marker := isFenceLine(data[contentBeg:end], nil, fenceMarker); marker != "" {
				if fenceMarker == "" {
					fenceMarker = marker
				} else {
					fenceMarker = ""
				}
			}
		}
		beg = end
	}

	if p.extensions&Mmark != 0 {
		if captionContent, id, consumed := parseCaption(data[end:], []byte(captionQuote)); consumed > 0 {
			figure := &ast.CaptionFigure{HeadingID: id}
			caption := &ast.Caption{}
			p.Inline(caption, captionContent)

			p.AddBlock(figure)
			block := &ast.BlockQuote{Container: ast.Container{Attribute: figure.Attribute}}
			p.addChild(block)
			p.Block(raw.Bytes())
			p.Finalize(block)

			p.addChild(caption)
			p.Finalize(figure)
			return end + consumed
		}
	}

	block := p.AddBlock(&ast.BlockQuote{})
	p.Block(raw.Bytes())
	p.Finalize(block)

	return end
}
