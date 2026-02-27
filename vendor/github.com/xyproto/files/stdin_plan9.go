//go:build plan9

package files

import "os"

// DataReadyOnStdin checks if data is ready on stdin.
// On Plan 9, ModeNamedPipe is never set, so we check
// that stdin is not a character device instead.
func DataReadyOnStdin() bool {
	fileInfo, err := os.Stdin.Stat()
	return err == nil && fileInfo.Mode()&os.ModeCharDevice == 0
}
