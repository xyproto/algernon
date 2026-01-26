//go:build plan9

package env

import "path/filepath"

// TrashPath returns the current user's trash directory (the least offensive path, if plan9 were to have one)
func TrashPath() string {
	return filepath.Join(HomeDir(), "trash")
}
