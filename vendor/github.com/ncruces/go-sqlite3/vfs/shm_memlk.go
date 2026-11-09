//go:build freebsd || openbsd || netbsd || dragonfly || illumos || sqlite3_flock || sqlite3_dotlk

package vfs

import (
	"sync"

	"github.com/ncruces/go-sqlite3/internal/errutil"
)

// +checklocks:s.Mutex
func (s *vfsShm) shmMemLock(offset, n int32, flags _ShmFlag) error {
	p := s.vfsShmParent
	switch {
	case flags&_SHM_UNLOCK != 0:
		for i := offset; i < offset+n; i++ {
			if s.lock[i] {
				if p.lock[i] <= 0 {
					p.lock[i] = 0
				} else {
					p.lock[i]--
				}
				s.lock[i] = false
			}
		}
	case flags&_SHM_SHARED != 0:
		for i := offset; i < offset+n; i++ {
			if !s.lock[i] &&
				p.lock[i]+1 <= 0 {
				return _BUSY
			}
		}
		for i := offset; i < offset+n; i++ {
			if !s.lock[i] {
				s.lock[i] = true
				p.lock[i]++
			}
		}
	case flags&_SHM_EXCLUSIVE != 0:
		for i := offset; i < offset+n; i++ {
			if s.lock[i] {
				// SQLite never requests an exclusive lock that it already holds.
				panic(errutil.AssertErr())
			}
			if p.lock[i] != 0 {
				return _BUSY
			}
		}
		for i := offset; i < offset+n; i++ {
			p.lock[i] = -1
			s.lock[i] = true
		}
	default:
		panic(errutil.AssertErr())
	}
	return nil
}

// vfsShmIOLock implements striped I/O locks for
// WAL-indexes shared by all the connections in a process.
// If the WAL-index is also shared with other processes,
// the first reader (or the writer) additionally locks the file,
// and the last one unlocks it; those locks are per process.
//
// Each cnt entry is: >0 reader count, 0 unlocked, -1 exclusive (or locking).
type vfsShmIOLock struct {
	waiter sync.Cond // +checklocks:mtx
	mtx    sync.Mutex
	cnt    [_SHM_NIOLK]int8 // +checklocks:mtx
}

// +checklocks:l.mtx
func (l *vfsShmIOLock) wait() {
	if l.waiter.L == nil {
		l.waiter.L = &l.mtx
	}
	l.waiter.Wait()
}

func (s *vfsShm) IOLock(lock LockLevel, id int64) (err error) {
	if s.vfsShmParent == nil {
		// The WAL-index is not open.
		return nil
	}

	idx := uint64(id) % _SHM_NIOLK
	p := s.vfsShmParent
	l := &p.ioLock

	switch lock {
	case LOCK_NONE:
		l.mtx.Lock()
		if l.cnt[idx] <= 1 {
			err = p.ioLockFile(lock, int64(idx))
			l.cnt[idx] = 0
		} else {
			l.cnt[idx]--
		}
		l.waiter.Broadcast()
		l.mtx.Unlock()
		return err

	case LOCK_SHARED:
		l.mtx.Lock()
		for l.cnt[idx]+1 <= 0 {
			l.wait()
		}
		if l.cnt[idx] > 0 {
			l.cnt[idx]++
			l.mtx.Unlock()
			return nil
		}
		l.cnt[idx] = -1
		l.mtx.Unlock()

		err = p.ioLockFile(lock, int64(idx))

		l.mtx.Lock()
		if err == nil {
			l.cnt[idx] = 1
		} else {
			l.cnt[idx] = 0
		}
		l.waiter.Broadcast()
		l.mtx.Unlock()

		return err

	case LOCK_EXCLUSIVE:
		l.mtx.Lock()
		for l.cnt[idx] != 0 {
			l.wait()
		}
		l.cnt[idx] = -1
		l.mtx.Unlock()

		err = p.ioLockFile(lock, int64(idx))
		if err != nil {
			l.mtx.Lock()
			l.cnt[idx] = 0
			l.waiter.Broadcast()
			l.mtx.Unlock()
		}
		return err

	default:
		panic(errutil.AssertErr())
	}
}
