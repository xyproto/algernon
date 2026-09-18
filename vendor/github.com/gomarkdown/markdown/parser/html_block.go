package parser

import (
	"bytes"

	"github.com/gomarkdown/markdown/ast"
)

var (
	// blockTags is a set of tags that are recognized as HTML block tags.
	// Any of these can be included in markdown text without special escaping.
	blockTags = stringSet(
		"blockquote", "del", "dd", "div", "dl", "dt", "fieldset", "form",
		"h1", "h2", "h3", "h4", "h5", "h6",
		// Kept inline for compatibility with existing simple HTML parsing.
		// "hr",
		"iframe", "ins", "li", "math", "noscript", "ol", "pre", "p",
		"script", "style", "table", "ul",

		// HTML5
		"address", "article", "aside", "canvas", "details", "dialog",
		"figcaption", "figure", "footer", "header", "hgroup", "main", "nav",
		"output", "progress", "section", "svg", "video",
	)

	markdownHTMLBlockTags = stringSet("details", "div")

	// Tags whose interiors are walked as nested HTML when MarkdownInHTML is set.
	htmlStructureTags = stringSet("table", "thead", "tbody", "tfoot", "tr")

	// Extra block tags recognized only with MarkdownInHTML.
	markdownInHTMLTags = stringSet("table", "thead", "tbody", "tfoot", "tr", "td", "th")
)

