package parser

import (
	"bytes"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/internal/textutil"
)

// Tokens stay in a linked sequence until delimiters have been resolved. This
// lets emphasis and links group their children without repeatedly moving the
// remaining nodes in an AST child slice.
type commonMarkToken struct {
	node       ast.Node
	prev, next *commonMarkToken
}

type commonMarkDelimiter struct {
	token       *commonMarkToken
	char        byte
	length      int
	open, close bool
	prev, next  *commonMarkDelimiter
}

type commonMarkBracket struct {
	token     *commonMarkToken
	delimiter *commonMarkDelimiter
	start     int
	image     bool
	prev      *commonMarkBracket
}

func appendCommonMarkText(parent ast.Node, literal []byte) {
	if len(literal) == 0 {
		return
	}
	if previous, ok := ast.GetLastChild(parent).(*ast.Text); ok {
		previous.Literal = append(previous.Literal, literal...)
	} else {
		ast.AppendChild(parent, newTextNode(bytes.Clone(literal)))
	}
}

func commonMarkAppendTokens(parent ast.Node, first, end *commonMarkToken) {
	for token := first; token != end; token = token.next {
		if text, ok := token.node.(*ast.Text); ok {
			appendCommonMarkText(parent, text.Literal)
		} else {
			ast.AppendChild(parent, token.node)
		}
	}
}

func commonMarkRemoveToken(token *commonMarkToken) {
	if token.prev != nil {
		token.prev.next = token.next
	}
	if token.next != nil {
		token.next.prev = token.prev
	}
}

func commonMarkRemoveDelimiter(d *commonMarkDelimiter, last **commonMarkDelimiter) {
	if d.prev != nil {
		d.prev.next = d.next
	}
	if d.next != nil {
		d.next.prev = d.prev
	} else {
		*last = d.prev
	}
}

func commonMarkEmphasis(bottom *commonMarkDelimiter, last **commonMarkDelimiter) {
	var closer *commonMarkDelimiter
	if bottom != nil {
		closer = bottom.next
	} else {
		closer = *last
		for closer != nil && closer.prev != nil {
			closer = closer.prev
		}
	}
	// Remember failed searches separately for each delimiter and rule-of-three
	// class, so unmatched runs do not repeatedly search the same prefix.
	var floors [2][3][2]*commonMarkDelimiter
	for closer != nil {
		if !closer.close {
			closer = closer.next
			continue
		}
		kind := 0
		if closer.char == '_' {
			kind = 1
		}
		both := 0
		if closer.open {
			both = 1
		}
		floor := floors[kind][closer.length%3][both]
		if floor == nil {
			floor = bottom
		}
		opener := closer.prev
		for opener != nil && opener != floor && opener != bottom {
			if opener.char == closer.char && opener.open &&
				!((opener.close || closer.open) && (opener.length+closer.length)%3 == 0 && (opener.length%3 != 0 || closer.length%3 != 0)) {
				break
			}
			opener = opener.prev
		}
		if opener == nil || opener == floor || opener == bottom {
			floors[kind][closer.length%3][both] = closer.prev
			next := closer.next
			if !closer.open {
				commonMarkRemoveDelimiter(closer, last)
			}
			closer = next
			continue
		}
		use := 1
		var node ast.Node = &ast.Emph{}
		if opener.length >= 2 && closer.length >= 2 {
			use = 2
			node = &ast.Strong{}
		}
		commonMarkAppendTokens(node, opener.token.next, closer.token)
		token := &commonMarkToken{node: node, prev: opener.token, next: closer.token}
		opener.token.next, closer.token.prev = token, token
		opener.length -= use
		closer.length -= use
		opener.token.node.(*ast.Text).Literal = opener.token.node.(*ast.Text).Literal[:opener.length]
		closer.token.node.(*ast.Text).Literal = closer.token.node.(*ast.Text).Literal[use:]
		for d := opener.next; d != closer; {
			next := d.next
			commonMarkRemoveDelimiter(d, last)
			d = next
		}
		if opener.length == 0 {
			commonMarkRemoveToken(opener.token)
			commonMarkRemoveDelimiter(opener, last)
		}
		if closer.length == 0 {
			next := closer.next
			commonMarkRemoveToken(closer.token)
			commonMarkRemoveDelimiter(closer, last)
			closer = next
		}
	}
	for *last != bottom {
		commonMarkRemoveDelimiter(*last, last)
	}
}

