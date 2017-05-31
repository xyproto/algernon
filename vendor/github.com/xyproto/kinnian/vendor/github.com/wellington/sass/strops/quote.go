package strops

import (
	"strconv"
	"strings"
)

func Unquote(in string) string {
	return unescape(in)
}

const (
	sassEscape = `\`
	goEscape   = `\u`
	quote      = `"`
)

// unquote converts Sass's bizarre unicode escape format to valid
// unicode text
func unescape(in string) string {
	ss := strings.Split(in, sassEscape)
	// No sass unicode
	if len(ss) == 1 {
		return in
	}
	// Attempt unquote on each Sass escape found
	for i, s := range ss {
		if len(s) == 0 {
			continue
		}
		in := quote + goEscape + s + quote
		unq, err := strconv.Unquote(in)
		// if unquote was successful, replace
		if err == nil {
			ss[i] = unq
		}
	}

	return strings.Join(ss, "")
}
