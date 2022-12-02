package internal

import "io"

// WriteNopCloser returns an io.WriteCloser that writes to w and does nothing
// when Close() is called. Go's stdlib unfortunately only provides io.NopCloser
// for io.Reader, see: https://github.com/golang/go/issues/22823
func WriteNopCloser(w io.Writer) io.WriteCloser {
	return writeNopeCloser{w}
}

type writeNopeCloser struct{ io.Writer }

func (nc writeNopeCloser) Close() error { return nil }
