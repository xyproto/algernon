package recwatch

import (
	"github.com/fsnotify/fsnotify"
)

// Event is a custom type, so that packages that depends on recwatch does not also have to import fstnotify
type Event fsnotify.Event

func (e Event) String() string {
	return fsnotify.Event(e).String()
}
