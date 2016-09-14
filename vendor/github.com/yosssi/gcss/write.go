package gcss

import (
	"bufio"
	"io"
	"os"
)

// writeFlusher is the interface that groups the basic Write and Flush methods.
type writeFlusher interface {
	io.Writer
	Flush() error
}

var newBufWriter = func(w io.Writer) writeFlusher {
	return bufio.NewWriter(w)
}

// write writes the input byte data to the CSS file.
func write(path string, bc <-chan []byte, berrc <-chan error) (<-chan struct{}, <-chan error) {
	done := make(chan struct{})
	errc := make(chan error)

	go func() {
		f, err := os.Create(path)

		if err != nil {
			errc <- err
			return
		}

		defer f.Close()

		w := newBufWriter(f)

		for {
			select {
			case b, ok := <-bc:
				if !ok {
					if err := w.Flush(); err != nil {
						errc <- err
						return
					}

					done <- struct{}{}

					return
				}

				if _, err := w.Write(b); err != nil {
					errc <- err
					return
				}
			case err := <-berrc:
				errc <- err
				return
			}
		}
	}()

	return done, errc
}
