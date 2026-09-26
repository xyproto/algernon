package parser

// parseInlineLink parses the parenthesized destination and optional title
// following link text. open is the index of the opening parenthesis.
func parseInlineLink(p *Parser, data []byte, open int) (end int, destination, title []byte, ok bool) {
	i := skipSpace(data, open+1)
	destinationStart := i

	for i < len(data) {
		char := data[i]
		switch {
		case char == '\\':
			i += 2
		case char == '(':
			// A nested destination must itself be balanced. Jumping over it via
			// the shared delimiter table avoids rescanning an unclosed suffix
			// from every link opener.
			closeAt, _, found := lookupDelimiter(&p.parens, data, i, '(', ')')
			if !found {
				return 0, nil, nil, false
			}
			i = closeAt + 1
		case char == ')':
			goto destinationEnd
		case (char == '\'' || char == '"') && i > destinationStart && IsSpace(data[i-1]):
			goto destinationEnd
		default:
			i++
		}
	}
	return 0, nil, nil, false

destinationEnd:
	destinationEnd := i
	if data[i] == '\'' || data[i] == '"' {
		delimiter := data[i]
		titleStart := i + 1
		quoteEnd, found := p.linkStops.next(data, titleStart, delimiter)
		if !found {
			return 0, nil, nil, false
		}
		i, found = p.linkStops.next(data, quoteEnd+1, ')')
		if !found {
			return 0, nil, nil, false
		}
		end := i - 1
		for end > titleStart && IsSpace(data[end]) {
			end--
		}
		if data[end] == delimiter {
			if end > titleStart {
				title = data[titleStart:end]
			}
		} else {
			destinationEnd = i
		}
	}

	for destinationEnd > destinationStart && IsSpace(data[destinationEnd-1]) {
		destinationEnd--
	}
	if data[destinationStart] == '<' {
		destinationStart++
	}
	if data[destinationEnd-1] == '>' {
		destinationEnd--
	}
	if destinationEnd > destinationStart {
		destination = data[destinationStart:destinationEnd]
	}
	return i + 1, destination, title, true
}