func commonMarkFlanking(data []byte, start, end int) (left, right bool) {
	before, after := rune(' '), rune(' ')
	if start > 0 {
		before, _ = utf8.DecodeLastRune(data[:start])
	}
	if end < len(data) {
		after, _ = utf8.DecodeRune(data[end:])
	}
	beforeSpace, afterSpace := commonMarkUnicodeSpace(before), commonMarkUnicodeSpace(after)
	beforePunct := unicode.IsPunct(before) || unicode.IsSymbol(before)
	afterPunct := unicode.IsPunct(after) || unicode.IsSymbol(after)
	left = !afterSpace && (!afterPunct || beforeSpace || beforePunct)
	right = !beforeSpace && (!beforePunct || afterSpace || afterPunct)
	if data[start] == '_' {
		return left && (!right || beforePunct), right && (!left || afterPunct)
	}
	return
}

func (p *Parser) commonMarkInline(parent ast.Node, data []byte) {
	data = bytes.TrimRight(data, " \t\n")
	previousStops := p.linkStops
	p.linkStops = byteIndex{data: data, chars: "\"')(", honorEscapes: true}
	defer func() { p.linkStops = previousStops }()
	root := &commonMarkToken{}
	tail := root
	appendNode := func(node ast.Node) *commonMarkToken {
		token := &commonMarkToken{node: node, prev: tail}
		tail.next = token
		tail = token
		return token
	}
	var lastDelimiter *commonMarkDelimiter
	var bracket *commonMarkBracket
	lastLinkStart := -1
	var angleClosers [4]nextMatchCache
	// Index complete backtick runs once. A closer must have exactly the same
	// length as its opener, and unmatched runs remain literal as a whole.
	backticks := map[int][]int{}
	for i := 0; i < len(data); {
		if data[i] != '`' {
			i++
			continue
		}
		end := skipChar(data, i, '`')
		backticks[end-i] = append(backticks[end-i], i)
		i = end
	}
	for i := 0; i < len(data); {
		c := data[i]
		switch c {
		case '*', '_':
			end := skipChar(data, i, c)
			token := appendNode(newTextNode(data[i:end]))
			open, close := commonMarkFlanking(data, i, end)
			d := &commonMarkDelimiter{token: token, char: c, length: end - i, open: open, close: close, prev: lastDelimiter}
			if lastDelimiter != nil {
				lastDelimiter.next = d
			}
			lastDelimiter = d
			i = end
			continue
		case '[', '!':
			image := c == '!'
			if !image || i+1 < len(data) && data[i+1] == '[' {
				width := 1
				if image {
					width = 2
				}
				token := appendNode(newTextNode(data[i : i+width]))
				bracket = &commonMarkBracket{token: token, delimiter: lastDelimiter, start: i + width, image: image, prev: bracket}
				i += width
				continue
			}
		case ']':
			if bracket != nil {
				b := bracket
				bracket = b.prev
				if b.image || b.start > lastLinkStart {
					end, destination, title, ok := p.commonMarkLinkTarget(data, i+1, data[b.start:i])
					if ok {
						commonMarkEmphasis(b.delimiter, &lastDelimiter)
						var node ast.Node = &ast.Link{Destination: destination, Title: title}
						if b.image {
							node = &ast.Image{Destination: destination, Title: title}
						}
						commonMarkAppendTokens(node, b.token.next, nil)
						b.token.node = node
						b.token.next = nil
						tail = b.token
						if !b.image {
							// All earlier link openers are now inactive; image
							// openers can still contain this link.
							lastLinkStart = b.start
						}
						i = end
						continue
					}
				}
			}
		case '`':
			end := skipChar(data, i, '`')
			count := end - i
			positions := backticks[count]
			for len(positions) > 0 && positions[0] <= i {
				positions = positions[1:]
			}
			backticks[count] = positions
			if len(positions) > 0 {
				close := positions[0]
				literal := bytes.ReplaceAll(data[end:close], []byte{'\n'}, []byte{' '})
				if len(literal) >= 2 && literal[0] == ' ' && literal[len(literal)-1] == ' ' && len(bytes.Trim(literal, " ")) > 0 {
					literal = literal[1 : len(literal)-1]
				}
				appendNode(&ast.Code{Leaf: ast.Leaf{Literal: literal}})
				i = close + count
			} else {
				appendNode(newTextNode(data[i:end]))
				i = end
			}
			continue
		case '<':
			if consumed, node := commonMarkAngle(data, i, &angleClosers); consumed > 0 {
				if _, link := node.(*ast.Link); link {
					lastLinkStart = i + 1
				}
				appendNode(node)
				i += consumed
				continue
			}
		case '\\':
			if i+1 < len(data) && data[i+1] == '\n' {
				appendNode(&ast.Hardbreak{})
				i = skipHSpace(data, i+2)
				continue
			}
			if i+1 < len(data) && isEscapable(data[i+1]) {
				appendNode(newTextNode(data[i+1 : i+2]))
				i += 2
				continue
			}
		case '&':
			if consumed, node := entity(p, data, i); consumed > 0 {
				appendNode(node)
				i += consumed
				continue
			}
		case ' ', '\t':
			end := skipHSpace(data, i)
			if end == len(data) {
				i = end
				continue
			}
			if data[end] == '\n' {
				node := ast.Node(newTextNode([]byte{'\n'}))
				if end-i >= 2 && c == ' ' && bytes.IndexByte(data[i:end], '\t') < 0 {
					node = &ast.Hardbreak{}
				}
				appendNode(node)
				i = skipHSpace(data, end+1)
				continue
			}
			appendNode(newTextNode(data[i:end]))
			i = end
			continue
		case '\n':
			appendNode(newTextNode([]byte{'\n'}))
			i = skipHSpace(data, i+1)
			continue
		}
		end := i + 1
		for end < len(data) && !bytes.ContainsRune([]byte("*_[]!`<\\& \t\n"), rune(data[end])) {
			end++
		}
		appendNode(newTextNode(data[i:end]))
		i = end
	}
	commonMarkEmphasis(nil, &lastDelimiter)
	commonMarkAppendTokens(parent, root.next, nil)
}

