//go:build !plan9

package files

import "os"

// DataReadyOnStdin checks if data is ready on stdin
func DataReadyOnStdin() bool {
	fileInfo, err := os.Stdin.Stat()
	return err == nil && !(fileInfo.Mode()&os.ModeNamedPipe == 0)
}
