package internal

import (
	"fmt"
	"io"
	"os"
)

// NewFileWriter returns a io.WriteCloser.
func NewFileWriter(name string) io.WriteCloser {
	return &fileWriter{name: name, firstWrite: true}
}

type fileWriter struct {
	name       string
	firstWrite bool
	file       *os.File
	err        error
}

// Close implements io.Writer.
func (f *fileWriter) Write(p []byte) (int, error) {
	if f.firstWrite {
		f.firstWrite = false
		f.file, f.err = os.Create(f.name)
	}
	if f.err != nil {
		return 0, f.err
	}
	return f.file.Write(p)
}

// Close implements io.Closer.
func (f *fileWriter) Close() error {
	if f.file != nil {
		return f.file.Close()
	}
	return f.err
}

// String implements fmt.Stringer.
func (f *fileWriter) String() string {
	return f.name
}

// GoString implements fmt.GoStringer.
func (f *fileWriter) GoString() string {
	return fmt.Sprintf("*%#v", *f)
}
