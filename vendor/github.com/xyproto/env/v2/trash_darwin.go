//go:build darwin

package env

import "path/filepath"

// TrashPath returns the current user's trash directory on macOS.
func TrashPath() string {
	return filepath.Join(HomeDir(), ".Trash")
}
