//go:build windows

package env

import (
	"path/filepath"
	"strings"
)

// oneSuffix makes sure that the given path ends with one (and only one) path separator
func oneSuffix(path string) string {
	pSep := string(filepath.Separator)
	return strings.TrimRight(path, pSep) + pSep
}

// TrashPath returns the current user's recycle bin container on Windows.
// This is the per-volume $Recycle.Bin folder on the user's home drive.
func TrashPath() string {
	drive := filepath.VolumeName(HomeDir())
	if drive == "" {
		drive = Str("SystemDrive", "C:")
	}
	if drive == "" {
		drive = "C:"
	}
	root := oneSuffix(drive)
	return filepath.Join(root, "$Recycle.Bin")
}
