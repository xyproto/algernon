package parser

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/gomarkdown/markdown/ast"
)

// CommonMark uses the same block/inline passes and AST as the historical
// grammar, with stricter rules for container continuation and interruption.
func (p *Parser) commonMarkBlocks(data []byte) (loose bool) {
	if p.nesting >= p.maxNesting {
		return
	}
	p.nesting++
	defer func() { p.nesting-- }()
	previousStops := p.linkStops
	p.linkStops = byteIndex{data: data, chars: "\"')(", honorEscapes: true}
	defer func() { p.linkStops = previousStops }()
	parent := p.tip
	blank := false
	for len(data) > 0 {
		if n := IsEmpty(data); n > 0 {
			blank = true
			data = data[n:]
			continue
		}
		if blank && len(parent.GetChildren()) > 0 {
			loose = true
		}
		blank = false
		p.tip = parent
		n := p.commonMarkBlock(data)
		if n <= 0 {
			panic("CommonMark block parser made no progress")
		}
		data = data[n:]
	}
	p.tip = parent
	return
}

func commonMarkLine(data []byte) (line []byte, end int) {
	end = bytes.IndexByte(data, '\n')
	if end < 0 {
		return data, len(data)
	}
	return data[:end], end + 1
}

func commonMarkBlank(line []byte) bool { return len(bytes.Trim(line, " \t")) == 0 }

// Removing indentation may consume only part of a tab's four-column stop.
// Preserve its remaining columns as spaces, and leave later tabs untouched.
func commonMarkIndent(data []byte) int {
	column := 0
	for _, c := range data {
		if c == ' ' {
			column++
		} else if c == '\t' {
			column += 4 - column%4
		} else {
			break
		}
	}
	return column
}

func commonMarkDedent(data []byte, width int) []byte {
	return commonMarkDedentFrom(data, 0, width)
}

func commonMarkDedentFrom(data []byte, startColumn, width int) []byte {
	column, i := startColumn, 0
	for i < len(data) && column < startColumn+width {
		if data[i] == ' ' {
			column++
		} else if data[i] == '\t' {
			column += 4 - column%4
		} else {
			break
		}
		i++
	}
	if (startColumn+width)%4 != 0 {
		for i < len(data) && (data[i] == ' ' || data[i] == '\t') {
			if data[i] == ' ' {
				column++
			} else {
				column += 4 - column%4
			}
			i++
		}
	}
	if column > startColumn+width {
		return append(bytes.Repeat([]byte{' '}, column-startColumn-width), data[i:]...)
	}
	return data[i:]
}

func commonMarkHeading(data []byte) (level int, content []byte) {
	line, _ := commonMarkLine(data)
	if commonMarkIndent(line) > 3 {
		return
	}
	line = bytes.TrimLeft(line, " \t")
	level = skipChar(line, 0, '#')
	if level == 0 || level > 6 || level < len(line) && line[level] != ' ' && line[level] != '\t' {
		return 0, nil
	}
	content = bytes.Trim(line[level:], " \t")
	end := len(content)
	for end > 0 && content[end-1] == '#' {
		end--
	}
	if end < len(content) && (end == 0 || content[end-1] == ' ' || content[end-1] == '\t') {
		content = bytes.TrimRight(content[:end], " \t")
	}
	return
}

func commonMarkSetext(line []byte) int {
	if commonMarkIndent(line) > 3 {
		return 0
	}
	line = bytes.Trim(line, " \t")
	if len(line) == 0 || line[0] != '=' && line[0] != '-' {
		return 0
	}
	if skipChar(line, 0, line[0]) != len(line) {
		return 0
	}
	if line[0] == '=' {
		return 1
	}
	return 2
}

func commonMarkThematic(line []byte) bool {
	if commonMarkIndent(line) > 3 {
		return false
	}
	line = bytes.TrimLeft(line, " \t")
	if len(line) == 0 || line[0] != '-' && line[0] != '*' && line[0] != '_' {
		return false
	}
	marker, count := line[0], 0
	for _, c := range line {
		if c == marker {
			count++
		} else if c != ' ' && c != '\t' {
			return false
		}
	}
	return count >= 3
}

