// Package recwatch provides a way to watch directories recursively, using fsnotify
package recwatch

import (
	"errors"

	"github.com/fsnotify/fsnotify"
)

// RecursiveWatcher keeps the data for watching files and directories
type RecursiveWatcher struct {
	*fsnotify.Watcher
	Files   chan string
	Folders chan string
}

// NewRecursiveWatcher creates a new RecursiveWatcher.
// Takes a path to a directory to watch.
func NewRecursiveWatcher(path string) (*RecursiveWatcher, error) {
	folders := Subfolders(path)
	if len(folders) == 0 {
		return nil, errors.New("No directories to watch.")
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

// AddFolder adds a directory to watch, non-recursively
func (watcher *RecursiveWatcher) AddFolder(folder string) error {
	if err := watcher.Add(folder); err != nil {
		return err
	}
	watcher.Folders <- folder
	return nil
}
