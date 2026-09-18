package parser

import (
	"bytes"

	"github.com/gomarkdown/markdown/ast"
)

func (p *Parser) inlineHTMLComment(data []byte) int {
	if !bytes.HasPrefix(data, []byte("<!--")) {
		return 0
	}
	i := bytes.Index(data[3:], commentEnd)
	if i < 0 {
		return 0
	}
	return i + 3 + len(commentEnd)
}

func stripMailto(link []byte) []byte {
	if bytes.HasPrefix(link, []byte("mailto://")) {
		return link[9:]
	}
	if bytes.HasPrefix(link, []byte("mailto:")) {
		return link[7:]
	}
	return link
}

// autolinkType specifies a kind of autolink that gets detected.
type autolinkType int

// These are the possible flag values for the autolink renderer.
const (
	notAutolink autolinkType = iota
	normalAutolink
	emailAutolink
)

func findGt(d []byte) int { return bytes.IndexByte(d, '>') }

var commentEnd = []byte("-->")

func findCommentEnd(d []byte) int { return bytes.Index(d, commentEnd) }

func leftAngle(p *Parser, data []byte, offset int) (int, ast.Node) {
	buf := data
	data = data[offset:]

	if p.extensions&Mmark != 0 {
		id, consumed := IsCallout(data)
		if consumed > 0 {
			node := &ast.Callout{}
			node.ID = id
			return consumed, node
		}
	}

	// A tag, an autolink and a comment all need a '>' after the cursor.
	// Without one the scans below would run to the end of the buffer for
	// every '<'.
	if p.nextGt.next(buf, offset, findGt) == len(buf) {
		return 0, nil
	}
	altype, end := tagLength(data)
	// A comment "<!--" ends at the first "-->" that does not reuse the
	// opener's dashes. Searching for it through the cache means an opener
	// that nothing closes does not scan to the end of the buffer again.
	if len(data) >= 5 && data[1] == '!' && data[2] == '-' && data[3] == '-' {
		if at := p.nextCommentEnd.next(buf, offset+3, findCommentEnd); at < len(buf) {
			end = at + len(commentEnd) - offset
		}
	}
	if end <= 2 {
		return end, nil
	}
	if altype == notAutolink {
		htmlTag := &ast.HTMLSpan{}
		htmlTag.Literal = data[:end]
		return end, htmlTag
	}

	link := unescapeBytes(data[1 : end-1])
	if len(link) == 0 {
		return end, nil
	}
	node := &ast.Link{
		Destination: link,
	}
	if altype == emailAutolink {
		node.Destination = append([]byte("mailto:"), link...)
	}
	ast.AppendChild(node, newTextNode(stripMailto(link)))
	return end, node
}

func linkEndsWithEntity(data []byte, linkEnd int) bool {
	start := bytes.LastIndexByte(data[:linkEnd], '&')
	return start >= 0 && findEntityEnd(data, start) == linkEnd
}

// hasPrefixCaseInsensitive is a custom implementation of
//
//	strings.HasPrefix(strings.ToLower(s), prefix)
//
// we rolled our own because ToLower pulls in a huge machinery of lowercasing
// anything from Unicode and that's very slow. Since this func will only be
// used on ASCII protocol prefixes, we can take shortcuts.
func hasPrefixCaseInsensitive(s, prefix []byte) bool {
	if len(s) < len(prefix) {
		return false
	}
	delta := byte('a' - 'A')
	for i, b := range prefix {
		if b != s[i] && b != s[i]+delta {
			return false
		}
	}
	return true
}

var protocolPrefixes = [][]byte{
	[]byte("http://"),
	[]byte("https://"),
	[]byte("ftp://"),
	[]byte("file://"),
	[]byte("mailto:"),
}

const shortestPrefix = 6 // len("ftp://"), the shortest of the above

func maybeAutoLink(p *Parser, data []byte, offset int) (int, ast.Node) {
	// quick check to rule out most false hits
	if p.InsideLink || len(data) < offset+shortestPrefix {
		return 0, nil
	}
	// Handler is registered on h/m/f. Almost all hits are words like "the"/"from".
	c1 := data[offset+1] | 0x20
	switch data[offset] | 0x20 {
	case 'h':
		if c1 != 't' { // http(s)
			return 0, nil
		}
	case 'm':
		if c1 != 'a' { // mailto
			return 0, nil
		}
	case 'f':
		if c1 != 't' && c1 != 'i' { // ftp, file
			return 0, nil
		}
	}
	endOfHead := offset + 8 // length of the longest supported prefix
	if endOfHead > len(data) {
		endOfHead = len(data)
	}
	for _, prefix := range protocolPrefixes {
		if hasPrefixCaseInsensitive(data[offset:endOfHead], prefix) {
			return autoLink(p, data, offset)
		}
	}
	return 0, nil
}

