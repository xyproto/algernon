//go:build sqlite3_dotlk

package vfs

import (
	"sync/atomic"
	"unsafe"
)

const (
	_WALINDEX_HDR_SIZE = 136
	_WALINDEX_PGSZ     = 32768
	_WAL_WRITE_LOCK    = 0
)

// This seems a safe way of keeping the WAL-index in sync.
//
// The WAL-index file starts with a header,
// and the index doesn't meaningfully change if the header doesn't change.
//
// The header starts with two 48 byte, checksummed, copies of the same information,
// which are accessed independently between memory barriers.
// The checkpoint information that follows uses 4 byte aligned words.
//
// Sections of the header are modified under different locks,
// so they can change concurrently.
// We keep a shadow copy of the header to know which words changed,
// and merge those word by word.
//
// Then we have the WAL-index hash tables,
// which are only modified holding the exclusive WAL_WRITE_LOCK,
// and only matter once the header changes to reflect them.
//
// The writer copies them out whenever it publishes a new header,
// and everyone else copies them in whenever they see one.
// A writer that rolls back leaves garbage in its private copy,
// but nothing that SQLite would look at.
//
// https://sqlite.org/walformat.html#the_wal_index_file_format

// +checklocks:s.Mutex
func (s *vfsShm) shmAcquire(errp *error, writer bool) {
	if errp != nil && *errp != nil {
		return
	}
	// No changes to the header means nothing needs to be loaded.
	if len(s.ptrs) == 0 || shmEqual(s.shadow[:], s.shared[0][:]) {
		return
	}
	// Merge modified header words from shared to private memory.
	shared := shmHdr(s.shared[0][:])
	shadow := shmHdr(s.shadow[:])
	privat := shmHdr(s.wrp.Bytes(s.ptrs[0], _WALINDEX_HDR_SIZE))
	for i, shared := range shared {
		if shadow[i] != shared {
			shadow[i] = shared
			privat[i] = shared
		}
	}
	// If we're the writer, our hash tables are authoritative.
	if writer {
		return
	}
	// Otherwise, copy hash tables from shared to private memory.
	for pgno, ptr := range s.ptrs {
		s.shmCopyTabs(pgno, s.wrp.Bytes(ptr, _WALINDEX_PGSZ), s.shared[pgno][:])
	}
}

// +checklocks:s.Mutex
func (s *vfsShm) shmRelease(writer bool) {
	// No changes to the header mean nothing is ready to be shared.
	if len(s.ptrs) == 0 || shmEqual(s.shadow[:], s.wrp.Bytes(s.ptrs[0], _WALINDEX_HDR_SIZE)) {
		return
	}
	// Merge modified header words from private to shared memory.
	shared := shmHdr(s.shared[0][:])
	shadow := shmHdr(s.shadow[:])
	privat := shmHdr(s.wrp.Bytes(s.ptrs[0], _WALINDEX_HDR_SIZE))
	for i, privat := range privat {
		if shadow[i] != privat {
			shadow[i] = privat
			shared[i] = privat
		}
	}

	if !writer {
		return
	}
	// We're the writer, our hash tables are authoritative,
	// copy them from private to shared memory.
	for pgno, ptr := range s.ptrs {
		s.shmCopyTabs(pgno, s.shared[pgno][:], s.wrp.Bytes(ptr, _WALINDEX_PGSZ))
	}
}

func (s *vfsShm) shmBarrier() {
	var b atomic.Bool
	s.Lock()
	writer := s.lock[_WAL_WRITE_LOCK]
	s.shmAcquire(nil, writer)
	b.Swap(true)
	s.shmRelease(writer)
	s.Unlock()
}

//go:nosplit
func shmHdr(s []byte) *[_WALINDEX_HDR_SIZE / 4]uint32 {
	ptr := (*uint32)(unsafe.Pointer(unsafe.SliceData(s)))
	return (*[_WALINDEX_HDR_SIZE / 4]uint32)(unsafe.Slice(ptr, _WALINDEX_HDR_SIZE/4))
}

//go:nosplit
func shmEqual(v1, v2 []byte) bool {
	return *(*[_WALINDEX_HDR_SIZE]byte)(v1[:]) == *(*[_WALINDEX_HDR_SIZE]byte)(v2[:])
}

func (*vfsShm) shmCopyTabs(page int, dst, src []byte) {
	if page == 0 {
		dst = dst[_WALINDEX_HDR_SIZE:]
		src = src[_WALINDEX_HDR_SIZE:]
	}
	copy(dst, src)
}
