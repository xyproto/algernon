package parser

import (
	"bytes"
	"unicode"

	"github.com/gomarkdown/markdown/ast"
)

// sanitizeHeadingID returns a sanitized anchor name for the given text.
// Taken from https://github.com/shurcooL/sanitized_anchor_name/blob/master/main.go#L14:1
func sanitizeHeadingID(text string) string {
	var anchorName []rune
	var futureDash = false
	for _, r := range text {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			if futureDash && len(anchorName) > 0 {
				anchorName = append(anchorName, '-')
			}
			futureDash = false
			anchorName = append(anchorName, unicode.ToLower(r))
		default:
			futureDash = true
		}
	}
	if len(anchorName) == 0 {
		return "empty"
	}
	return string(anchorName)
}

// Parse Block-level data.
// Note: this function and many that it calls assume that
// the input buffer ends with a newline.
func (p *Parser) isPrefixHeading(data []byte) bool {
	if len(data) > 0 && data[0] != '#' {
		return false
	}

	if p.extensions&SpaceHeadings != 0 {
		level := skipCharN(data, 0, '#', 6)
		if level == len(data) || data[level] != ' ' {
			return false
		}
	}
	return true
}

// parseHeadingContent extracts the text range and optional {#id} for a heading
// whose content starts at i and whose line ends just before the newline at end.
// It returns the heading id ("" if none), the index where the heading text ends
// (after trimming a trailing {#id} and any closing '#' markers and spaces), and
// the number of bytes the whole heading line occupies.
func (p *Parser) parseHeadingContent(data []byte, i, end int) (id string, contentEnd, skip int) {
	skip = end
	if p.extensions&HeadingIDs != 0 {
		j, k := 0, 0
		// find start/end of heading id
		for j = i; j < end-1 && (data[j] != '{' || data[j+1] != '#'); j++ {
		}
		for k = j + 1; k < end && data[k] != '}'; k++ {
		}
		// extract heading id iff found
		if j < end && k < end {
			id = string(data[j+2 : k])
			end = j
			skip = k + 1
			end = backChar(data, end, ' ')
		}
	}
	// strip trailing closing '#' markers and surrounding spaces
	for end > 0 && data[end-1] == '#' {
		if isEscape(data, end-1) {
			break
		}
		end--
	}
	end = backChar(data, end, ' ')
	return id, end, skip
}

// setHeadingID assigns block's id, auto-generating one from text when id is
// empty and AutoHeadingIDs is enabled (recording it for later uniquification).
func (p *Parser) setHeadingID(block *ast.Heading, id string, text []byte) {
	block.HeadingID = id
	if id == "" && p.extensions&AutoHeadingIDs != 0 {
		block.HeadingID = sanitizeHeadingID(string(text))
		p.allHeadingsWithAutoID = append(p.allHeadingsWithAutoID, block)
	}
}

func (p *Parser) prefixHeading(data []byte) int {
	level := skipCharN(data, 0, '#', 6)
	i := skipChar(data, level, ' ')
	end := skipUntilChar(data, i, '\n')
	id, end, skip := p.parseHeadingContent(data, i, end)
	if end > i {
		block := &ast.Heading{
			Container: ast.Container{Content: data[i:end]},
			Level:     level,
		}
		p.setHeadingID(block, id, data[i:end])
		p.AddBlock(block)
	}
	return skip
}

func (p *Parser) isPrefixSpecialHeading(data []byte) bool {
	if p.extensions|Mmark == 0 {
		return false
	}
	if len(data) < 4 {
		return false
	}
	if data[0] != '.' {
		return false
	}
	if data[1] != '#' {
		return false
	}
	if data[2] == '#' { // we don't support level, so nack this.
		return false
	}

	if p.extensions&SpaceHeadings != 0 {
		if data[2] != ' ' {
			return false
		}
	}
	return true
}

func (p *Parser) prefixSpecialHeading(data []byte) int {
	i := skipChar(data, 2, ' ') // ".#" skipped
	end := skipUntilChar(data, i, '\n')
	id, end, skip := p.parseHeadingContent(data, i, end)
	if end > i {
		block := &ast.Heading{
			Container: ast.Container{Literal: data[i:end], Content: data[i:end]},
			IsSpecial: true,
			Level:     1,
		}
		p.setHeadingID(block, id, data[i:end])
		p.AddBlock(block)
	}
	return skip
}

func isUnderlinedHeading(data []byte) int {
	marker := data[0]
	if marker != '=' && marker != '-' {
		return 0
	}
	i := skipChar(data, 1, marker)
	i = skipChar(data, i, ' ')
	if i < len(data) && data[i] == '\n' {
		if marker == '=' {
			return 1
		}
		return 2
	}
	return 0
}

func (p *Parser) titleBlock(data []byte) int {
	if data[0] != '%' {
		return 0
	}
	splitData := bytes.Split(data, []byte("\n"))
	var i int
	for idx, b := range splitData {
		if !bytes.HasPrefix(b, []byte("%")) {
			i = idx // - 1
			break
		}
	}

	data = bytes.Join(splitData[0:i], []byte("\n"))
	consumed := len(data)
	data = bytes.TrimPrefix(data, []byte("% "))
	data = bytes.Replace(data, []byte("\n% "), []byte("\n"), -1)
	block := &ast.Heading{
		Level:        1,
		IsTitleblock: true,
	}
	block.Content = data
	p.AddBlock(block)

	return consumed
}

func isHRule(data []byte) bool {
	i := skipCharN(data, 0, ' ', 3)

	// look at the hrule char
	if data[i] != '*' && data[i] != '-' && data[i] != '_' {
		return false
	}
	c := data[i]

	// the whole line must be the char or whitespace
	n := 0
	for i < len(data) && data[i] != '\n' {
		switch {
		case data[i] == c:
			n++
		case data[i] != ' ':
			return false
		}
		i++
	}

	return n >= 3
}
