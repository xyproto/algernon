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
	var floors [3][3][2]*commonMarkDelimiter
	for closer != nil {
		if !closer.close {
			closer = closer.next
			continue
		}
		kind := 0
		switch closer.char {
		case '_':
			kind = 1
		case '~':
			kind = 2
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
			// strikethrough delimiters are always two tildes, so the rule of
			// three doesn't apply to them
			if opener.char == closer.char && opener.open && (closer.char == '~' ||
				!((opener.close || closer.open) && (opener.length+closer.length)%3 == 0 && (opener.length%3 != 0 || closer.length%3 != 0))) {
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
		if closer.char == '~' {
			use = 2
			node = &ast.Del{}
		} else if opener.length >= 2 && closer.length >= 2 {
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
	strikethrough := p.commonMarkExtensions&Strikethrough != 0
	autolink := p.commonMarkExtensions&Autolink != 0
	// autolinks are the links made from bare URLs, which become text again
	// when they end up inside another link
	var autolinks map[*ast.Link]bool
	// inside raw <a> ... </a> URLs are not made into links
	inAnchor := false
	specials := "*_[]!`<\\& \t\n"
	if strikethrough {
		specials += "~"
	}
	for i := 0; i < len(data); {
		c := data[i]
		switch c {
		case '~':
			if strikethrough {
				end := skipChar(data, i, c)
				token := appendNode(newTextNode(data[i:end]))
				if end-i == 2 {
					open, close := commonMarkFlanking(data, i, end)
					d := &commonMarkDelimiter{token: token, char: c, length: 2, open: open, close: close, prev: lastDelimiter}
					if lastDelimiter != nil {
						lastDelimiter.next = d
					}
					lastDelimiter = d
				}
				i = end
				continue
			}
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
				if span, ok := node.(*ast.HTMLSpan); ok && autolink {
					if commonMarkIsTag(span.Literal, "<a") {
						inAnchor = true
					} else if commonMarkIsTag(span.Literal, "</a") {
						inAnchor = false
					}
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
			if isBackslashEscape(data, i) {
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
		if autolink && !inAnchor && commonMarkURLStart(data, i) {
			if n := p.commonMarkBareURL(data[i:], bracket != nil); n > 0 {
				literal := commonMarkLiteral(data[i : i+n])
				link := &ast.Link{Destination: commonMarkURL(literal)}
				ast.AppendChild(link, newTextNode(literal))
				if autolinks == nil {
					autolinks = map[*ast.Link]bool{}
				}
				autolinks[link] = true
				appendNode(link)
				i += n
				continue
			}
		}
		end := i + 1
		for end < len(data) {
			if c := data[end]; c == ' ' || c == '\n' {
				// A single space between words, or a line break before a
				// line that starts with text, stays in the text. A space
				// run, or a space or line break before a tab, a space or a
				// line break, goes to the ' ' and '\n' cases above.
				if end+1 == len(data) || data[end+1] == ' ' || data[end+1] == '\t' || data[end+1] == '\n' {
					break
				}
			} else if strings.IndexByte(specials, data[end]) >= 0 || autolink && commonMarkURLStart(data, end) {
				break
			}
			end++
		}
		appendNode(newTextNode(data[i:end]))
		i = end
	}
	commonMarkEmphasis(nil, &lastDelimiter)
	commonMarkAppendTokens(parent, root.next, nil)
	if len(autolinks) > 0 {
		commonMarkUnnestAutolinks(parent, autolinks, false)
	}
}

// commonMarkIsTag reports whether an HTML span starts with tag, like "<a".
func commonMarkIsTag(literal []byte, tag string) bool {
	return len(literal) > len(tag) && hasPrefixCaseInsensitive(literal, []byte(tag)) &&
		(IsSpace(literal[len(tag)]) || literal[len(tag)] == '>' || literal[len(tag)] == '/')
}

// commonMarkURLStart reports whether a bare URL could start at data[i]: one
// of its schemes starts there and not in the middle of a word.
func commonMarkURLStart(data []byte, i int) bool {
	switch data[i] | 0x20 {
	case 'h', 'f', 'm':
	default:
		return false
	}
	if i > 0 && IsAlnum(data[i-1]) {
		return false
	}
	for _, prefix := range commonMarkURLSchemes {
		if hasPrefixCaseInsensitive(data[i:], prefix) {
			return true
		}
	}
	return false
}

var commonMarkURLSchemes = [][]byte{[]byte("http://"), []byte("https://"), []byte("ftp://"), []byte("mailto:")}

// commonMarkBareURL returns the length of the URL that data starts with, or 0.
// Like GitHub Flavored Markdown, trailing punctuation is not part of it and
// neither is a closing parenthesis or bracket without an opening one. In
// link text (inBrackets), the URL ends at a ']'.
func (p *Parser) commonMarkBareURL(data []byte, inBrackets bool) int {
	isSafeURL := p.IsSafeURLOverride
	if isSafeURL == nil {
		isSafeURL = IsSafeURL
	}
	if !isSafeURL(data) {
		return 0
	}
	end := 0
	// how many more closing than opening (), [] and {} the URL has
	var unbalanced [3]int
	const closers, openers = ")]}", "([{"
	for end < len(data) && !isEndOfLink(data[end]) && !(inBrackets && data[end] == ']') {
		if k := strings.IndexByte(closers, data[end]); k >= 0 {
			unbalanced[k]++
		} else if k := strings.IndexByte(openers, data[end]); k >= 0 {
			unbalanced[k]--
		}
		end++
	}
	for end > 0 {
		c := data[end-1]
		if strings.IndexByte("?!.,:*_~'\"", c) >= 0 {
			end--
			continue
		}
		if c == ';' {
			if linkEndsWithEntity(data, end) {
				end = bytes.LastIndexByte(data[:end], '&')
			} else {
				end--
			}
			continue
		}
		if k := strings.IndexByte(closers, c); k >= 0 && unbalanced[k] > 0 {
			unbalanced[k]--
			end--
			continue
		}
		break
	}
	for _, prefix := range commonMarkURLSchemes {
		if hasPrefixCaseInsensitive(data, prefix) && end <= len(prefix) {
			return 0
		}
	}
	return end
}

// commonMarkUnnestAutolinks turns autolinks inside other links back into text.
func commonMarkUnnestAutolinks(node ast.Node, autolinks map[*ast.Link]bool, inLink bool) {
	children := node.GetChildren()
	replaced := false
	for i, child := range children {
		link, isLink := child.(*ast.Link)
		if isLink && inLink && autolinks[link] {
			text := newTextNode(link.Children[0].AsLeaf().Literal)
			text.SetParent(node)
			children[i] = text
			replaced = true
			continue
		}
		commonMarkUnnestAutolinks(child, autolinks, inLink || isLink)
	}
	if replaced {
		// relinks the siblings
		node.SetChildren(children)
	}
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
	if !found || !p.expandRef(ref) {
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
		if isBackslashEscape(data, i) {
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
		if isBackslashEscape(source, i) {
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
			if isBackslashEscape(data, i) {
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
		if isBackslashEscape(data, i) {
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
		if isBackslashEscape(data, i) {
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