func commonMarkLabel(label []byte) string {
	return textutil.CaseFold(strings.Join(strings.FieldsFunc(string(label), func(r rune) bool { return r == ' ' || r == '\t' || r == '\n' || r == '\r' }), " "))
}

func commonMarkUnicodeSpace(r rune) bool {
	return r == '\t' || r == '\n' || r == '\f' || r == '\r' || unicode.Is(unicode.Zs, r)
}

func commonMarkURL(source []byte) []byte {
	const hex = "0123456789ABCDEF"
	result := make([]byte, 0, len(source))
	for _, c := range source {
		if c >= 0x80 || c <= 0x20 || bytes.IndexByte([]byte("\"<>\\`{}|^[]"), c) >= 0 {
			result = append(result, '%', hex[c>>4], hex[c&15])
		} else {
			result = append(result, c)
		}
	}
	return result
}

func (p *Parser) commonMarkLinkTarget(data []byte, start int, label []byte) (end int, destination, title []byte, ok bool) {
	if start < len(data) && data[start] == '(' {
		if end, destination, title, ok = p.parseCommonMarkInlineLink(data, start); ok {
			return
		}
	}
	end = start
	if start < len(data) && data[start] == '[' {
		if close := commonMarkLabelEnd(data, start+1); close >= 0 {
			if close > start+1 {
				label = data[start+1 : close]
			}
			end = close + 1
		}
	}
	if utf8.RuneCount(label) > 999 || commonMarkLabel(label) == "" {
		return 0, nil, nil, false
	}
	ref, found := p.getRef(string(label))
	if !found {
		return 0, nil, nil, false
	}
	return end, commonMarkURL(commonMarkLiteral(ref.link)), commonMarkLiteral(ref.title), true
}

func commonMarkLabelEnd(data []byte, start int) int {
	for i, count := start, 0; i < len(data); {
		if data[i] == ']' {
			return i
		}
		if count >= 999 {
			return -1
		}
		if data[i] == '\\' && i+1 < len(data) && isEscapable(data[i+1]) {
			i += 2
			count += 2
			if count > 999 {
				return -1
			}
			continue
		}
		if data[i] == '[' {
			return -1
		}
		if data[i] == '\n' && IsEmpty(data[i+1:]) > 0 {
			return -1
		}
		_, width := utf8.DecodeRune(data[i:])
		i += width
		count++
	}
	return -1
}

func commonMarkLiteral(source []byte) []byte {
	if source == nil {
		return nil
	}
	result := make([]byte, 0, len(source))
	for i := 0; i < len(source); i++ {
		if source[i] == '\\' && i+1 < len(source) && isEscapable(source[i+1]) {
			i++
			result = append(result, source[i])
			continue
		}
		if source[i] == '&' {
			if n, node := entity(nil, source, i); n > 0 {
				result = append(result, node.AsLeaf().Literal...)
				i += n - 1
				continue
			}
		}
		result = append(result, source[i])
	}
	return result
}

// The same destination and title scanners serve inline links and definitions.
func commonMarkDestination(data []byte, start int) (end int, destination []byte, ok bool) {
	if start >= len(data) {
		return
	}
	if data[start] == '<' {
		for i := start + 1; i < len(data); i++ {
			if data[i] == '\\' && i+1 < len(data) && isEscapable(data[i+1]) {
				i++
				continue
			}
			if data[i] == '\n' || data[i] == '<' {
				return
			}
			if data[i] == '>' {
				return i + 1, data[start+1 : i], true
			}
		}
		return
	}
	depth := 0
	i := start
	for ; i < len(data); i++ {
		c := data[i]
		if c == '\\' && i+1 < len(data) && isEscapable(data[i+1]) {
			i++
			continue
		}
		if c <= 0x20 || c == 0x7f {
			break
		}
		if c == '(' {
			depth++
			if depth > 32 {
				return
			}
		}
		if c == ')' {
			if depth == 0 {
				break
			}
			depth--
		}
	}
	if depth != 0 || i == start {
		return
	}
	return i, data[start:i], true
}

