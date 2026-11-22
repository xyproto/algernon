// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build (linux || darwin) && (amd64 || arm64)

package vfs

import (
	"bytes"
	"encoding/binary"
	"math"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"golang.org/x/sys/unix"
)

// When a sync fails, Linux marks the pages it could not write clean: the
// page cache still shows them, a later sync succeeds, and once the kernel
// evicts them they read back as the disk holds them. Under
// synchronous=NORMAL the WAL is synced only by a checkpoint, so after its
// sync fails the next checkpoint would copy frames that may not be on disk
// into the database, and the database is damaged once they are evicted.
//
// So the first failed sync of a WAL stops it in the process: every File of
// the WAL then fails each read, write, truncate and sync, and no new File of
// it opens. The WAL is cut at the first byte written since its last good
// sync, which drops the frames that may not be on disk, from the page cache
// too. Under synchronous=FULL a failed commit never made its frames part of
// the WAL, and the cut drops only them.
//
// The WAL index in the -shm, which every process with the database open
// shares, still counts the frames cut: a process that went on with it would
// write past the cut and read the gap as pages of zeros. So the cut is made
// only while this process holds the DMS lock of the -shm for writing, which
// it gets when no other process has the -shm open (see lockDMS). It keeps
// that lock as long as it has the -shm open, so a process that opens the
// database meanwhile waits, and it empties the -shm as it closes it, as the
// first process to open one does. The next connection of any process then
// has SQLite rebuild the WAL index from the frames on disk.
//
// The cut keeps every frame that a checkpoint may have copied into the
// database, in whole or in part. A checkpoint of another process may have
// copied frames this process never synced, and so may one of this process
// under synchronous=OFF. Without them the frames left in the WAL would read
// in place of newer pages of the database. Once the WAL is stopped the
// process no longer writes the database, so no checkpoint of it copies a
// frame that the cut drops. When what was copied cannot be told, the WAL is
// not cut and stays stopped as long as the process runs.
//
// While another process has the -shm open, the WAL stays as it is and stays
// stopped here: that process goes on as with SQLite's own VFS. Once it is
// gone, the cut is made if the WAL, its index and the database are as the
// failed sync left them. If they are not, another process wrote or made a
// checkpoint meanwhile, over frames that may not be on disk: no cut is sound
// then, and the WAL opens as it is. A cut that fails is tried again.
//
// With locking_mode=EXCLUSIVE from the start, the WAL index is in memory and
// no other process has the database: the cut needs no lock.

// wal is a WAL file of one or more Files of the process.
type wal struct {
	id   fileID
	name string // the name the first File opened it by
	nRef int    // Files of the WAL, under wals

	// io is held shared by each write and truncate, from its check that the
	// WAL is not stopped to its end, and alone by each sync. So a sync knows
	// every byte written before it, and no write lands after a cut.
	io sync.RWMutex
	// dirty is the first byte written since the last good sync, clean when
	// none was.
	dirty atomic.Int64
	// failed says a sync failed. It is set with io held alone, under wals.
	failed atomic.Bool
	// settled says the stopped WAL is fit to open again: it was cut, or it
	// is left to the process that wrote it since. Under wals.
	settled bool
	// settling says a goroutine is at the cut, without wals: the WAL is not
	// dropped meanwhile, and no other goroutine tries. Under wals.
	settling bool
	// stop is what the failed sync left, nil when that could not be
	// learned: the WAL then stays stopped. It is set under wals, and used
	// by the goroutine that set settling.
	stop *stopped
	// held is a descriptor of the stopped WAL, -1 when none. It makes the
	// cut, and keeps the inode, which names the WAL in wals, from being
	// given to a new file while it stays there. Under wals.
	held int
}

const clean = math.MaxInt64

// wals are the WALs open in the process, and those a failed sync stopped
// until they open again. The mutex is held to find, make, count and drop
// them.
var wals struct {
	sync.Mutex
	m map[fileID]*wal
}

