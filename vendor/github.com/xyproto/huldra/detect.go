package huldra

import (
	"errors"
	"strings"
)

var errNoHTML = errors.New("could not find a <html> tag")

func IsHTML(data []byte) bool {
	return HasHTMLTag(data, 200)
}

func HasHTMLTag(data []byte, maxpos uint64) bool {
	var (
		foundCounter uint8
		i            uint64
	)
	for _, r := range data {
		switch foundCounter {
		case 1:
			if r == 'h' || r == 'H' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 2:
			if r == 't' || r == 'T' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 3:
			if r == 'm' || r == 'M' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 4:
			if r == 'l' || r == 'L' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 5:
			if r == '>' || r == ' ' {
				// found "<html " or "<html>"
				return true
			}
		default:
			if r == '<' {
				foundCounter = 1
			} else {
				foundCounter = 0
			}
		}
		if maxpos > 0 && i > maxpos {
			return false // no <html> tag for the first maxpos runes
		}
		i++
	}
	return false
}

func HasScriptTag(data []byte, maxpos uint64) bool {
	var (
		foundCounter uint8
		i            uint64
	)
	for _, r := range data {
		switch foundCounter {
		case 1:
			if r == 's' || r == 'S' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 2:
			if r == 'c' || r == 'C' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 3:
			if r == 'r' || r == 'R' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 4:
			if r == 'i' || r == 'I' {
				foundCounter++
			} else {
				foundCounter = 0
			}

		case 5:
			if r == 'p' || r == 'P' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 6:
			if r == 't' || r == 'T' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 7:
			if r == '>' || r == ' ' {
				// found "<script " or "<script>"
				return true
			}
		default:
			if r == '<' {
				foundCounter = 1
			} else {
				foundCounter = 0
			}
		}
		if maxpos > 0 && i > maxpos {
			return false // no <script> tag for the first maxpos runes
		}
		i++
	}
	return false
}

func HTMLIndex(data []byte, maxpos uint64) (uint64, error) {
	var (
		foundCounter uint8
		i            uint64
		pos          uint64
	)
	for _, r := range data {
		switch foundCounter {
		case 1:
			if r == 'h' || r == 'H' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 2:
			if r == 't' || r == 'T' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 3:
			if r == 'm' || r == 'M' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 4:
			if r == 'l' || r == 'L' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 5:
			if r == '>' || r == ' ' {
				// found "<html " or "<html>"
				return pos, nil
			}
		default:
			if r == '<' {
				pos = i
				foundCounter = 1
			} else {
				foundCounter = 0
			}
		}
		if maxpos > 0 && i > maxpos {
			return 0, errNoHTML // no <html> tag for the first maxpos runes
		}
		i++
	}
	return 0, errNoHTML
}

func HTMLIndexString(s string, maxpos uint64) (uint64, error) {
	var (
		foundCounter uint8
		pos          uint64
		i            uint64
	)
	for _, r := range s {
		switch foundCounter {
		case 1:
			if r == 'h' || r == 'H' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 2:
			if r == 't' || r == 'T' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 3:
			if r == 'm' || r == 'M' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 4:
			if r == 'l' || r == 'L' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 5:
			if r == '>' || r == ' ' {
				// found "<html " or "<html>"
				return pos, nil
			}
		default:
			if r == '<' {
				pos = i
				foundCounter = 1
			} else {
				foundCounter = 0
			}
		}
		if maxpos > 0 && i > maxpos {
			return 0, errNoHTML // no <html> tag for the first maxpos runes
		}
		i++
	}
	return 0, errNoHTML
}

func HasHTMLTagString(s string, maxpos uint64) bool {
	var (
		foundCounter uint8
		i            uint64
	)
	for _, r := range s {
		switch foundCounter {
		case 1:
			if r == 'h' || r == 'H' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 2:
			if r == 't' || r == 'T' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 3:
			if r == 'm' || r == 'M' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 4:
			if r == 'l' || r == 'L' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 5:
			if r == '>' || r == ' ' {
				// found "<html " or "<html>"
				return true
			} else {
				foundCounter = 0
			}
		default:
			if r == '<' {
				foundCounter = 1
			} else {
				foundCounter = 0
			}
		}
		if maxpos > 0 && i > maxpos {
			return false // no <html> tag for the first maxpos runes
		}
		i++
	}
	return false
}

func HasScriptTagString(s string, maxpos uint64) bool {
	var (
		foundCounter uint8
		i            uint64
	)
	for _, r := range s {
		switch foundCounter {
		case 1:
			if r == 's' || r == 'S' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 2:
			if r == 'c' || r == 'C' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 3:
			if r == 'r' || r == 'R' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 4:
			if r == 'i' || r == 'I' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 5:
			if r == 'p' || r == 'P' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 6:
			if r == 't' || r == 'T' {
				foundCounter++
			} else {
				foundCounter = 0
			}
		case 7:
			if r == '>' || r == ' ' {
				// found "<script " or "<script>
				return true
			}
		default:
			if r == '<' {
				foundCounter = 1
			} else {
				foundCounter = 0
			}
		}
		if maxpos > 0 && i > maxpos {
			return false // no <script> tag for the first maxpos runes
		}
		i++
	}
	return false
}

func GetHTMLTag(data []byte, maxpos uint64) ([]byte, error) {
	pos, err := HTMLIndex(data, maxpos)
	if err != nil {
		return nil, err
	}
	var collected []byte
	collected = append(collected, '<')
	for _, r := range data[pos+1:] {
		if r == '>' { // encountered the end of the tag
			collected = append(collected, r)
			break
		} else if r == '<' { // encountered another tag
			break
		}
		// continue collecting the contents of the <html ...> tag
		collected = append(collected, r)
	}
	return collected, nil
}

func GetHTMLTagString(s string, maxpos uint64) (string, error) {
	pos, err := HTMLIndexString(s, maxpos)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.WriteRune('<')
	for _, r := range s[pos+1:] {
		if r == '>' { // encountered the end of the tag
			sb.WriteRune(r)
			break
		} else if r == '<' { // encountered another tag
			break
		}
		// continue collecting the contents of the <html ...> tag
		sb.WriteRune(r)
	}
	return sb.String(), nil
}