func (p *Parser) commonMarkTitle(data []byte, start int) (end int, title []byte, ok bool) {
	if start >= len(data) || !bytes.ContainsRune([]byte("\"'("), rune(data[start])) {
		return
	}
	close := data[start]
	if close == '(' {
		close = ')'
	}
	end, found := p.linkStops.next(data, start+1, close)
	if !found {
		return 0, nil, false
	}
	for i := start + 1; i < end; i++ {
		if data[i] == '\\' && i+1 < len(data) && isEscapable(data[i+1]) {
			i++
			continue
		}
		if close == ')' && data[i] == '(' || data[i] == '\n' && IsEmpty(data[i+1:]) > 0 {
			return
		}
	}
	return end + 1, data[start+1 : end], true
}

func (p *Parser) parseCommonMarkInlineLink(data []byte, open int) (end int, destination, title []byte, ok bool) {
	i := commonMarkLinkSpace(data, open+1)
	if i < len(data) && data[i] == ')' {
		return i + 1, nil, nil, true
	}
	destEnd, dest, found := commonMarkDestination(data, i)
	if !found {
		return
	}
	j := commonMarkLinkSpace(data, destEnd)
	if j > destEnd {
		if titleEnd, rawTitle, found := p.commonMarkTitle(data, j); found {
			title = commonMarkLiteral(rawTitle)
			j = commonMarkLinkSpace(data, titleEnd)
		}
	}
	if j >= len(data) || data[j] != ')' {
		return 0, nil, nil, false
	}
	return j + 1, commonMarkURL(commonMarkLiteral(dest)), title, true
}

func commonMarkLinkSpace(data []byte, start int) int {
	i := skipHSpace(data, start)
	if i < len(data) && data[i] == '\n' {
		i = skipHSpace(data, i+1)
	}
	return i
}

var commonMarkAutolink = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]{1,31}:[^<>\x00-\x20]*$`)
var commonMarkEmail = regexp.MustCompile("^[A-Za-z0-9.!#$%&'*+/=?^_`{|}~-]+@[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?(?:\\.[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)*$")
var commonMarkHTMLTag = regexp.MustCompile("(?i)^(?:</[A-Za-z][A-Za-z0-9-]*[ \\t\\n]*>|<[A-Za-z][A-Za-z0-9-]*(?:[ \\t\\n]+[A-Za-z_:][A-Za-z0-9_.:-]*(?:[ \\t\\n]*=[ \\t\\n]*(?:[^ \\t\\n\"'=<>`]+|'[^']*'|\"[^\"]*\"))?)*[ \\t\\n]*/?>)")

func commonMarkAngle(source []byte, offset int, closers *[4]nextMatchCache) (int, ast.Node) {
	gt := closers[0].next(source, offset, findGt)
	if gt == len(source) {
		return 0, nil
	}
	data := source[offset:]
	for _, short := range []string{"<!-->", "<!--->"} {
		if bytes.HasPrefix(data, []byte(short)) {
			return len(short), &ast.HTMLSpan{Leaf: ast.Leaf{Literal: data[:len(short)]}}
		}
	}
	if end := gt - offset; end > 1 {
		label := data[1:end]
		if commonMarkAutolink.Match(label) || commonMarkEmail.Match(label) {
			destination := commonMarkURL(label)
			if commonMarkEmail.Match(label) {
				destination = append([]byte("mailto:"), destination...)
			}
			link := &ast.Link{Destination: destination}
			ast.AppendChild(link, newTextNode(label))
			return end + 1, link
		}
	}
	if tag := commonMarkHTMLTag.Find(data); tag != nil {
		return len(tag), &ast.HTMLSpan{Leaf: ast.Leaf{Literal: tag}}
	}
	for index, pair := range [][2]string{{"<!--", "-->"}, {"<?", "?>"}, {"<![CDATA[", "]]>"}} {
		if bytes.HasPrefix(data, []byte(pair[0])) {
			at := closers[index+1].next(source, offset+len(pair[0]), func(data []byte) int { return bytes.Index(data, []byte(pair[1])) })
			if at < len(source) {
				end := at + len(pair[1]) - offset
				return end, &ast.HTMLSpan{Leaf: ast.Leaf{Literal: data[:end]}}
			}
			return 0, nil
		}
	}
	if len(data) > 2 && data[1] == '!' && data[2] >= 'A' && data[2] <= 'Z' {
		end := gt - offset + 1
		return end, &ast.HTMLSpan{Leaf: ast.Leaf{Literal: data[:end]}}
	}
	return 0, nil
}