// attachWAL gives f, a WAL just opened, the wal of its file. A WAL that a
// failed sync stopped opens only when it can again.
func (f *File) attachWAL() int {
	if err := unix.Fstat(f.fd, &f.st); err != nil {
		return IOErrFstat
	}
	id := fileID{uint64(f.st.Dev), uint64(f.st.Ino)}
	wals.Lock()
	defer wals.Unlock()
	w := wals.m[id]
	if w != nil && w.failed.Load() {
		if !w.lift() {
			return IOErr
		}
		w = wals.m[id]
	}
	if w == nil {
		// A stopped WAL that is no longer the file of this name was deleted:
		// SQLite does that once no connection of any process has the
		// database open, and then no one reads its frames again.
		for _, w := range wals.m {
			if w.name == f.path && w.nRef == 0 && w.failed.Load() && !w.settling {
				w.drop()
			}
		}
		w = &wal{id: id, name: f.path, held: -1}
		w.dirty.Store(clean)
		if wals.m == nil {
			wals.m = make(map[fileID]*wal)
		}
		wals.m[id] = w
	}
	w.nRef++
	f.wal = w
	return OK
}

// WALStopped says a failed sync stopped the WAL of the database name, the
// name SQLite opened it by, and it cannot open yet. It lets the WAL go when
// it can: a connection that is only asked whether it is valid, or pinged,
// would otherwise never open the WAL, which is where it goes.
func WALStopped(name string) bool {
	if !WALsStopped() {
		return false
	}
	wals.Lock()
	defer wals.Unlock()
	for _, w := range wals.m {
		if w.name == name+"-wal" && w.failed.Load() && !w.lift() {
			return true
		}
	}
	return false
}

// lift lets the stopped WAL w open again when it can: it is settled, and no
// File of the process has it open. It is called under wals, and lets go of
// wals while it tries the cut: on a disk that fails the cut may take long.
func (w *wal) lift() bool {
	if !w.settled && !w.settling && w.stop != nil && w.held >= 0 {
		w.settling = true
		wals.Unlock()
		settled := w.stop.settle(w.name, w.held)
		wals.Lock()
		w.settled, w.settling = settled, false
	}
	if w.nRef > 0 || !w.settled || w.settling {
		return false
	}
	w.drop()
	return true
}

// stopped is what a failed sync left of a WAL.
type stopped struct {
	wal, db unix.Stat_t // the WAL and the database
	// shared says the WAL index is in the -shm, and idx is what it held.
	shared bool
	idx    index
	// at is where to cut: the first byte written since the last good sync,
	// or later, after the frames a checkpoint may have copied. clean when
	// there is nothing to cut.
	at int64
	// wiped says settle emptied the -shm before a cut that then failed.
	wiped bool
}

// index is what the cut needs of a WAL index: its header, which every commit
// changes, the frames that checkpoints copied into the database, and the
// frames the last checkpoint set out to copy.
type index struct {
	hdr                  [48]byte
	nBackfill, attempted uint32
}

// salts are the salts of the WAL the index is of, as the header of that WAL
// has them.
func (x *index) salts() [8]byte { return [8]byte(x.hdr[32:40]) }

// readIndex reads the WAL index of the descriptor shm. ok is false when it
// is not built.
func readIndex(shm int) (x index, ok bool) {
	// Two copies of the header, of 48 bytes each, nBackfill at byte 96 and
	// nBackfillAttempted at byte 128. isInit is byte 12 of a header.
	var b [132]byte
	for range 100 {
		if n, err := unix.Pread(shm, b[:], 0); err != nil || n != len(b) || b[12] == 0 {
			return x, false
		}
		if bytes.Equal(b[:48], b[48:96]) {
			x.hdr = [48]byte(b[:48])
			x.nBackfill = binary.NativeEndian.Uint32(b[96:])
			x.attempted = binary.NativeEndian.Uint32(b[128:])
			return x, true
		}
		// A commit of this process is writing the header.
		runtime.Gosched()
	}
	return x, false
}

// kept is the offset in the WAL of the descriptor wal after its first
// frames frames, which checkpoints may have copied into the database as the
// WAL index x counts them. ok is false when it cannot be told.
func kept(x *index, frames uint32, wal int) (off int64, ok bool) {
	var hdr [32]byte
	n, err := unix.Pread(wal, hdr[:], 0)
	if err != nil {
		return 0, false
	}
	if salts := x.salts(); n == len(hdr) && !bytes.Equal(hdr[16:24], salts[:]) {
		// The index is not of this WAL yet: a writer of the process wrote
		// the header, its first commit under it is not made, and no frame
		// was copied. The other way round, a header older than the index,
		// is not found at a failed sync: a writer syncs after it wrote the
		// header, and a checkpoint syncs only when the index counts frames,
		// which are those under the header.
		return 0, true
	}
	if frames == 0 {
		return 0, true
	}
	magic, page := binary.BigEndian.Uint32(hdr[:]), int64(binary.BigEndian.Uint32(hdr[8:]))
	if n != len(hdr) || magic&^1 != 0x377f0682 || page < 512 || page > 65536 || page&(page-1) != 0 {
		return 0, false
	}
	return 32 + int64(frames)*(24+page), true
}

