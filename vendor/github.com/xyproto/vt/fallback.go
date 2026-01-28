//go:build !windows

package vt

// parseCSIFallback handles common CSI sequences that include parameters.
// seq is the parameter bytes between ESC[ and the final byte.
func parseCSIFallback(seq []byte, final byte) (Event, bool) {
	switch final {
	case 'A':
		return Event{Kind: EventKey, Key: 253}, true // Up Arrow
	case 'B':
		return Event{Kind: EventKey, Key: 255}, true // Down Arrow
	case 'C':
		return Event{Kind: EventKey, Key: 254}, true // Right Arrow
	case 'D':
		return Event{Kind: EventKey, Key: 252}, true // Left Arrow
	case 'H':
		return Event{Kind: EventKey, Key: 1}, true // Home
	case 'F':
		return Event{Kind: EventKey, Key: 5}, true // End
	case '~':
		params, ok := parseCSIParams(seq)
		if !ok || len(params) == 0 {
			return Event{}, false
		}
		switch params[0] {
		case 1, 7:
			return Event{Kind: EventKey, Key: 1}, true // Home
		case 4, 8:
			return Event{Kind: EventKey, Key: 5}, true // End
		case 5:
			return Event{Kind: EventKey, Key: 251}, true // Page Up
		case 6:
			return Event{Kind: EventKey, Key: 250}, true // Page Down
		}
	}
	return Event{}, false
}

func parseCSIParams(seq []byte) ([]int, bool) {
	if len(seq) == 0 {
		return nil, true
	}
	params := make([]int, 0, 2)
	value := 0
	hasDigit := false
	for _, b := range seq {
		switch {
		case b >= '0' && b <= '9':
			value = value*10 + int(b-'0')
			hasDigit = true
		case b == ';':
			if hasDigit {
				params = append(params, value)
			} else {
				params = append(params, 0)
			}
			value = 0
			hasDigit = false
		default:
			return nil, false
		}
	}
	if hasDigit {
		params = append(params, value)
	} else if len(seq) > 0 {
		// Trailing ';' implies an empty parameter.
		params = append(params, 0)
	}
	return params, true
}
