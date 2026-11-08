package engine

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/xyproto/algernon/utils"
)

const deadlineStep = utils.MiB

type deadlineWriter struct {
	http.ResponseWriter
	rc      *http.ResponseController
	timeout time.Duration
	left    int64
}

func (dw *deadlineWriter) progress(n int64) error {
	dw.left -= n
	if dw.left > 0 {
		return nil
	}
	dw.left = deadlineStep
	if err := dw.rc.SetWriteDeadline(time.Now().Add(dw.timeout)); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return err
	}
	return nil
}

func (dw *deadlineWriter) Write(p []byte) (int, error) {
	var total int
	for len(p) > 0 {
		k := min(int64(len(p)), dw.left)
		n, err := dw.ResponseWriter.Write(p[:k])
		total += n
		if err != nil {
			return total, err
		}
		if err := dw.progress(int64(n)); err != nil {
			return total, err
		}
		p = p[n:]
	}
	return total, nil
}

func (dw *deadlineWriter) ReadFrom(src io.Reader) (int64, error) {
	rf, ok := dw.ResponseWriter.(io.ReaderFrom)
	if !ok {
		return io.Copy(struct{ io.Writer }{dw}, src)
	}
	r, remaining := src, int64(-1)
	if lr, ok := src.(*io.LimitedReader); ok {
		r, remaining = lr.R, lr.N
		defer func() { lr.N = remaining }()
	}
	var total int64
	for remaining != 0 {
		chunk := dw.left
		if remaining > 0 {
			chunk = min(chunk, remaining)
		}
		n, err := rf.ReadFrom(&io.LimitedReader{R: r, N: chunk})
		total += n
		if remaining > 0 {
			remaining -= n
		}
		if err != nil {
			return total, err
		}
		if err := dw.progress(n); err != nil {
			return total, err
		}
		if n < chunk {
			break
		}
	}
	return total, nil
}

func newDeadlineWriter(w http.ResponseWriter, req *http.Request) http.ResponseWriter {
	srv, ok := req.Context().Value(http.ServerContextKey).(*http.Server)
	if !ok || srv.WriteTimeout <= 0 {
		return w
	}
	inner := w
	if sc, ok := w.(interface{ ResponseWriter() http.ResponseWriter }); ok {
		inner = sc.ResponseWriter()
	}
	return &deadlineWriter{
		ResponseWriter: w,
		rc:             http.NewResponseController(inner),
		timeout:        srv.WriteTimeout,
		left:           deadlineStep,
	}
}
