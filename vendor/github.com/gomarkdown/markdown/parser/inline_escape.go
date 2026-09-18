package parser

import (
	"bytes"
	"html"
	"strconv"

	"github.com/gomarkdown/markdown/ast"
)

// '\\' backslash escape
var EscapeChars = []byte("\\`*_{}[]()#+-.!:|&<>~^$")

func escape(p *Parser, data []byte, offset int) (int, ast.Node) {
	data = data[offset:]

	// A backslash with nothing after it escapes nothing and is literal text.
	if len(data) <= 1 {
		return 0, nil
	}

	if p.extensions&NonBlockingSpace != 0 && data[1] == ' ' {
		return 2, &ast.NonBlockingSpace{}
	}

	if p.extensions&BackslashLineBreak != 0 && data[1] == '\n' {
		return 2, &ast.Hardbreak{}
	}

	if bytes.IndexByte(EscapeChars, data[1]) < 0 {
		return 0, nil
	}

	return 2, newTextNode(data[1:2])
}

func unescapeText(ob *bytes.Buffer, src []byte) {
	i := 0
	for i < len(src) {
		org := i
		for i < len(src) && src[i] != '\\' {
			i++
		}

		if i > org {
			ob.Write(src[org:i])
		}

		if i+1 >= len(src) {
			break
		}

		ob.WriteByte(src[i+1])
		i += 2
	}
}

func unescapeBytes(src []byte) []byte {
	var out bytes.Buffer
	unescapeText(&out, src)
	return out.Bytes()
}

// '&' escaped when it doesn't belong to an entity
// valid entities are assumed to be anything matching &#?[A-Za-z0-9]+;
func entity(p *Parser, data []byte, offset int) (int, ast.Node) {
	data = data[offset:]

	end := skipCharN(data, 1, '#', 1)
	end = skipAlnum(data, end)

	if end < len(data) && data[end] == ';' {
		end++ // real entity
	} else {
		return 0, nil // lone '&'
	}

	ent := data[:end]
	if ent[1] != '#' {
		// A named entity becomes the character it names, so the renderer
		// escapes the result rather than the ampersand of the reference
		// (&amp; round-trips as &amp; instead of &amp;amp;). A name the
		// HTML5 table does not know stays literal text.
		if s := html.UnescapeString(string(ent)); s != string(ent) {
			return end, newTextNode([]byte(s))
		}
		return end, newTextNode(ent)
	}
	if len(ent) < 4 {
		return end, newTextNode(ent)
	}

	codepoint := uint64(0)
	var err error
	if ent[2] == 'x' || ent[2] == 'X' { // hexadecimal
		codepoint, err = strconv.ParseUint(string(ent[3:len(ent)-1]), 16, 64)
	} else {
		codepoint, err = strconv.ParseUint(string(ent[2:len(ent)-1]), 10, 64)
	}
	if err == nil { // only if conversion was valid return here.
		// Replace invalid codepoints with U+FFFD per CommonMark spec section
		// 6.2. Check before converting to rune, which would wrap large values.
		r := '\uFFFD'
		if codepoint != 0 && (codepoint < 0xD800 || codepoint > 0xDFFF) && codepoint <= 0x10FFFF {
			r = rune(codepoint)
		}
		return end, newTextNode([]byte(string(r)))
	}

	return end, newTextNode(ent)
}
