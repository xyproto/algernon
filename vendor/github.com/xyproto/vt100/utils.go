package vt100

// For each element in a slice, apply the function f
func mapSB(sl []string, f func(string) byte) []byte {
	result := make([]byte, len(sl))
	for i, s := range sl {
		result[i] = f(s)
	}
	return result
}

// For each element in a slice, apply the function f
func mapBS(bl []byte, f func(byte) string) []string {
	result := make([]string, len(bl))
	for i, b := range bl {
		result[i] = f(b)
	}
	return result
}

// umin finds the smallest uint
func umin(a, b uint) uint {
	if a < b {
		return a
	}
	return b
}
