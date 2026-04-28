package sqlite3_wrap

import (
	"math"
	"reflect"
	"unsafe"

	"golang.org/x/sys/windows"
)

type Memory struct {
	Buf []byte
	Max int64
	com int
	ptr uintptr
}

func (m *Memory) Slice() *[]byte {
	return &m.Buf
}

func (m *Memory) Grow(delta, _ int64) int64 {
	if m.Buf == nil {
		m.allocate(uint64(m.Max) << 16)
	}

	len := len(m.Buf)
	old := int64(len >> 16)
	if delta == 0 {
		return old
	}
	new := old + delta
	if new > m.Max {
		return -1
	}
	m.reallocate(uint64(new) << 16)
	return old
}

func (m *Memory) allocate(max uint64) {
	// Round up to the page size.
	rnd := uint64(windows.Getpagesize() - 1)
	res := (max + rnd) &^ rnd

	if res > math.MaxInt {
		// This ensures uintptr(res) overflows to a large value,
		// and windows.VirtualAlloc returns an error.
		res = math.MaxUint64
	}

	// Reserve res bytes of address space, to ensure we won't need to move it.
	r, err := windows.VirtualAlloc(0, uintptr(res), windows.MEM_RESERVE, windows.PAGE_READWRITE)
	if err != nil {
		panic(err)
	}
	m.ptr = r

	// SliceHeader, although deprecated, avoids a go vet warning.
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&m.Buf))
	sh.Data = r
	sh.Len = 0
	sh.Cap = int(res)
}

func (m *Memory) reallocate(size uint64) {
	com := uint64(m.com)
	res := uint64(cap(m.Buf))
	if com < size && size <= res {
		// Grow geometrically, round up to the page size.
		rnd := uint64(windows.Getpagesize() - 1)
		new := com + com>>3
		new = min(max(size, new), res)
		new = (new + rnd) &^ rnd

		// Commit additional memory up to new bytes.
		_, err := windows.VirtualAlloc(m.ptr, uintptr(new), windows.MEM_COMMIT, windows.PAGE_READWRITE)
		if err != nil {
			panic(err)
		}
		m.com = int(new)
	}
	m.Buf = m.Buf[:size]
}

func (m *Memory) Close() error {
	err := windows.VirtualFree(m.ptr, 0, windows.MEM_RELEASE)
	m.Buf = nil
	m.com = 0
	m.ptr = 0
	return err
}