func commonMarkFence(line []byte) (marker byte, count, indent int, info []byte) {
	indent = commonMarkIndent(line)
	if indent > 3 {
		return
	}
	line = bytes.TrimLeft(line, " \t")
	if len(line) == 0 || line[0] != '`' && line[0] != '~' {
		return
	}
	marker = line[0]
	count = skipChar(line, 0, marker)
	if count < 3 {
		return 0, 0, 0, nil
	}
	info = bytes.Trim(line[count:], " \t")
	if marker == '`' && bytes.IndexByte(info, '`') >= 0 {
		return 0, 0, 0, nil
	}
	return
}

func commonMarkQuotePrefix(line []byte) int {
	if commonMarkIndent(line) > 3 {
		return 0
	}
	i := skipHSpace(line, 0)
	if i < len(line) && line[i] == '>' {
		return i + 1
	}
	return 0
}

func commonMarkQuoteContent(line []byte, prefix int) []byte {
	if prefix < len(line) && (line[prefix] == ' ' || line[prefix] == '\t') {
		return commonMarkDedentFrom(line[prefix:], prefix, 1)
	}
	return line[prefix:]
}

type commonMarkListMarker struct {
	indent, width, start int
	char                 byte
	ordered              bool
	content              []byte
}

func commonMarkListPrefix(line []byte) (m commonMarkListMarker, ok bool) {
	m.indent = commonMarkIndent(line)
	if m.indent > 3 {
		return
	}
	i := skipHSpace(line, 0)
	start := i
	if i >= len(line) {
		return
	}
	if line[i] == '-' || line[i] == '+' || line[i] == '*' {
		m.char = line[i]
		i++
	} else {
		for i < len(line) && line[i] >= '0' && line[i] <= '9' {
			i++
		}
		if i == start || i-start > 9 || i >= len(line) || line[i] != '.' && line[i] != ')' {
			return
		}
		m.ordered = true
		m.start, _ = strconv.Atoi(string(line[start:i]))
		m.char = line[i]
		i++
	}
	if i < len(line) && line[i] != ' ' && line[i] != '\t' {
		return
	}
	markerWidth := i
	if commonMarkBlank(line[i:]) {
		m.width = markerWidth + 1
		m.content = nil
		return m, true
	}
	column := i
	j := i
	for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
		if line[j] == ' ' {
			column++
		} else {
			column += 4 - column%4
		}
		j++
	}
	padding := column - i
	if padding > 4 {
		padding = 1
	}
	m.width = markerWidth + padding
	m.content = commonMarkDedentFrom(line[markerWidth:], markerWidth, padding)
	return m, true
}

func commonMarkInterrupts(line []byte) bool {
	if commonMarkIndent(line) > 3 {
		return false
	}
	if level, _ := commonMarkHeading(line); level > 0 {
		return true
	}
	if commonMarkThematic(line) || commonMarkQuotePrefix(line) > 0 {
		return true
	}
	if marker, _, _, _ := commonMarkFence(line); marker != 0 {
		return true
	}
	if kind, _ := commonMarkHTMLStart(line); kind > 0 && kind < 7 {
		return true
	}
	if m, ok := commonMarkListPrefix(line); ok && len(m.content) > 0 && (!m.ordered || m.start == 1) {
		return true
	}
	return false
}

func (p *Parser) commonMarkBlock(data []byte) int {
	line, end := commonMarkLine(data)
	if commonMarkIndent(line) >= 4 {
		return p.commonMarkCode(data)
	}
	if level, content := commonMarkHeading(line); level > 0 {
		p.AddBlock(&ast.Heading{Level: level, Container: ast.Container{Content: content}})
		return end
	}
	if marker, count, indent, info := commonMarkFence(line); marker != 0 {
		return p.commonMarkFencedCode(data, end, marker, count, indent, info)
	}
	if commonMarkThematic(line) {
		p.AddBlock(&ast.HorizontalRule{})
		return end
	}
	if commonMarkQuotePrefix(line) > 0 {
		return p.commonMarkQuote(data)
	}
	if kind, closer := commonMarkHTMLStart(line); kind > 0 {
		return p.commonMarkHTML(data, kind, closer)
	}
	if m, ok := commonMarkListPrefix(line); ok {
		return p.commonMarkList(data, m)
	}
	return p.commonMarkParagraph(data)
}

func (p *Parser) commonMarkCode(data []byte) int {
	var work bytes.Buffer
	i, last := 0, 0
	for i < len(data) {
		line, end := commonMarkLine(data[i:])
		if commonMarkBlank(line) {
			work.Write(commonMarkDedent(line, 4))
			work.WriteByte('\n')
		} else {
			if commonMarkIndent(line) < 4 {
				break
			}
			work.Write(commonMarkDedent(line, 4))
			work.WriteByte('\n')
			last = work.Len()
		}
		i += end
	}
	p.AddBlock(&ast.CodeBlock{Leaf: ast.Leaf{Literal: work.Bytes()[:last]}})
	return i
}

