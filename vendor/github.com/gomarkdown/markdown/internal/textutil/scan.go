package textutil

// Concat returns prefix followed by b in one allocation. append([]byte(prefix), b...)
// allocates the prefix and then a second slice once b makes it grow.
func Concat(prefix string, b []byte) []byte {
	out := make([]byte, len(prefix)+len(b))
	copy(out, prefix)
	copy(out[len(prefix):], b)
	return out
}

// ByteAt reports whether data[i] exists and is c.
func ByteAt(data []byte, i int, c byte) bool {
	return i >= 0 && i < len(data) && data[i] == c
}

// SkipWhile advances i as long as keep(data[i]) holds.
func SkipWhile(data []byte, i int, keep func(byte) bool) int {
	for i < len(data) && keep(data[i]) {
		i++
	}
	return i
}

// SkipChar advances i as long as data[i] == c.
func SkipChar(data []byte, i int, c byte) int {
	return SkipWhile(data, i, func(b byte) bool { return b == c })
}

// SkipCharN is like SkipChar but skips at most max bytes.
func SkipCharN(data []byte, i int, c byte, max int) int {
	for i < len(data) && max > 0 && data[i] == c {
		i++
		max--
	}
	return i
}

// SkipDigits advances i past the ASCII digits at data[i:].
func SkipDigits(data []byte, i int) int {
	return SkipWhile(data, i, func(b byte) bool { return '0' <= b && b <= '9' })
}

// BackChar moves i back as long as data[i-1] == c.
func BackChar(data []byte, i int, c byte) int {
	for i > 0 && data[i-1] == c {
		i--
	}
	return i
}
