package vt

import "time"

// Timeout returns the configured read timeout for the TTY.
func (tty *TTY) Timeout() time.Duration {
	return tty.timeout
}