func (p *Parser) commonMarkFencedCode(data []byte, i int, marker byte, count, indent int, info []byte) int {
	var work bytes.Buffer
	for i < len(data) {
		line, end := commonMarkLine(data[i:])
		trimmed := bytes.TrimLeft(line, " \t")
		if commonMarkIndent(line) <= 3 && len(trimmed) > 0 && trimmed[0] == marker && skipChar(trimmed, 0, marker) >= count && commonMarkBlank(trimmed[skipChar(trimmed, 0, marker):]) {
			i += end
			break
		}
		work.Write(commonMarkDedent(line, indent))
		work.WriteByte('\n')
		i += end
	}
	p.AddBlock(&ast.CodeBlock{IsFenced: true, Info: commonMarkLiteral(info), Leaf: ast.Leaf{Literal: work.Bytes()}})
	return i
}

func (p *Parser) commonMarkParagraph(data []byte) int {
	i := 0
	// Definitions are accepted only at the beginning of a paragraph.
	if n := p.commonMarkReference(data); n > 0 {
		return n
	}
	var work bytes.Buffer
	for i < len(data) {
		line, end := commonMarkLine(data[i:])
		if commonMarkBlank(line) {
			break
		}
		if i > 0 {
			if level := commonMarkSetext(line); level > 0 && !p.commonMarkLazy(data[i:]) {
				p.AddBlock(&ast.Heading{Level: level, Container: ast.Container{Content: bytes.Trim(work.Bytes(), " \t\n")}})
				return i + end
			}
			if commonMarkInterrupts(line) {
				break
			}
		}
		work.Write(bytes.TrimLeft(line, " \t"))
		work.WriteByte('\n')
		i += end
	}
	p.renderParagraph(work.Bytes())
	return i
}

func (p *Parser) commonMarkReference(data []byte) int {
	start := skipHSpace(data, 0)
	if start >= len(data) || data[start] != '[' {
		return 0
	}
	close := commonMarkLabelEnd(data, start+1)
	if close < 0 || close+1 >= len(data) || data[close+1] != ':' {
		return 0
	}
	label := data[start+1 : close]
	if commonMarkLabel(label) == "" {
		return 0
	}
	i := skipHSpace(data, close+2)
	if i < len(data) && data[i] == '\n' {
		i = skipHSpace(data, i+1)
	}
	destEnd, destination, ok := commonMarkDestination(data, i)
	if !ok {
		return 0
	}
	end := skipHSpace(data, destEnd)
	lineEnd := end
	if end < len(data) && data[end] != '\n' {
		lineEnd = -1
	}
	var title []byte
	titleStart := commonMarkLinkSpace(data, destEnd)
	if titleStart > destEnd {
		if titleEnd, rawTitle, found := p.commonMarkTitle(data, titleStart); found {
			titleEnd = skipHSpace(data, titleEnd)
			if titleEnd == len(data) || data[titleEnd] == '\n' {
				title = rawTitle
				end = titleEnd
				lineEnd = end
			}
		}
	}
	if lineEnd < 0 {
		return 0
	}
	if end < len(data) && data[end] == '\n' {
		end++
	}
	key := commonMarkLabel(label)
	if _, exists := p.refs[key]; !exists {
		p.refs[key] = &reference{link: destination, title: title}
	}
	p.AddBlock(&ast.ReferenceDefinition{Label: label, Destination: commonMarkURL(commonMarkLiteral(destination)), Title: commonMarkLiteral(title), Leaf: ast.Leaf{Literal: data[:end]}})
	return end
}

// A container may omit its prefix only while its innermost paragraph is open.
// Track that leaf while collecting the container's source for the shared block
// pass; fenced code and HTML cannot use lazy continuation.
type commonMarkContinuation struct {
	paragraph bool
	fence     byte
	count     int
	html      int
	closer    string
}

