package internal

import "io"

var _ io.Writer = ErrWriter{}

type ErrWriter struct {
	Err error
}

func (e ErrWriter) Write(p []byte) (int, error) {
	return 0, e.Err
}
