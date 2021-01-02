package term

import (
	"errors"
)

type Term struct {
}

var errNotSupported = errors.New("not supported")

// Open opens an asynchronous communications port.
func Open(name string, options ...func(*Term) error) (*Term, error) {
	return nil, errNotSupported
}
