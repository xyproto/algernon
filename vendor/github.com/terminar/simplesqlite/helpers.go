package simplesqlite

import (
	"strings"
)

var Verbose = false

/* --- Helper functions --- */

// Split a string into two parts, given a delimiter.
// Returns the two parts and true if it works out.
func twoFields(s, delim string) (string, string, bool) {
	if strings.Count(s, delim) != 1 {
		return s, "", false
	}
	fields := strings.Split(s, delim)
	return fields[0], fields[1], true
}

func leftOf(s, delim string) string {
	if left, _, ok := twoFields(s, delim); ok {
		return strings.TrimSpace(left)
	}
	return ""
}

func rightOf(s, delim string) string {
	if _, right, ok := twoFields(s, delim); ok {
		return strings.TrimSpace(right)
	}
	return ""
}
