//go:build linux

package env

import "path/filepath"

// TrashPath returns the current user's trash directory on Linux.
// If $XDG_DATA_HOME points to an existing directory, $XDG_DATA_HOME/Trash is used.
// Otherwise, ~/.local/share/Trash is returned.
func TrashPath() string {
	dataHome := ExpandUser(Str("XDG_DATA_HOME"))
	if dataHome != "" && isDir(dataHome) {
		return filepath.Join(dataHome, "Trash")
	}
	return filepath.Join(HomeDir(), ".local", "share", "Trash")
}
