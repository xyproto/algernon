//go:build (386 || arm || amd64 || arm64 || riscv64 || ppc64le || loong64) && !sqlite3_dotlk

package vfs

import (
	"io"
	"os"
	"sync"

	"golang.org/x/sys/windows"

	"github.com/ncruces/go-sqlite3/internal/errutil"
	"github.com/ncruces/go-sqlite3/internal/sqlite3_wrap"
)

type vfsShm struct {
	*os.File
	wrp      *sqlite3_wrap.Wrapper
	path     string
	regions  []*sqlite3_wrap.MappedRegion
	shared   [][]byte
	shadow   [][_WALINDEX_PGSZ]byte
	ptrs     []ptr_t
	fileLock bool
	sync.Mutex
}

func (s *vfsShm) Close() error {
	// Unmap regions.
	for _, r := range s.regions {
		r.Unmap()
	}
	s.regions = nil

	// Close the file.
	return s.File.Close()
}

func (s *vfsShm) shmOpen() error {
	if s.fileLock {
		return nil
	}
	if s.File == nil {
		f, err := os.OpenFile(s.path, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			return sysError{err, _CANTOPEN}
		}
		s.fileLock = false
		s.File = f
	}

	// Dead man's switch.
	if osWriteLock(s.File, _SHM_DMS, 1, 0) == nil {
		err := s.Truncate(0)
		osUnlock(s.File, _SHM_DMS, 1)
		if err != nil {
			return sysError{err, _IOERR_SHMOPEN}
		}
	}
	err := osReadLock(s.File, _SHM_DMS, 1, 0)
	s.fileLock = err == nil
	return err
}

func (s *vfsShm) shmMap(wrp *sqlite3_wrap.Wrapper, id, size int32, extend bool) (_ ptr_t, err error) {
	// Ensure size is a multiple of the OS page size.
	if size != _WALINDEX_PGSZ || (windows.Getpagesize()-1)&_WALINDEX_PGSZ != 0 {
		return 0, _IOERR_SHMMAP
	}
	if s.wrp == nil {
		s.wrp = wrp
	}
	if err := s.shmOpen(); err != nil {
		return 0, err
	}

	defer s.shmAcquire(&err)

	// Check if file is big enough.
	o, err := s.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, sysError{err, _IOERR_SHMSIZE}
	}
	if n := (int64(id) + 1) * int64(size); n > o {
		if !extend {
			return 0, nil
		}
		if err := osAllocate(s.File, n); err != nil {
			return 0, sysError{err, _IOERR_SHMSIZE}
		}
	}

	// Maps regions into memory.
	for int(id) >= len(s.shared) {
		r, err := sqlite3_wrap.MapRegion(s.File, int64(id)*int64(size), size)
		if err != nil {
			return 0, err
		}
		s.regions = append(s.regions, r)
		s.shared = append(s.shared, r.Data)
	}

	// Allocate shadow memory.
	if int(id) >= len(s.shadow) {
		s.shadow = append(s.shadow, make([][_WALINDEX_PGSZ]byte, int(id)-len(s.shadow)+1)...)
	}

	// Allocate local memory.
	for int(id) >= len(s.ptrs) {
		ptr := wrp.Xsqlite3_malloc64(int64(size))
		if ptr == 0 {
			panic(errutil.OOMErr)
		}
		clear(wrp.Bytes(ptr_t(ptr), _WALINDEX_PGSZ))
		s.ptrs = append(s.ptrs, ptr_t(ptr))
	}

	s.shadow[0][4] = 1
	return s.ptrs[id], nil
}

func (s *vfsShm) shmLock(offset, n int32, flags _ShmFlag) (err error) {
	if s.File == nil {
		return _IOERR_SHMLOCK
	}

	switch {
	case flags&_SHM_LOCK != 0:
		defer s.shmAcquire(&err)
	case flags&_SHM_EXCLUSIVE != 0:
		s.shmRelease()
	}

	switch {
	case flags&_SHM_UNLOCK != 0:
		return osUnlock(s.File, _SHM_BASE+uint32(offset), uint32(n))
	case flags&_SHM_SHARED != 0:
		return osReadLock(s.File, _SHM_BASE+uint32(offset), uint32(n), 0)
	case flags&_SHM_EXCLUSIVE != 0:
		return osWriteLock(s.File, _SHM_BASE+uint32(offset), uint32(n), 0)
	default:
		panic(errutil.AssertErr())
	}
}

func (s *vfsShm) shmUnmap(delete bool) {
	if s.File == nil {
		return
	}

	s.shmRelease()

	// Free local memory.
	for _, p := range s.ptrs {
		s.wrp.Xsqlite3_free(int32(p))
	}
	s.ptrs = nil
	s.shadow = nil
	s.shared = nil

	// Close the file.
	s.Close()
	s.File = nil
	s.fileLock = false
	if delete {
		os.Remove(s.path)
	}
}
