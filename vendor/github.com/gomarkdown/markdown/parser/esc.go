package parser

// isEscape reports whether byte i follows an odd-length backslash run.
func isEscape(data []byte, i int) bool {
	start := i
	for start > 0 && data[start-1] == '\\' {
		start--
	}
	return (i-start)%2 != 0
}
