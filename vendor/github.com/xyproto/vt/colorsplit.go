package vt

import "strings"

// ColorSplit splits on the first sep in line. It returns two parts: left and right.
// The right part includes the sep itself (so subsequent splits see it).
// nil color funcs are skipped. reverse=true swaps which side gets the fallback when sep is absent.
func ColorSplit(line, sep string, headColor, sepColor, tailColor AttributeColor, reverse bool) (string, string) {
	if sep == "" {
		if reverse {
			return "", line
		}
		return line, ""
	}
	before, after, ok := strings.Cut(line, sep)
	if !ok {
		if reverse {
			return "", line
		}
		return line, ""
	}
	head := before
	tail := after
	var a, b string
	if reverse {
		if tailColor != 0 {
			a = tailColor.Get(tail)
		} else {
			a = tail
		}
		if sepColor != 0 {
			a = sepColor.Get(sep) + a
		} else {
			a = sep + a
		}
		if headColor != 0 {
			b = headColor.Get(head)
		} else {
			b = head
		}
	} else {
		if headColor != 0 {
			a = headColor.Get(head)
		} else {
			a = head
		}
		if sepColor != 0 {
			a += sepColor.Get(sep)
		} else {
			a += sep
		}
		if tailColor != 0 {
			b = tailColor.Get(tail)
		} else {
			b = tail
		}
	}
	return a, b
}
