package textutil

// SkipDigits advances i past the ASCII digits at data[i:].
func SkipDigits(data []byte, i int) int {
	for i < len(data) && '0' <= data[i] && data[i] <= '9' {
		i++
	}
	return i
}