func (s *commonMarkContinuation) feed(line []byte) {
	for {
		if prefix := commonMarkQuotePrefix(line); prefix > 0 {
			line = commonMarkQuoteContent(line, prefix)
			continue
		}
		if m, ok := commonMarkListPrefix(line); ok && !commonMarkThematic(line) {
			line = m.content
			continue
		}
		break
	}
	if s.fence != 0 {
		trimmed := bytes.Trim(line, " \t")
		if commonMarkIndent(line) <= 3 && len(trimmed) > 0 && trimmed[0] == s.fence && skipChar(trimmed, 0, s.fence) >= s.count && commonMarkBlank(trimmed[skipChar(trimmed, 0, s.fence):]) {
			s.fence = 0
		}
		s.paragraph = false
		return
	}
	if s.html != 0 {
		if s.html >= 6 && commonMarkBlank(line) || s.closer != "" && strings.Contains(strings.ToLower(string(line)), s.closer) {
			s.html = 0
		}
		s.paragraph = false
		return
	}
	if commonMarkBlank(line) {
		s.paragraph = false
		return
	}
	if marker, count, _, _ := commonMarkFence(line); marker != 0 {
		s.fence, s.count = marker, count
		s.paragraph = false
		return
	}
	if kind, closer := commonMarkHTMLStart(line); kind > 0 && (!s.paragraph || kind < 7) {
		if closer == "" || !strings.Contains(strings.ToLower(string(line)), closer) {
			s.html, s.closer = kind, closer
		}
		s.paragraph = false
		return
	}
	if level, _ := commonMarkHeading(line); level > 0 || commonMarkThematic(line) || s.paragraph && commonMarkSetext(line) > 0 {
		s.paragraph = false
		return
	}
	if !s.paragraph && commonMarkIndent(line) >= 4 {
		return
	}
	s.paragraph = true
}

func (p *Parser) commonMarkQuote(data []byte) int {
	var work bytes.Buffer
	var state commonMarkContinuation
	var lazyOffsets []int
	i := 0
	for i < len(data) {
		line, end := commonMarkLine(data[i:])
		lazy := p.commonMarkLazy(data[i:])
		if prefix := commonMarkQuotePrefix(line); prefix > 0 {
			line = commonMarkQuoteContent(line, prefix)
			state.feed(line)
		} else if !state.paragraph || commonMarkBlank(line) || commonMarkInterrupts(line) {
			break
		} else {
			lazy = true
		}
		if lazy {
			lazyOffsets = append(lazyOffsets, work.Len())
		}
		work.Write(line)
		work.WriteByte('\n')
		i += end
	}
	block := p.AddBlock(&ast.BlockQuote{})
	p.commonMarkMarkLazy(work.Bytes(), lazyOffsets)
	p.Block(work.Bytes())
	p.Finalize(block)
	return i
}

func (p *Parser) commonMarkList(data []byte, first commonMarkListMarker) int {
	list := &ast.List{Tight: true, BulletChar: first.char, Delimiter: first.char, Start: first.start}
	if first.ordered {
		list.ListFlags = ast.ListTypeOrdered
	}
	p.AddBlock(list)
	i := 0
	for i < len(data) {
		line, end := commonMarkLine(data[i:])
		m, ok := commonMarkListPrefix(line)
		if !ok || m.char != first.char || m.ordered != first.ordered || commonMarkThematic(line) {
			break
		}
		item := &ast.ListItem{ListFlags: list.ListFlags, BulletChar: m.char, Delimiter: m.char}
		p.tip = list
		p.addChild(item)
		var work bytes.Buffer
		var lazyOffsets []int
		if p.commonMarkLazy(data[i:]) {
			lazyOffsets = append(lazyOffsets, 0)
		}
		work.Write(m.content)
		work.WriteByte('\n')
		var state commonMarkContinuation
		state.feed(m.content)
		i += end
		blankStart := -1
		betweenItems := false
		for i < len(data) {
			line, end = commonMarkLine(data[i:])
			if commonMarkBlank(line) {
				if blankStart < 0 {
					blankStart = i
				}
				work.WriteByte('\n')
				state.paragraph = false
				i += end
				continue
			}
			if commonMarkIndent(line) >= m.width && (len(m.content) > 0 || blankStart < 0) {
				if p.commonMarkLazy(data[i:]) {
					lazyOffsets = append(lazyOffsets, work.Len())
				}
				content := commonMarkDedent(line, m.width)
				work.Write(content)
				work.WriteByte('\n')
				state.feed(content)
				blankStart = -1
				i += end
				continue
			}
			if next, found := commonMarkListPrefix(line); found && !commonMarkThematic(line) {
				betweenItems = next.char == first.char && next.ordered == first.ordered
				break
			}
			if state.paragraph && blankStart < 0 && !commonMarkInterrupts(line) {
				lazyOffsets = append(lazyOffsets, work.Len())
				work.Write(line)
				work.WriteByte('\n')
				i += end
				continue
			}
			break
		}
		p.commonMarkMarkLazy(work.Bytes(), lazyOffsets)
		if p.commonMarkBlocks(work.Bytes()) {
			list.Tight = false
		}
		if blankStart >= 0 {
			if betweenItems {
				list.Tight = false
			} else {
				i = blankStart
			}
		}
		p.tip = list
		if !betweenItems {
			break
		}
	}
	children := list.GetChildren()
	for index, child := range children {
		item := child.(*ast.ListItem)
		item.Tight = list.Tight
		if index == 0 {
			item.ListFlags |= ast.ListItemBeginningOfList
		}
		if index == len(children)-1 {
			item.ListFlags |= ast.ListItemEndOfList
		}
		if !list.Tight {
			item.ListFlags |= ast.ListItemContainsBlock
		}
	}
	p.Finalize(list)
	return i
}

