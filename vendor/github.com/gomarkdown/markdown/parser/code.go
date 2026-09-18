package parser

import (
	"bytes"
	"html"

	"github.com/gomarkdown/markdown/ast"
)

func isFenceLine(data []byte, syntax *string, oldmarker string) (end int, marker string) {
	i := skipCharN(data, 0, ' ', 3)
	if i >= len(data) || data[i] != '~' && data[i] != '`' {
		return 0, ""
	}

	c := data[i]
	start := i
	i = skipChar(data, i, c)
	size := i - start
	if size < 3 {
		return 0, ""
	}
	marker = string(data[start:i])

	// if this is the end marker, it must match the beginning marker
	if oldmarker != "" && marker != oldmarker {
		return 0, ""
	}

	// if just read the beginning marker, read the syntax
	if oldmarker == "" {
		i = skipChar(data, i, ' ')
		if i >= len(data) {
			return i, marker
		}

		syntaxStart, syntaxLen := syntaxRange(data, &i)
		if syntaxStart == 0 && syntaxLen == 0 {
			return 0, ""
		}

		// caller wants the syntax
		if syntax != nil {
			*syntax = string(data[syntaxStart : syntaxStart+syntaxLen])
		}
	}

	i = skipChar(data, i, ' ')
	if i >= len(data) {
		return i, marker
	}
	if data[i] != '\n' {
		return 0, ""
	}
	return i + 1, marker // Take newline into account.
}

func syntaxRange(data []byte, iout *int) (int, int) {
	n := len(data)
	syn := 0
	i := *iout
	syntaxStart := i
	if data[i] == '{' {
		i++
		syntaxStart++

		for i < n && data[i] != '}' && data[i] != '\n' {
			syn++
			i++
		}

		if i >= n || data[i] != '}' {
			return 0, 0
		}

		// strip all whitespace at the beginning and the end
		// of the {} block
		for syn > 0 && IsSpace(data[syntaxStart]) {
			syntaxStart++
			syn--
		}

		for syn > 0 && IsSpace(data[syntaxStart+syn-1]) {
			syn--
		}

		i++
	} else {
		for i < n && data[i] != '\n' {
			syn++
			i++
		}
	}

	*iout = i
	return syntaxStart, syn
}

// fencedCodeBlock returns the end index if data contains a fenced code block at the beginning,
// or 0 otherwise. It writes to out if doRender is true, otherwise it has no side effects.
// If doRender is true, a final newline is mandatory to recognize the fenced code block.
func (p *Parser) fencedCodeBlock(data []byte, doRender bool) int {
	var syntax string
	beg, marker := isFenceLine(data, &syntax, "")
	if beg == 0 || beg >= len(data) {
		return 0
	}

	var work bytes.Buffer

	for {
		// check for the end of the code block
		fenceEnd, _ := isFenceLine(data[beg:], nil, marker)
		if fenceEnd != 0 {
			beg += fenceEnd
			break
		}

		// copy the current line
		end := skipUntilChar(data, beg, '\n') + 1

		// did we reach the end of the buffer without a closing marker?
		if end >= len(data) {
			return 0
		}

		// verbatim copy to the working buffer
		work.Write(data[beg:end])
		beg = end
	}

	if !doRender {
		return beg
	}
	codeBlock := &ast.CodeBlock{
		Leaf:     ast.Leaf{Literal: work.Bytes()},
		IsFenced: true,
		Info:     unescapeString([]byte(syntax)),
	}

	if p.extensions&Mmark != 0 {
		if captionContent, id, consumed := parseCaption(data[beg:], []byte(captionFigure)); consumed > 0 {
			figure := &ast.CaptionFigure{HeadingID: id}
			caption := &ast.Caption{}
			p.Inline(caption, captionContent)

			p.AddBlock(figure)
			codeBlock.Attribute = figure.Attribute
			p.addChild(codeBlock)
			p.addChild(caption)
			p.Finalize(figure)
			return beg + consumed
		}
	}

	p.AddBlock(codeBlock)
	return beg
}

func unescapeString(str []byte) []byte {
	var out []byte
	for i := 0; i < len(str); i++ {
		switch str[i] {
		case '\\':
			if i+1 < len(str) && isEscapable(str[i+1]) {
				if out == nil {
					out = make([]byte, 0, len(str))
					out = append(out, str[:i]...)
				}
				out = append(out, str[i+1])
				i++
				continue
			}
		case '&':
			entityEnd := findEntityEnd(str, i)
			if entityEnd > i {
				replacement := html.UnescapeString(string(str[i:entityEnd]))
				if replacement != string(str[i:entityEnd]) {
					if out == nil {
						out = make([]byte, 0, len(str))
						out = append(out, str[:i]...)
					}
					out = append(out, replacement...)
					i = entityEnd - 1
					continue
				}
			}
		}
		if out != nil {
			out = append(out, str[i])
		}
	}
	if out != nil {
		return out
	}
	return str
}

func isEscapable(c byte) bool {
	switch c {
	case '!', '"', '#', '$', '%', '&', '\'', '(', ')', '*', '+', ',', '.', '/', ':',
		';', '<', '=', '>', '?', '@', '[', '\\', ']', '^', '_', '`', '{', '|', '}', '~', '-':
		return true
	default:
		return false
	}
}

func findEntityEnd(str []byte, start int) int {
	i := start + 1
	if i >= len(str) {
		return 0
	}
	if str[i] == '#' {
		i++
		if i >= len(str) {
			return 0
		}
		hex := str[i] == 'x' || str[i] == 'X'
		if hex {
			i++
		}
		digitStart := i
		for i < len(str) && i-digitStart < 8 && (str[i] >= '0' && str[i] <= '9' || hex && isHexDigit(str[i])) {
			i++
		}
		if i == digitStart || i >= len(str) || str[i] != ';' {
			return 0
		}
		return i + 1
	}
	if !IsLetter(str[i]) {
		return 0
	}
	i++
	for i < len(str) && i-start <= 32 && IsAlnum(str[i]) {
		i++
	}
	if i >= len(str) || str[i] != ';' {
		return 0
	}
	return i + 1
}

func isHexDigit(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

func codePrefix(data []byte) int {
	n := len(data)
	if n >= 1 && data[0] == '\t' {
		return 1
	}
	if n >= 4 && data[3] == ' ' && data[2] == ' ' && data[1] == ' ' && data[0] == ' ' {
		return 4
	}
	return 0
}

func (p *Parser) code(data []byte) int {
	var work bytes.Buffer

	i := 0
	for i < len(data) {
		beg := i

		i = skipUntilChar(data, i, '\n')
		i = skipCharN(data, i, '\n', 1)

		blankline := IsEmpty(data[beg:i]) > 0
		if pre := codePrefix(data[beg:i]); pre > 0 {
			beg += pre
		} else if !blankline {
			// non-empty, non-prefixed line breaks the pre
			i = beg
			break
		}

		// verbatim copy to the working buffer
		if blankline {
			work.WriteByte('\n')
		} else {
			work.Write(data[beg:i])
		}
	}

	// trim all the \n off the end of work
	workbytes := work.Bytes()

	eol := backChar(workbytes, len(workbytes), '\n')

	if eol != len(workbytes) {
		work.Truncate(eol)
	}

	work.WriteByte('\n')

	codeBlock := &ast.CodeBlock{Leaf: ast.Leaf{Literal: work.Bytes()}}
	p.AddBlock(codeBlock)

	return i
}
