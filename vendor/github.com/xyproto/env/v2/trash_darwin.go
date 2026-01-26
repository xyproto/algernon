//go:build !linux && !windows && !plan9

package env

import "path/filepath"

// TrashPath returns the current user's trash directory on macOS, and also on other platforms
func TrashPath() string {
	return filepath.Join(HomeDir(), ".Trash")
}
