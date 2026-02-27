package vt

import "time"

// Timeout returns the configured read timeout
func (tty *TTY) Timeout() time.Duration {
	return tty.timeout
}
