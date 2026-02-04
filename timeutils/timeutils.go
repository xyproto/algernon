package timeutils

import (
	"strconv"
	"time"
)

// DurationToMS converts time.Duration to milliseconds, as a string,
// (just the number as a string, no "ms" suffix).
func DurationToMS(d time.Duration, multiplier float64) string {
	return strconv.Itoa(int(d.Seconds() * 1000.0 * multiplier))
}
