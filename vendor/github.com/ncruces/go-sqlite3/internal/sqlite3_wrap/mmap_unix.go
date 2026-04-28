//go:build unix

package sqlite3_wrap

import (
	"os"
	"unsafe"

	"github.com/ncruces/go-sqlite3/internal/errutil"
	"golang.org/x/sys/unix"
)

type mmapState struct {
	regions []*MappedRegion
}

func (w *Wrapper) MapRegion(f *os.File, offset int64, size int32, readOnly bool) (*MappedRegion, error) {
	r := w.newRegion(size)
	err := r.mmap(f, offset, readOnly)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (w *Wrapper) newRegion(size int32) *MappedRegion {
	// Find unused region.
	for _, r := range w.regions {
		if !r.used && r.size == size {
			return r
		}
	}

	// Allocate page aligned memmory.
	ptr := Ptr_t(w.Xaligned_alloc(int32(unix.Getpagesize()), size))
	if ptr == 0 {
		panic(errutil.OOMErr)
	}

	// Save the newly allocated region.
	buf := w.Bytes(ptr, int64(size))
	ret := &MappedRegion{
		Ptr:  ptr,
		size: size,
		addr: unsafe.Pointer(&buf[0]),
	}
	w.regions = append(w.regions, ret)
	return ret
}

type MappedRegion struct {
	addr unsafe.Pointer
	Ptr  Ptr_t
	size int32
	used bool
}

func (r *MappedRegion) Unmap() error {
	// We can't munmap the region, otherwise it could be remaped by the runtime.
	// We shouldn't create a hole, because unaligned reads might fail.
	// Instead remap it readonly, and if successful,
	// it can be reused for a subsequent mmap.
	_, err := unix.MmapPtr(-1, 0, r.addr, uintptr(r.size),
		unix.PROT_READ, unix.MAP_PRIVATE|unix.MAP_FIXED|unix.MAP_ANON)
	r.used = err != nil
	return err
}

func (r *MappedRegion) mmap(f *os.File, offset int64, readOnly bool) error {
	prot := unix.PROT_READ
	if !readOnly {
		prot |= unix.PROT_WRITE
	}
	_, err := unix.MmapPtr(int(f.Fd()), offset, r.addr, uintptr(r.size),
		prot, unix.MAP_SHARED|unix.MAP_FIXED)
	r.used = err == nil
	return err
}