func stringSet(values ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func inStringSet(set map[string]struct{}, value string) bool {
	_, ok := set[value]
	return ok
}

func (p *Parser) html(data []byte, doRender bool) int {
	// identify the opening tag
	if data[0] != '<' {
		return 0
	}
	curtag, tagfound := p.htmlFindTag(data[1:])

	// handle special cases
	if !tagfound {
		// check for an HTML comment
		if size := p.htmlComment(data, doRender); size > 0 {
			return size
		}

		// check for an <hr> tag
		if size := p.htmlHr(data, doRender); size > 0 {
			return size
		}

		// no special case recognized
		return 0
	}

	if p.markdownHTMLTag(curtag) {
		if p.extensions&MarkdownInHTML != 0 && isHTMLStructureTag(curtag) {
			if size := p.htmlStructuredBlock(data, curtag, doRender); size > 0 {
				return size
			}
		}
		if size := p.htmlMarkdownBlock(data, curtag, doRender); size > 0 {
			return size
		}
	}

	// Following Markdown.pl, ins and del cannot form these blocks.
	if curtag == "ins" || curtag == "del" {
		return 0
	}
	_, consumed := p.findHTMLCloseTag(data, curtag, 1, false)
	if consumed == 0 {
		return 0
	}

	if doRender {
		end := backChar(data, consumed, '\n')
		p.addHTMLBlock(data[:end])
	}
	return consumed
}

func (p *Parser) markdownHTMLTag(tag string) bool {
	return inStringSet(markdownHTMLBlockTags, tag) ||
		p.extensions&MarkdownInHTML != 0 && inStringSet(markdownInHTMLTags, tag)
}

func isHTMLStructureTag(tag string) bool {
	return inStringSet(htmlStructureTags, tag)
}

func (p *Parser) htmlMarkdownBlock(data []byte, tag string, doRender bool) int {
	openEnd := bytes.IndexByte(data, '>')
	if openEnd < 0 {
		return 0
	}
	openEnd++

	loose := p.extensions&MarkdownInHTML != 0
	closeStart, consumed := p.findHTMLCloseTag(data, tag, openEnd, loose)
	if consumed == 0 {
		return 0
	}
	if !hasBlankLineAfter(data, openEnd) || !hasBlankLineBefore(data, closeStart, loose) {
		return 0
	}

	closeEnd := closeStart + len("</"+tag+">")
	if doRender {
		p.addHTMLBlock(bytes.TrimRight(data[:openEnd], "\n"))

		inner := bytes.TrimPrefix(data[openEnd:closeStart], []byte("\n"))
		if len(inner) > 0 {
			p.Block(inner)
		}

		p.addHTMLBlock(bytes.TrimRight(data[closeStart:closeEnd], "\n"))
	}

	return consumed
}

func hasBlankLineAfter(data []byte, pos int) bool {
	first := IsEmpty(data[pos:])
	if first == 0 {
		return false
	}
	return IsEmpty(data[pos+first:]) > 0
}

func hasBlankLineBefore(data []byte, pos int, skipIndent bool) bool {
	if skipIndent {
		for pos > 0 && (data[pos-1] == ' ' || data[pos-1] == '\t') {
			pos--
		}
	}
	if pos < 2 || data[pos-1] != '\n' {
		return false
	}
	lineEnd := pos - 1
	lineStart := bytes.LastIndexByte(data[:lineEnd], '\n') + 1
	return len(bytes.Trim(data[lineStart:lineEnd], " \t")) == 0
}

func (p *Parser) findHTMLCloseTag(data []byte, tag string, start int, loose bool) (closeStart int, consumed int) {
	for i := start; i < len(data); i++ {
		for i < len(data) && !(data[i-1] == '<' && data[i] == '/') {
			i++
		}
		if i+2+len(tag) > len(data) {
			return 0, 0
		}

		var j int
		if loose {
			j = htmlFindEndLoose(tag, data[i-1:])
		} else {
			j = p.htmlFindEnd(tag, data[i-1:])
		}
		if j > 0 {
			return i - 1, i + j - 1
		}
	}
	return 0, 0
}

func htmlFindEndLoose(tag string, data []byte) int {
	closetag := []byte("</" + tag + ">")
	if !bytes.HasPrefix(data, closetag) {
		return 0
	}
	i := len(closetag)
	for i < len(data) && data[i] != '\n' {
		if data[i] != ' ' && data[i] != '\t' {
			return 0
		}
		i++
	}
	if i < len(data) && data[i] == '\n' {
		i++
	}
	return i
}

func (p *Parser) htmlStructuredBlock(data []byte, tag string, doRender bool) int {
	openEnd := bytes.IndexByte(data, '>')
	if openEnd < 0 {
		return 0
	}
	openEnd++

	closeStart, consumed := p.findHTMLCloseTag(data, tag, openEnd, true)
	if consumed == 0 {
		return 0
	}

	closeEnd := closeStart + len("</"+tag+">")
	if doRender {
		p.addHTMLBlock(bytes.TrimRight(data[:openEnd], "\n"))
		p.parseHTMLInterior(data[openEnd:closeStart])
		p.addHTMLBlock(bytes.TrimRight(data[closeStart:closeEnd], "\n"))
	}
	return consumed
}

func (p *Parser) parseHTMLInterior(data []byte) {
	for len(data) > 0 {
		if data[0] == '\n' {
			data = data[1:]
			continue
		}
		i := skipHSpace(data, 0)
		if i < len(data) && data[i] == '<' {
			if n := p.html(data[i:], true); n > 0 {
				data = data[i+n:]
				continue
			}
		}
		nl := bytes.IndexByte(data, '\n')
		if nl < 0 {
			if len(bytes.TrimSpace(data)) > 0 {
				p.addHTMLBlock(bytes.TrimRight(data, "\n"))
			}
			return
		}
		chunk := data[:nl]
		if len(bytes.TrimSpace(chunk)) > 0 {
			p.addHTMLBlock(chunk)
		}
		data = data[nl+1:]
	}
}

func (p *Parser) addHTMLBlock(literal []byte) {
	p.AddBlock(&ast.HTMLBlock{Leaf: ast.Leaf{Literal: literal}})
}

// HTML comment, lax form
func (p *Parser) htmlComment(data []byte, doRender bool) int {
	i := p.inlineHTMLComment(data)
	// needs to end with a blank line
	if j := IsEmpty(data[i:]); j > 0 {
		size := i + j
		if doRender {
			// trim trailing newlines
			end := backChar(data, size, '\n')
			p.addHTMLBlock(data[:end])
		}
		return size
	}
	return 0
}

// HR, which is the only self-closing block tag considered
func (p *Parser) htmlHr(data []byte, doRender bool) int {
	if len(data) < 4 {
		return 0
	}
	if (data[1] != 'h' && data[1] != 'H') || (data[2] != 'r' && data[2] != 'R') {
		return 0
	}
	if data[3] != ' ' && data[3] != '/' && data[3] != '>' {
		// not an <hr> tag after all; at least not a valid one
		return 0
	}
	i := 3
	for i < len(data) && data[i] != '>' && data[i] != '\n' {
		i++
	}
	if i < len(data) && data[i] == '>' {
		i++
		if j := IsEmpty(data[i:]); j > 0 {
			size := i + j
			if doRender {
				// trim newlines
				end := backChar(data, size, '\n')
				p.addHTMLBlock(data[:end])
			}
			return size
		}
	}
	return 0
}

func (p *Parser) htmlFindTag(data []byte) (string, bool) {
	i := skipAlnum(data, 0)
	key := string(data[:i])
	if inStringSet(blockTags, key) ||
		p.extensions&MarkdownInHTML != 0 && inStringSet(markdownInHTMLTags, key) {
		return key, true
	}
	return "", false
}

func (p *Parser) htmlFindEnd(tag string, data []byte) int {
	// assume data[0] == '<' && data[1] == '/' already tested
	if tag == "hr" {
		return 2
	}
	// check if tag is a match
	closetag := []byte("</" + tag + ">")
	if !bytes.HasPrefix(data, closetag) {
		return 0
	}
	i := len(closetag)

	// check that the rest of the line is blank
	skip := IsEmpty(data[i:])
	if skip == 0 {
		return 0
	}
	i += skip
	if i >= len(data) {
		return i
	}

	if p.extensions&LaxHTMLBlocks != 0 {
		return i
	}
	if skip = IsEmpty(data[i:]); skip == 0 {
		// following line must be blank
		return 0
	}

	return i + skip
}
