package timeutils

import (
	"testing"
	"time"
)

func TestDurationToMS(t *testing.T) {
	tests := []struct {
		d          time.Duration
		multiplier float64
		want       string
	}{
		{time.Second, 1.0, "1000"},
		{500 * time.Millisecond, 1.0, "500"},
		{time.Second, 2.0, "2000"},
		{0, 1.0, "0"},
		{100 * time.Millisecond, 0.5, "50"},
	}
	for _, tt := range tests {
		got := DurationToMS(tt.d, tt.multiplier)
		if got != tt.want {
			t.Errorf("DurationToMS(%v, %v) = %q, want %q", tt.d, tt.multiplier, got, tt.want)
		}
	}
}
