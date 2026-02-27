package vt

// umin returns the smaller of two uint values
func umin(a, b uint) uint {
	if a < b {
		return a
	}
	return b
}
