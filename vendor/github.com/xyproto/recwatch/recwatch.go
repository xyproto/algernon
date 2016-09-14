// Package recwatch provides a way to watch directories recursively with fsnotify
package recwatch

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

type (
	Event fsnotify.Event

	RecursiveWatcher struct {
		*fsnotify.Watcher
		Files   chan string
		Folders chan string
	}
)

func (e Event) String() string {
	return fsnotify.Event(e).String()
}

func NewRecursiveWatcher(path string) (*RecursiveWatcher, error) {
	folders := Subfolders(path)
	if len(folders) == 0 {
		return nil, errors.New("No folders to watch.")
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	rw := &RecursiveWatcher{Watcher: watcher}

	rw.Files = make(chan string, 10)
	rw.Folders = make(chan string, len(folders))

	for _, folder := range folders {
		if err = rw.AddFolder(folder); err != nil {
			return nil, err
		}
	}
	return rw, nil
}

func (watcher *RecursiveWatcher) AddFolder(folder string) error {
	if err := watcher.Add(folder); err != nil {
		return err
	}
	watcher.Folders <- folder
	return nil
}

// Subfolders returns a slice of subfolders (recursive), including the folder provided.
func Subfolders(path string) (paths []string) {
	filepath.Walk(path, func(newPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			name := info.Name()
			// skip folders that begin with a dot
			if ShouldIgnoreFile(name) && name != "." && name != ".." {
				return filepath.SkipDir
			}
			paths = append(paths, newPath)
		}
		return nil
	})
	return paths
}

// ShouldIgnoreFile determines if a file should be ignored.
// File names that begin with "." or "_" are ignored by the go tool.
func ShouldIgnoreFile(name string) bool {
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}