// learn takes down what the failed sync left of the WAL name, of the
// descriptor fd, dirty being the first byte written since its last good
// sync, and cuts the WAL when no other process has the database open.
// settled says the WAL is fit to open again. s is nil when what was left
// could not be learned.
func learn(name string, fd int, dirty int64) (s *stopped, settled bool) {
	s = &stopped{at: dirty}
	db := strings.TrimSuffix(name, "-wal")
	if unix.Fstat(fd, &s.wal) != nil || unix.Stat(db, &s.db) != nil {
		return nil, false
	}
	node := lockNode(fileID{uint64(s.db.Dev), uint64(s.db.Ino)}, name)
	if node == nil {
		// The WAL index is in memory.
		return s, s.cut(name, fd, -1)
	}
	defer node.mu.Unlock()
	// From here no checkpoint of the process copies a page: see copies.
	node.stopped = true
	// The lock comes before the look at the WAL index and the files: until
	// it is had, a checkpoint of another process may be copying frames.
	alone := node.lockDMSAlone()
	if alone && (unix.Fstat(fd, &s.wal) != nil || unix.Stat(db, &s.db) != nil) {
		return nil, false
	}
	s.shared = true
	var ok bool
	if s.idx, ok = readIndex(node.fd); !ok {
		return nil, false
	}
	frames := max(s.idx.nBackfill, s.idx.attempted)
	if node.slots[shmCkptSlot].state.Load()&slotExcl != 0 || node.excl {
		// A connection of the process may be at a checkpoint, and then what
		// the index says a checkpoint set out to copy is its own mark, which
		// it wrote before its sync of the WAL. If that is the sync that
		// failed, it copied nothing, and the mark would keep every frame
		// that may not be on disk. So what counts is what was noted: the
		// mark that was there before, and its own once it copied a page.
		if !node.ckpt.ok {
			return nil, false
		}
		frames = s.idx.nBackfill
		if node.ckpt.salts == s.idx.salts() {
			frames = max(frames, node.ckpt.attempted)
		}
	}
	keep, ok := kept(&s.idx, frames, fd)
	if !ok {
		return nil, false
	}
	s.at = max(dirty, keep)
	if !alone {
		return s, false
	}
	return s, s.cut(name, fd, -1)
}

// settle cuts the WAL name, of the descriptor fd, some time after its sync
// failed, when no other process has the database open and nothing was
// written since. It says the WAL is fit to open again.
func (s *stopped) settle(name string, fd int) bool {
	db := strings.TrimSuffix(name, "-wal")
	var st unix.Stat_t
	if unix.Stat(db, &st) != nil {
		return false
	}
	shm, own, unlock := lockDMSAlone(name, fileID{uint64(st.Dev), uint64(st.Ino)})
	if unlock == nil {
		return false
	}
	defer unlock()
	// With the lock, no other process writes.
	if unix.Stat(db, &st) != nil || !same(&st, &s.db) {
		return true
	}
	if unix.Fstat(fd, &st) != nil {
		return false
	}
	if !same(&st, &s.wal) {
		return true
	}
	// An -shm this process emptied, and that no one built since, holds
	// nothing to look at.
	if s.shared && !(s.wiped && unix.Fstat(shm, &st) == nil && st.Size <= 3) {
		if x, ok := readIndex(shm); !ok || x != s.idx {
			return true
		}
	}
	if !own {
		shm = -1
	}
	return s.cut(name, fd, shm)
}

// cut cuts the WAL name, of the descriptor fd, at s.at, and says it is
// done. shm is a descriptor of the -shm when no one maps it, and -1 when
// there is none or a node of the process maps it: the node empties it when
// it closes.
func (s *stopped) cut(name string, fd, shm int) bool {
	var st unix.Stat_t
	if unix.Fstat(fd, &st) != nil {
		return false
	}
	if s.at >= st.Size {
		return true
	}
	if shm >= 0 {
		// The -shm goes first, as when the first process opens it, so that
		// no process finds a WAL index that counts the frames cut.
		if retry(func() error { return unix.Ftruncate(shm, 3) }) != nil {
			return false
		}
		s.wiped = true
	}
	// As the sync that failed, the cut may not reach the disk.
	if faultOn(name, OpTruncate) != OK {
		return false
	}
	return retry(func() error { return unix.Ftruncate(fd, s.at) }) == nil && fullFsync(fd, true) == nil
}

