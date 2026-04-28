//go:build !unix && !windows

package sqlite3_wrap

type Memory struct {
	Buf []byte
	Max int64
}

func (m *Memory) Slice() *[]byte {
	return &m.Buf
}

func (m *Memory) Grow(delta, _ int64) int64 {
	len := len(m.Buf)
	old := int64(len >> 16)
	if delta == 0 {
		return old
	}
	new := old + delta
	add := int(new)<<16 - len
	if new > m.Max || add < 0 {
		return -1
	}
	m.Buf = append(m.Buf, make([]byte, add)...)
	return old
}

func (m *Memory) Close() error {
	m.Buf = nil
	return nil
}