func (p *Parser) commonMarkLazy(data []byte) bool {
	return len(data) > 0 && p.commonMarkLazyLines[&data[0]]
}

func (p *Parser) commonMarkMarkLazy(data []byte, offsets []int) {
	if len(offsets) == 0 {
		return
	}
	if p.commonMarkLazyLines == nil {
		p.commonMarkLazyLines = make(map[*byte]bool)
	}
	for _, offset := range offsets {
		if offset < len(data) {
			p.commonMarkLazyLines[&data[offset]] = true
		}
	}
}

var commonMarkBlockTags = stringSet("address", "article", "aside", "base", "basefont", "blockquote", "body", "caption", "center", "col", "colgroup", "dd", "details", "dialog", "dir", "div", "dl", "dt", "fieldset", "figcaption", "figure", "footer", "form", "frame", "frameset", "h1", "h2", "h3", "h4", "h5", "h6", "head", "header", "hr", "html", "iframe", "legend", "li", "link", "main", "menu", "menuitem", "nav", "noframes", "ol", "optgroup", "option", "p", "param", "search", "section", "summary", "table", "tbody", "td", "tfoot", "th", "thead", "title", "tr", "track", "ul")

func commonMarkHTMLStart(line []byte) (kind int, closer string) {
	if commonMarkIndent(line) > 3 {
		return
	}
	line = bytes.TrimLeft(line, " \t")
	if len(line) == 0 || line[0] != '<' {
		return
	}
	lower := strings.ToLower(string(line))
	for _, tag := range []string{"script", "pre", "style", "textarea"} {
		prefix := "<" + tag
		if strings.HasPrefix(lower, prefix) && (len(line) == len(prefix) || line[len(prefix)] == '>' || IsSpace(line[len(prefix)])) {
			return 1, "</" + tag + ">"
		}
	}
	for index, pair := range [][2]string{{"<!--", "-->"}, {"<?", "?>"}, {"<!", ">"}, {"<![CDATA[", "]]>"}} {
		if bytes.HasPrefix(line, []byte(pair[0])) {
			if index == 2 && (len(line) <= 2 || line[2] < 'A' || line[2] > 'Z') {
				continue
			}
			return index + 2, pair[1]
		}
	}
	i := 1
	if i < len(line) && line[i] == '/' {
		i++
	}
	start := i
	for i < len(line) && (IsAlnum(line[i]) || line[i] == '-') {
		i++
	}
	if i > start && (i == len(line) || IsSpace(line[i]) || line[i] == '>' || line[i] == '/' && i+1 < len(line) && line[i+1] == '>') && inStringSet(commonMarkBlockTags, strings.ToLower(string(line[start:i]))) {
		return 6, ""
	}
	if tag := commonMarkHTMLTag.Find(line); tag != nil && commonMarkBlank(line[len(tag):]) {
		return 7, ""
	}
	return
}

func (p *Parser) commonMarkHTML(data []byte, kind int, closer string) int {
	i := 0
	for i < len(data) {
		line, end := commonMarkLine(data[i:])
		if kind >= 6 && commonMarkBlank(line) {
			break
		}
		i += end
		if closer != "" && strings.Contains(strings.ToLower(string(line)), closer) {
			break
		}
	}
	p.AddBlock(&ast.HTMLBlock{Leaf: ast.Leaf{Literal: data[:i]}})
	return i
}
