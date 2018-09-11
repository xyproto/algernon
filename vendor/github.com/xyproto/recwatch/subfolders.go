package recwatch

import (
	"os"
	"path/filepath"
	"strings"
)

// Subfolders returns a slice of subfolders (recursive), including the provided path
func Subfolders(path string) (paths []string) {
	filepath.Walk(path, func(newPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			name := info.Name()
			// Skip folders that begin with "_" or "."
			if ShouldIgnoreFile(name) {
				return filepath.SkipDir // returns to filepath.Walk
			}
			paths = append(paths, newPath)
		}
		return nil // returns to filepath.Walk
	})
	return paths // return collected paths
}

// ShouldIgnoreFile determines if a file should be ignored.
// File names that begin with "." or "_" are ignored by the go tool.
func ShouldIgnoreFile(name string) bool {
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}
