//go:build sqlite3_dotlk

package vfs

import (
	"errors"
	"io/fs"
	"sync"

	"github.com/ncruces/go-sqlite3/internal/dotlk"
	"github.com/ncruces/go-sqlite3/internal/sqlite3_wrap"
)

type vfsShmParent struct {
	shared [][_WALINDEX_PGSZ]byte // +checklocks:Mutex
	refs   int                    // +checklocks:vfsShmListMtx

	lock [_SHM_NLOCK]int8 // +checklocks:Mutex
	sync.Mutex

	ioLock vfsShmIOLock
}

var (
	// +checklocks:vfsShmListMtx
	vfsShmList    = map[string]*vfsShmParent{}
	vfsShmListMtx sync.Mutex
)

type vfsShm struct {
	*vfsShmParent
	wrp    *sqlite3_wrap.Wrapper
	path   string
	ptrs   []ptr_t
	shadow [_WALINDEX_HDR_SIZE]byte
	lock   [_SHM_NLOCK]bool
}

func (s *vfsShm) Close() error {
	if s.vfsShmParent == nil {
		return nil
	}

	vfsShmListMtx.Lock()
	defer vfsShmListMtx.Unlock()

	// Unlock everything.
	s.shmLock(0, _SHM_NLOCK, _SHM_UNLOCK)

	// Decrease reference count.
	if s.vfsShmParent.refs > 0 {
		s.vfsShmParent.refs--
		s.vfsShmParent = nil
		return nil
	}

	s.vfsShmParent = nil
	delete(vfsShmList, s.path)
	if err := dotlk.Unlock(s.path); err != nil {
		return sysError{err, _IOERR_UNLOCK}
	}
	return nil
}

func (s *vfsShm) shmOpen() error {
	if s.vfsShmParent != nil {
		return nil
	}

	vfsShmListMtx.Lock()
	defer vfsShmListMtx.Unlock()

	// Find a shared buffer, increase the reference count.
	if g, ok := vfsShmList[s.path]; ok {
		s.vfsShmParent = g
		g.refs++
		return nil
	}

	// Dead man's switch.
	err := dotlk.LockShm(s.path)
	if errors.Is(err, fs.ErrExist) {
		return _BUSY
	}
	if err != nil {
		return sysError{err, _IOERR_LOCK}
	}

	// Add the new shared buffer.
	s.vfsShmParent = &vfsShmParent{}
	vfsShmList[s.path] = s.vfsShmParent
	return nil
}

func (s *vfsShm) shmMap(wrp *sqlite3_wrap.Wrapper, id, size int32, extend bool) (ptr_t, error) {
	if size != _WALINDEX_PGSZ {
		return 0, _IOERR_SHMMAP
	}
	if s.wrp == nil {
		s.wrp = wrp
	}
	if err := s.shmOpen(); err != nil {
		return 0, err
	}

	s.Lock()
	defer s.Unlock()
	defer s.shmAcquire(nil, s.lock[_WAL_WRITE_LOCK])

	// Extend shared memory.
	if int(id) >= len(s.shared) {
		if !extend {
			return 0, nil
		}
		s.shared = append(s.shared, make([][_WALINDEX_PGSZ]byte, int(id)-len(s.shared)+1)...)
	}

	// Allocate local memory, initialized from shared memory.
	for int(id) >= len(s.ptrs) {
		ptr := wrp.Xsqlite3_malloc64(int64(size))
		if ptr == 0 {
			return 0, _IOERR_NOMEM
		}
		copy(wrp.Bytes(ptr_t(ptr), _WALINDEX_PGSZ), s.shared[len(s.ptrs)][:])
		s.ptrs = append(s.ptrs, ptr_t(ptr))
	}

	return s.ptrs[id], nil
}

func (s *vfsShm) shmLock(offset, n int32, flags _ShmFlag) (err error) {
	if s.vfsShmParent == nil {
		return _IOERR_SHMLOCK
	}

	s.Lock()
	defer s.Unlock()
	writer := s.lock[_WAL_WRITE_LOCK]

	switch {
	case flags&_SHM_LOCK != 0:
		defer s.shmAcquire(&err, writer)
	case flags&_SHM_EXCLUSIVE != 0:
		s.shmRelease(writer)
	}

	return s.shmMemLock(offset, n, flags)
}

func (s *vfsShm) shmUnmap(delete bool) {
	if s.vfsShmParent == nil {
		return
	}
	defer s.Close()

	s.Lock()
	s.shmRelease(s.lock[_WAL_WRITE_LOCK])
	defer s.Unlock()

	for _, p := range s.ptrs {
		s.wrp.Xsqlite3_free(int32(p))
	}
	s.ptrs = nil
	clear(s.shadow[:])
}

// The WAL-index is private to this process: no file locks are needed.

func (*vfsShmParent) ioLockFile(LockLevel, int64) error { return nil }
