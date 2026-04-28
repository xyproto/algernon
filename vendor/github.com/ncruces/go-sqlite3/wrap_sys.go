package sqlite3

import (
	"io"

	"github.com/ncruces/go-sqlite3/internal/testenv"
)

func (e *env) Xexit(c int32) {
	testenv.Exit(c)
}

func (e *env) Xsystem(ptr int32) int32 {
	return testenv.System(e.Wrapper, ptr)
}

func (e *env) Xputs(ptr int32) int32 {
	if testenv.TB == nil {
		return -1
	}
	s := e.ReadString(ptr_t(ptr), _MAX_NAME)
	testenv.WriteString(s)
	testenv.WriteByte('\n')
	return 0
}

func (e *env) Xfclose(h int32) int32 {
	if testenv.TB == nil {
		return -1
	}
	if e.DelHandle(ptr_t(h)) != nil {
		return -1
	}
	return 0
}

func (e *env) Xfopen(path, mode int32) int32 {
	if testenv.TB == nil {
		return 0
	}
	p := e.ReadString(ptr_t(path), _MAX_NAME)
	f, err := testenv.FS.Open(p)
	if err != nil {
		return 0
	}
	return int32(e.AddHandle(f))
}

func (e *env) Xfflush(h int32) int32 {
	if testenv.TB == nil {
		return -1
	}
	return 0
}

func (e *env) Xfputc(c, h int32) int32 {
	if testenv.TB == nil {
		return -1
	}
	if testenv.WriteByte(byte(c)) != nil {
		return -1
	}
	return 0
}

func (e *env) Xfwrite(ptr, sz, cnt, h int32) int32 {
	if testenv.TB == nil {
		return 0
	}
	b := e.Buf[ptr:][:sz*cnt]
	n, _ := testenv.Write(b)
	return int32(n / int(sz))
}

func (e *env) Xfread(ptr, sz, cnt, h int32) int32 {
	if testenv.TB == nil {
		return 0
	}
	f := e.GetHandle(ptr_t(h)).(io.Reader)
	b := e.Buf[ptr:][:sz*cnt]
	n, _ := f.Read(b)
	return int32(n / int(sz))
}

func (e *env) Xftell(h int32) int32 {
	if testenv.TB == nil {
		return -1
	}
	f := e.GetHandle(ptr_t(h)).(io.Seeker)
	n, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return -1
	}
	return int32(n)
}

func (e *env) Xfseek(h, offset, whence int32) int32 {
	if testenv.TB == nil {
		return -1
	}
	f := e.GetHandle(ptr_t(h)).(io.Seeker)
	_, err := f.Seek(int64(offset), int(whence))
	if err != nil {
		return -1
	}
	return 0
}
