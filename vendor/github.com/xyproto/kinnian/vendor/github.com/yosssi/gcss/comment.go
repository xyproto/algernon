package gcss

import "io"

// comment represents a comment of CSS.
type comment struct {
	elementBase
}

// WriteTo does nothing.
func (c *comment) WriteTo(w io.Writer) (int64, error) {
	return 0, nil
}

// newComment creates and returns a comment.
func newComment(ln *line, parent element) *comment {
	return &comment{
		elementBase: newElementBase(ln, parent),
	}
}
