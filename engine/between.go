package engine

import "strings"

// Return what's between two strings, "a" and "b", in another string,
// but inclusively, so that "a" and "b" are also included in the return value.
func betweenInclusive(orig, a, b string) string {
	if strings.Contains(orig, a) && strings.Contains(orig, b) {
		posa := strings.Index(orig, a) + len(a)
		posb := strings.LastIndex(orig, b)
		if posa > posb {
			return ""
		}
		return a + orig[posa:posb] + b
	}
	return ""
}
