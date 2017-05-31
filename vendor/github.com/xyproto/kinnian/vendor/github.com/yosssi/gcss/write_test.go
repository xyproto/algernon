package gcss

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

var errTest = errors.New("test error")

type writeErrBufWriter struct{}

func (w *writeErrBufWriter) Write(p []byte) (int, error) {
	return 0, errTest
}

func (w *writeErrBufWriter) Flush() error {
	return nil
}

type flushErrBufWriter struct{}

func (w *flushErrBufWriter) Write(p []byte) (int, error) {
	return 0, nil
}

func (w *flushErrBufWriter) Flush() error {
	return errTest
}

func Test_write_err(t *testing.T) {
	bc := make(chan []byte)
	berrc := make(chan error)

	done, errc := write("not_exist_dir/not_exist_file", bc, berrc)

	select {
	case <-done:
		t.Error("error should be occurred")
	case err := <-errc:
		if expected, actual := "open not_exist_dir/not_exist_file: ", err.Error(); !strings.HasPrefix(actual, expected) || !os.IsNotExist(err) {
			t.Errorf("err should be %q [actual: %q]", expected, actual)
		}
	}
}

func Test_write_writeErr(t *testing.T) {
	newBufWriterBak := newBufWriter

	defer func() {
		newBufWriter = newBufWriterBak
	}()

	newBufWriter = func(w io.Writer) writeFlusher {
		return &writeErrBufWriter{}
	}

	bc := make(chan []byte)
	berrc := make(chan error)

	done, errc := write("test/0008.gcss", bc, berrc)

	bc <- []byte("test")

	select {
	case <-done:
		t.Error("error should be occurred")
	case err := <-errc:
		if err != errTest {
			t.Errorf("err should be %q [actual: %q]", errTest.Error(), err.Error())
		}
	}
}

func Test_write_flushErr(t *testing.T) {
	newBufWriterBak := newBufWriter

	defer func() {
		newBufWriter = newBufWriterBak
	}()

	newBufWriter = func(w io.Writer) writeFlusher {
		return &flushErrBufWriter{}
	}

	bc := make(chan []byte)
	berrc := make(chan error)

	done, errc := write("test/0008.gcss", bc, berrc)

	close(bc)

	select {
	case <-done:
		t.Error("error should be occurred")
	case err := <-errc:
		if err != errTest {
			t.Errorf("err should be %q [actual: %q]", errTest.Error(), err.Error())
		}
	}
}
