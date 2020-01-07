package recwatch

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xyproto/symwalk"
)

// ShouldIgnoreFile determines if a file should be ignored.
// File names that begin with "." or "_" are ignored by the go tool.
func ShouldIgnoreFile(name string) bool {
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}

// Subfolders returns a slice of subfolders (recursive), including the provided path
func Subfolders(path string) (paths []string) {
	var mut sync.Mutex
	symwalk.Walk(path, func(newPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			name := info.Name()
			// Skip folders that begin with "_" or "."
			if ShouldIgnoreFile(name) {
				return filepath.SkipDir // returns to filepath.Walk
			}
			mut.Lock()
			paths = append(paths, newPath)
			mut.Unlock()
		}
		return nil // returns to filepath.Walk
	})
	return paths // return collected paths
}
