package util

import (
	"bytes"
	"cmp"
	"math"
	"strconv"
)

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func GCD(m, n int) int {
	for n != 0 {
		m, n = n, m%n
	}
	return abs(m)
}

func LCM(m, n int) int {
	if n == 0 {
		return 0
	}
	return abs(n) * (abs(m) / GCD(m, n))
}

// https://developer.nvidia.com/blog/lerp-faster-cuda/
func Lerp(v0, v1, t float64) float64 {
	return math.FMA(t, v1, math.FMA(-t, v0, v0))
}

func FloatIsInt(f float64) (int64, bool) {
	// math.MaxInt64 is not exactly representable as a float64 (it rounds to 2^63),
	// but int64(2^63) may saturate back to math.MaxInt64.
	i := int64(f)
	return i, float64(i) == f && i != math.MaxInt64
}

func CmpIntFloat(i int64, f float64) int {
	// Assumes both FloatIsInt(f) and IsNaN(f) are false.
	if c := cmp.Compare(float64(i), f); c != 0 {
		return c
	}
	return -1 // float64(i) == f means f == 2^63
}

func AppendNumber(dst []byte, f float64, bits int) []byte {
	if i, ok := FloatIsInt(f); ok {
		return strconv.AppendInt(dst, i, 10)
	}
	return AppendFloat(dst, f, bits)
}

func AppendFloat(dst []byte, f float64, bits int) []byte {
	switch {
	case math.IsNaN(f):
		return append(dst, "null"...)
	case math.IsInf(f, 1):
		return append(dst, "9.0e999"...)
	case math.IsInf(f, -1):
		return append(dst, "-9.0e999"...)
	default:
		i := len(dst)
		dst = strconv.AppendFloat(dst, f, 'g', -1, bits)
		if !bytes.ContainsAny(dst[i:], ".e") {
			dst = append(dst, ".0"...)
		}
		return dst
	}
}