// lockNode returns the node of the database id, whose WAL is wal, with its
// mutex held, and nil when the process has none. A database whose file was
// replaced while it is open is found by the name of its -shm.
func lockNode(id fileID, wal string) *shmNode {
	shmNodes.Lock()
	defer shmNodes.Unlock()
	node := shmNodes.m[id]
	if node == nil {
		shm := strings.TrimSuffix(wal, "-wal") + "-shm"
		for _, n := range shmNodes.m {
			if n.path == shm {
				node = n
				break
			}
		}
	}
	if node != nil {
		// A node in shmNodes has its descriptor open: see ShmUnmap.
		node.mu.Lock()
	}
	return node
}

// lockDMSAlone takes the DMS lock of the node for writing, which it gets
// only when no other process has the -shm open, and keeps it until the node
// closes: the WAL index it maps may count frames that are cut, and no other
// process may come to use it. node.mu is held.
func (node *shmNode) lockDMSAlone() bool {
	if node.readOnly || setLock(node.fd, &node.fl, unix.F_WRLCK, shmDMS, 1) != nil {
		return false
	}
	node.alone = true
	return true
}

// lockDMSAlone takes the DMS lock of the -shm of the WAL wal, of the
// database id, for writing, which it gets only when no other process has
// the -shm open.
// unlock, nil when the lock was not had, is to be called after the work.
// shm is a descriptor of the -shm, -1 when there is no such file. own says
// the process has no node of it and the descriptor is the caller's own: the
// lock goes with it, and a node made meanwhile finds the lock taken, as it
// would find another process's.
func lockDMSAlone(wal string, id fileID) (shm int, own bool, unlock func()) {
	if node := lockNode(id, wal); node != nil {
		if !node.lockDMSAlone() {
			node.mu.Unlock()
			return -1, false, nil
		}
		return node.fd, false, node.mu.Unlock
	}
	fd, err := openFD(strings.TrimSuffix(wal, "-wal")+"-shm", unix.O_RDWR|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err == unix.ENOENT {
		return -1, false, func() {}
	}
	if err != nil {
		return -1, false, nil
	}
	var fl unix.Flock_t
	if setLock(fd, &fl, unix.F_WRLCK, shmDMS, 1) != nil {
		unix.Close(fd)
		return -1, false, nil
	}
	return fd, true, func() { unix.Close(fd) }
}

// drop lets the stopped WAL w go. Under wals.
func (w *wal) drop() {
	delete(wals.m, w.id)
	stoppedWALs.Add(-1)
	if w.held >= 0 {
		unix.Close(w.held)
		w.held = -1
	}
}

// detachWAL drops f from its wal. A WAL a failed sync stopped stays until
// it can open again: see lift.
func (f *File) detachWAL() {
	w := f.wal
	f.wal = nil
	wals.Lock()
	defer wals.Unlock()
	w.nRef--
	switch {
	case w.nRef > 0:
	case w.failed.Load():
		w.lift()
	default:
		delete(wals.m, w.id)
	}
}

// wrote notes that bytes from off on are about to be written. w.io is held
// shared.
func (w *wal) wrote(off int64) {
	for {
		d := w.dirty.Load()
		if d <= off || w.dirty.CompareAndSwap(d, off) {
			return
		}
	}
}

// fail stops the WAL after a sync of f failed, dirty being the first byte
// written since the last good sync, and cuts it there when it can. w.io is
// held alone.
func (w *wal) fail(f *File, dirty int64) {
	wals.Lock()
	w.failed.Store(true)
	stoppedWALs.Add(1)
	fd := f.fd
	if held, err := unix.FcntlInt(uintptr(f.fd), unix.F_DUPFD_CLOEXEC, 3); err == nil {
		w.held, fd = held, held
	}
	w.settling = true
	wals.Unlock()
	// Without wals: on a disk that fails the cut may take long.
	s, settled := learn(w.name, fd, dirty)
	wals.Lock()
	w.stop, w.settled, w.settling = s, settled, false
	wals.Unlock()
}
