package parser

import (
	"bytes"

	"github.com/gomarkdown/markdown/ast"
)

func (p *Parser) renderParagraph(data []byte) {
	if len(data) == 0 {
		return
	}

	// trim leading spaces
	beg := skipChar(data, 0, ' ')

	end := len(data)
	// trim trailing newline
	if data[len(data)-1] == '\n' {
		end--
	}

	// trim trailing spaces
	for end > beg && data[end-1] == ' ' {
		end--
	}
	para := &ast.Paragraph{}
	para.Content = data[beg:end]
	p.AddBlock(para)
}

func (p *Parser) interruptsParagraph(data []byte) bool {
	if p.extensions&LaxHTMLBlocks != 0 && data[0] == '<' && p.html(data, false) > 0 {
		return true
	}
	if p.isPrefixHeading(data) || p.isPrefixSpecialHeading(data) ||
		isHRule(data) || quotePrefix(data) > 0 {
		return true
	}
	if p.extensions&FencedCode != 0 && p.fencedCodeBlock(data, false) > 0 {
		return true
	}
	if p.extensions&Mmark != 0 && p.figureBlock(data, false) > 0 {
		return true
	}
	if p.extensions&Tables != 0 {
		size, _, _ := p.tableHeader(data, false)
		return size > 0
	}
	return false
}

func (p *Parser) paragraph(data []byte) int {
	// prev: index of 1st char of previous line
	// line: index of 1st char of current line
	// i: index of cursor/end of current line
	var prev, line, i int
	tabSize := tabSizeDefault
	if p.extensions&TabSizeEight != 0 {
		tabSize = tabSizeDouble
	}
	// keep going until we find something to mark the end of the paragraph
	for i < len(data) {
		// mark the beginning of the current line
		prev = line
		current := data[i:]
		line = i

		// did we find a reference or a footnote? If so, end a paragraph
		// preceding it and report that we have consumed up to the end of that
		// reference:
		if refEnd := isReference(p, current, tabSize); refEnd > 0 {
			p.renderParagraph(data[:i])
			if p.pendingRefDef != nil {
				p.AddBlock(p.pendingRefDef)
				p.pendingRefDef = nil
			}
			return i + refEnd
		}

		if p.extensions&Attributes != 0 && isAfterBlockIAL(current) {
			p.renderParagraph(data[:i])
			return i
		}

		// did we find a blank line marking the end of the paragraph?
		if n := IsEmpty(current); n > 0 {
			// did this blank line followed by a definition list item?
			if p.extensions&DefinitionLists != 0 {
				if i < len(data)-1 && data[i+1] == ':' {
					listLen := p.list(data[prev:], ast.ListTypeDefinition, 0, '.')
					if listLen > 0 {
						return prev + listLen
					}
				}
			}

			p.renderParagraph(data[:i])
			return i + n
		}

		// an underline under some text marks a heading, so our paragraph ended on prev line
		if i > 0 {
			if level := isUnderlinedHeading(current); level > 0 {
				// render the paragraph
				p.renderParagraph(data[:prev])

				// ignore leading and trailing whitespace
				eol := i - 1
				for prev < eol && data[prev] == ' ' {
					prev++
				}
				for eol > prev && data[eol-1] == ' ' {
					eol--
				}

				block := &ast.Heading{
					Level: level,
				}
				p.setHeadingID(block, "", data[prev:eol])

				block.Content = data[prev:eol]
				p.AddBlock(block)

				// find the end of the underline
				return skipUntilChar(data, i, '\n')
			}
		}

		if p.interruptsParagraph(current) {
			p.renderParagraph(data[:i])
			return i
		}

		// if there's a definition list item, prev line is a definition term
		if p.extensions&DefinitionLists != 0 && dliPrefix(current) != 0 {
			return prev + p.list(data[prev:], ast.ListTypeDefinition, 0, '.')
		}

		// if there's a list after this, paragraph is over
		if p.extensions&NoEmptyLineBeforeBlock != 0 &&
			(uliPrefix(current) != 0 || oliPrefix(current) != 0 || codePrefix(current) != 0) {
			p.renderParagraph(data[:i])
			return i
		}

		// otherwise, scan to the beginning of the next line
		nl := bytes.IndexByte(data[i:], '\n')
		if nl >= 0 {
			i += nl + 1
		} else {
			i += len(data[i:])
		}
	}

	p.renderParagraph(data[:i])
	return i
}