func autoLink(p *Parser, data []byte, offset int) (int, ast.Node) {
	// Now a more expensive check to see if we're not inside an anchor element
	anchorStart := p.angles.lastAt(data, offset, '<')
	if anchorStart < 0 {
		anchorStart = 0
	}
	offsetFromAnchor := offset - anchorStart

	anchorStr := anchorRe.Find(data[anchorStart:])
	if anchorStr != nil {
		anchorClose := &ast.HTMLSpan{}
		anchorClose.Literal = anchorStr[offsetFromAnchor:]
		return len(anchorStr) - offsetFromAnchor, anchorClose
	}

	// scan backward for a word boundary
	rewind := 0
	for offset-rewind > 0 && rewind <= 7 && IsLetter(data[offset-rewind-1]) {
		rewind++
	}
	if rewind > 6 { // longest supported protocol is "mailto" which has 6 letters
		return 0, nil
	}

	origData := data
	data = data[offset-rewind:]

	isSafeURL := p.IsSafeURLOverride
	if isSafeURL == nil {
		isSafeURL = IsSafeURL
	}
	if !isSafeURL(data) {
		return 0, nil
	}

	linkEnd := 0
	for linkEnd < len(data) && !isEndOfLink(data[linkEnd]) {
		linkEnd++
	}

	// Skip punctuation at the end of the link
	if (data[linkEnd-1] == '.' || data[linkEnd-1] == ',') && data[linkEnd-2] != '\\' {
		linkEnd--
	}

	// But don't skip semicolon if it's a part of escaped entity:
	if data[linkEnd-1] == ';' && data[linkEnd-2] != '\\' && !linkEndsWithEntity(data, linkEnd) {
		linkEnd--
	}

	// See if the link finishes with a punctuation sign that can be closed.
	var copen byte
	switch data[linkEnd-1] {
	case '"':
		copen = '"'
	case '\'':
		copen = '\''
	case ')':
		copen = '('
	case ']':
		copen = '['
	case '}':
		copen = '{'
	default:
		copen = 0
	}

	if copen != 0 {
		bufEnd := offset - rewind + linkEnd - 2

		openDelim := 1

		// Exclude the final closer only when its matching opener lies outside
		// the URL. A balanced pair within the URL remains part of it.
		for bufEnd >= 0 && origData[bufEnd] != '\n' && openDelim != 0 {
			if origData[bufEnd] == data[linkEnd-1] {
				openDelim++
			}

			if origData[bufEnd] == copen {
				openDelim--
			}

			bufEnd--
		}

		if openDelim == 0 {
			linkEnd--
		}
	}

	link := unescapeBytes(data[:linkEnd])
	if len(link) > 0 {
		node := &ast.Link{
			Destination: link,
		}
		ast.AppendChild(node, newTextNode(link))
		return linkEnd, node
	}

	return linkEnd, nil
}

func isEndOfLink(char byte) bool {
	return IsSpace(char) || char == '<'
}

// return the length of the given tag, or 0 is it's not valid
func tagLength(data []byte) (autolink autolinkType, end int) {
	if len(data) < 3 || data[0] != '<' {
		return
	}
	i := 1
	if data[i] == '/' {
		i++
	}
	if !IsAlnum(data[i]) {
		return
	}

	for i < len(data) && (IsAlnum(data[i]) || data[i] == '.' || data[i] == '+' || data[i] == '-') {
		i++
	}
	if i > 1 && i < len(data) && data[i] == '@' {
		if j := isMailtoAutoLink(data[i:]); j != 0 {
			return emailAutolink, i + j
		}
	}

	if i > 2 && i < len(data) && data[i] == ':' {
		autolink = normalAutolink
		i++
		contentStart := i
		for i < len(data) {
			if data[i] == '\\' {
				i += 2
			} else if data[i] == '>' || data[i] == '\'' || data[i] == '"' || IsSpace(data[i]) {
				break
			} else {
				i++
			}
		}
		if i >= len(data) {
			return autolink, 0
		}
		if i > contentStart && data[i] == '>' {
			return autolink, i + 1
		}
		autolink = notAutolink
	}
	if j := bytes.IndexByte(data[i:], '>'); j >= 0 {
		return autolink, i + j + 1
	}
	return autolink, 0
}

// look for the address part of a mail autolink and '>'
// this is less strict than the original markdown e-mail address matching
func isMailtoAutoLink(data []byte) int {
	nb := 0

	// address is assumed to be: [-@._a-zA-Z0-9]+ with exactly one '@'
	for i, c := range data {
		if IsAlnum(c) {
			continue
		}

		switch c {
		case '@':
			nb++

		case '-', '.', '_':
			// valid address punctuation

		case '>':
			if nb == 1 {
				return i + 1
			}
			return 0
		default:
			return 0
		}
	}

	return 0
}

// look for the next emph char, skipping other constructs
