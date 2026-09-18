// Package textutil contains text transformations shared across Markdown packages.
package textutil

// Slugify converts ASCII letters and digits into a URL fragment, replacing
// each run of other bytes with one hyphen and trimming edge hyphens.
func Slugify(in []byte) []byte {
	if len(in) == 0 {
		return in
	}
	out := make([]byte, 0, len(in))
	separator := true
	for _, char := range in {
		alphanumeric := '0' <= char && char <= '9' ||
			'a' <= char && char <= 'z' || 'A' <= char && char <= 'Z'
		if alphanumeric {
			out = append(out, char)
			separator = false
		} else if !separator {
			out = append(out, '-')
			separator = true
		}
	}
	if len(out) > 0 && out[len(out)-1] == '-' {
		out = out[:len(out)-1]
	}
	return out
}
