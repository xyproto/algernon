package parser

// parseInlineLink parses the parenthesized destination and optional title
// following link text. open is the index of the opening parenthesis.
func parseInlineLink(data []byte, open int) (end int, destination, title []byte, ok bool) {
	i := skipSpace(data, open+1)
	destinationStart := i
	depth := 0

	for i < len(data) {
		char := data[i]
		switch {
		case char == '\\':
			i += 2
		case char == '(':
			depth++
			i++
		case char == ')':
			if depth == 0 {
				goto destinationEnd
			}
			depth--
			i++
		case depth == 0 && (char == '\'' || char == '"') && i > destinationStart && IsSpace(data[i-1]):
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
		i = titleStart
		closed := false
		for i < len(data) {
			switch data[i] {
			case '\\':
				i += 2
				continue
			case delimiter:
				closed = true
			case ')':
				if closed {
					goto titleEnd
				}
			}
			i++
		}
		return 0, nil, nil, false

	titleEnd:
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
