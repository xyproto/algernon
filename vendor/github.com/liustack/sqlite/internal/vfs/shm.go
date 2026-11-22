// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build (linux || darwin) && (amd64 || arm64)

package vfs

import (
	"encoding/binary"
	"math/bits"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// shmNode is the wal-index of one database in this process, as os_unix's
// unixShmNode: the "-shm" file next to the database, opened once and mapped
// MAP_SHARED in regions, shared by all the Files of that database.
//
// The node's descriptor holds the process's locks on the -shm, and the node
// counts which connection of the process holds which slot. A connection
// conflicting with another of the process is refused here, and the kernel
// is asked only when the process's lock on a slot changes. With one lock
// owner per process, the kernel's lock list on the file stays short, and
// most read marks are taken with no system call at all. Each slot has its
// own state and its own mutex (shmSlot): the mutex of the node is not taken
// to lock or unlock a slot, but for the checkpoint slot.
type shmNode struct {
	id   fileID // of the database file
	path string
	fd   int
	nRef int // Files using the node, under shmNodes

	mu       sync.Mutex
	readOnly bool              // the -shm could only be opened for reading
	unlocked bool              // read-only and no one had it open: the DMS lock is not held
	slots    [ShmNLock]shmSlot // who holds each slot, in the process and in the kernel
	others   atomic.Bool       // another process had the -shm open when last looked: no idle slots
	regions  []uintptr         // address of each region mapped
	// mapped is regions as ShmMap may read it without mu: a copy of the
	// slice, stored under mu each time regions grows. Regions are only
	// added, and unmapped when the last File goes.
	mapped atomic.Pointer[[]uintptr]
	maps   []mapping    // what to unmap
	fl     unix.Flock_t // space for the fcntl argument, for the DMS byte
	timer  *time.Timer  // releases the idle slots
	armed  atomic.Bool  // timer is set to run
	closed bool         // the descriptor is closed: the timer does nothing
	// alone says the process took the DMS lock for writing after a sync of
	// the WAL failed, and keeps it: see wal_unix.go. The -shm is emptied as
	// the node closes.
	alone bool
	// stopped says a sync of the WAL failed: the process no longer writes
	// the database. See copies.
	stopped bool
	// excl says a File of the process holds EXCLUSIVE on the database. It is
	// then the only connection of any process, and SQLite takes no slot for
	// its checkpoints: the one it makes as it closes, and those under a
	// locking_mode=EXCLUSIVE set once the WAL was open. It is set by Lock,
	// and by ShmMap for a File that took the lock first.
	excl bool
	// ckpt is the WAL the index was of, and the most frames it said a
	// checkpoint had set out to copy, at the times that was not the mark of
	// a checkpoint that has copied nothing yet: see noteCkpt. ok is false
	// when it could not be read.
	ckpt struct {
		ok        bool
		salts     [8]byte
		attempted uint32
	}
}

// shmSlot is one slot of the wal-index in this process.
//
// state says who holds it: how many connections of the process hold it
// shared (slotCount), whether one holds it exclusive (slotExcl), and which
// lock the process has on it in the kernel (slotRead, slotWrite). A slot
// held shared is read locked in the kernel, and one held exclusive is write
// locked. A slot locked in the kernel with no holder is idle: a read slot
// kept for the next reader (see shmReadSlots), or a slot whose unlock
// failed, which the timer unlocks again.
//
// The lock in the kernel changes with mu held, and so does state when the
// process has no read lock on the slot. While the process has the read lock
// and no connection holds the slot exclusive, connections take it shared
// and give it back by changing the count alone, with no mutex. Before the
// read lock goes or becomes a write lock, it is taken out of state, so that
// from then on a reader waits for mu. A goroutine that waits for a mutex
// sleeps, and with every core busy it waits long for a core once it is
// woken: so a reader that takes a read slot the process has locked takes no
// mutex, and the mutex of the write slot is one no reader takes.
//
// A connection that has mu takes the slot or gives it back, and its call has
// not returned: an exclusive lock that finds mu held is refused the slot, as
// it is while the connection holds it, and does not sleep (see lockFree for
// the one case that is not so). Two take mu and may leave the slot with no
// holder: the timer, and an exclusive lock of several slots, which may be
// refused another of them. They are counted in pass from before they take mu
// until after they give it back, and an exclusive lock waits for them.
type shmSlot struct {
	state atomic.Int32
	mu    sync.Mutex
	pass  atomic.Uint64 // passCount of them now, and above it how many began
	fl    unix.Flock_t  // space for the fcntl argument, under mu
	// The slots are not on the same cache line: a writer changes the write
	// slot while readers change the slot of a read mark.
	_ [88]byte
}

const (
	passCount = 1<<32 - 1
	passBegin = 1<<32 + 1
	passEnd   = ^uint64(0) // one less in passCount
)

const (
	slotCount = 1<<24 - 1 // connections of the process that hold the slot shared
	slotRead  = 1 << 24   // the process has it read locked in the kernel
	slotWrite = 1 << 25   // the process has it write locked in the kernel
	slotExcl  = 1 << 26   // a connection holds it exclusive, with slotWrite
)

// shmExclTries is how many times a reader looks again at a read slot that
// another connection of the process holds exclusive, before it is refused:
// some microseconds. Tests change it.
var shmExclTries = 1 << 14

// lock takes the mutex of the slot. It tries for a moment before it waits:
// the mutex is held for one system call.
func (sl *shmSlot) lock() {
	for range 256 {
		if sl.mu.TryLock() {
			return
		}
	}
	sl.mu.Lock()
}

// shmFreeTries is how many times an exclusive lock looks again at a slot
// whose mutex a connection has, before the slot is refused: longer than the
// system call the connection makes. Tests change it.
var shmFreeTries = 1 << 12

// lockFree takes the mutex of the slot for an exclusive lock. It says false,
// with the mutex not taken, when the slot is refused: a connection holds it,
// or has its mutex for longer than a system call takes. It waits for the
// timer and for an exclusive lock of several slots, which may leave the slot
// free. own is 1 when the caller is such a lock, and counted in pass.
//
// pass is read before the mutex is tried and after: when nobody but the
// caller is counted and nobody began in between, the mutex was not with one
// that is counted. Then a connection has it, or is about to: it takes the
// slot, gives it back, or was itself refused the slot exclusive. The slot is
// looked at again until the connection is seen to hold it or the mutex is
// free, so that the caller is not refused a slot the connection leaves.
//
// When the tries run out the connection is most often off its core in the
// middle of a system call, and the slot is refused as if it held it. In one
// case nobody holds it then: the connection was itself refused the slot
// exclusive because of a reader, the reader left, and the connection lost
// its core in the few instructions before it gave the mutex back.
func (sl *shmSlot) lockFree(own uint64) bool {
	for tries := 0; ; {
		p := sl.pass.Load()
		if sl.mu.TryLock() {
			return true
		}
		if p&passCount != own {
			sl.lock()
			return true
		}
		if sl.pass.Load() != p {
			continue
		}
		if sl.state.Load()&(slotCount|slotExcl) != 0 {
			return false
		}
		if tries++; tries > shmFreeTries {
			return false
		}
	}
}

type fileID struct{ dev, ino uint64 }

type mapping struct {
	addr unsafe.Pointer
	size uintptr
}

// shmNodes are the open nodes, by database file. The mutex is held to find,
// make, count and close nodes, as os_unix's unixBigLock.
var shmNodes struct {
	sync.Mutex
	m map[fileID]*shmNode
}

// shm is a File's use of the node, as os_unix's unixShm: the slots this
// connection holds.
type shm struct {
	node   *shmNode
	shared uint16 // slots locked shared
	excl   uint16 // slots locked exclusive
}

// ShmMap returns the address of region n of the wal-index, each region size
// bytes. When the region is not there yet, extend makes it, and without
// extend the address is 0 and the result OK. The memory stays mapped until
// ShmUnmap. With a -shm that can only be read, the result is SQLITE_READONLY
// with the address, or SQLITE_READONLY_CANTINIT when no other connection has
// it open.
func (f *File) ShmMap(n, size int, extend bool) (uintptr, int) {
	if f.shm == nil {
		if rc := f.openShm(); rc != OK {
			return 0, rc
		}
	}
	node := f.shm.node
	// With pages larger than a region, map whole pages.
	perMap := os.Getpagesize() / size
	if perMap < 1 {
		perMap = 1
	}
	want := (n + perMap) / perMap * perMap

	// A region mapped already is found without the mutex (docs/busy-cores.md
	// section 7): connections ask for one each time they look up a frame in a
	// part of the index they have not used yet. A read-only -shm (its DMS
	// lock, its result) and a File that holds EXCLUSIVE (node.excl) go the
	// long way.
	if !node.readOnly && f.lock < LockExclusive {
		if r := node.mapped.Load(); r != nil && len(*r) >= want {
			return (*r)[n], OK
		}
	}

	node.mu.Lock()
	defer node.mu.Unlock()
	if node.unlocked {
		if rc := node.lockDMS(); rc != OK {
			return 0, rc
		}
		node.unlocked = false
	}

	rc := OK
	if len(node.regions) < want {
		rc = node.mapRegions(want, size, perMap, extend)
	}
	if f.lock >= LockExclusive && !node.excl && len(node.maps) > 0 {
		// The File took EXCLUSIVE before it opened the WAL index, as a
		// connection that leaves WAL mode without having read the WAL.
		node.excl = true
		node.noteCkpt()
	}
	var addr uintptr
	if n < len(node.regions) {
		addr = node.regions[n]
	}
	if node.readOnly && rc == OK {
		rc = ReadOnly
	}
	return addr, rc
}

// mapRegions maps regions until there are want of them, making the file
// larger first when extend is set. node.mu is held.
func (node *shmNode) mapRegions(want, size, perMap int, extend bool) int {
	defer node.publish()
	var st unix.Stat_t
	if err := unix.Fstat(node.fd, &st); err != nil {
		return IOErrShmSize
	}
	need := int64(want) * int64(size)
	if st.Size < need {
		if !extend {
			return OK
		}
		// Write a byte at the end of every new page, as os_unix, so that
		// the space is there now and not found missing (SIGBUS) when the
		// memory is used.
		const pgsz = 4096
		zero := []byte{0}
		for pg := st.Size / pgsz; pg < need/pgsz; pg++ {
			var n int
			err := retry(func() (err error) {
				n, err = unix.Pwrite(node.fd, zero, pg*pgsz+pgsz-1)
				return err
			})
			if err != nil || n != 1 {
				return IOErrShmSize
			}
		}
	}

	prot := unix.PROT_READ | unix.PROT_WRITE
	if node.readOnly {
		prot = unix.PROT_READ
	}
	for len(node.regions) < want {
		m := mapping{size: uintptr(size * perMap)}
		p, err := unix.MmapPtr(node.fd, int64(size)*int64(len(node.regions)), nil, m.size, prot, unix.MAP_SHARED)
		if err != nil {
			return IOErrShmMap
		}
		m.addr = p
		node.maps = append(node.maps, m)
		for i := range perMap {
			node.regions = append(node.regions, uintptr(p)+uintptr(i*size))
		}
	}
	return OK
}

// publish stores a copy of regions in mapped. node.mu is held.
func (node *shmNode) publish() {
	r := node.regions[:len(node.regions):len(node.regions)]
	node.mapped.Store(&r)
}

// openShm attaches the File to the node of its database, as os_unix's
// unixOpenSharedMemory. The first File of the process opens the -shm and
// takes the DMS lock. The others find the node.
func (f *File) openShm() int {
	id := fileID{f.dev, f.ino}
	shmNodes.Lock()
	defer shmNodes.Unlock()

	rc := OK
	node := shmNodes.m[id]
	if node == nil {
		if node, rc = newShmNode(f, id); rc != OK && rc != ReadOnlyCantInit {
			return rc
		}
		if shmNodes.m == nil {
			shmNodes.m = make(map[fileID]*shmNode)
		}
		shmNodes.m[id] = node
	}
	node.nRef++
	f.shm = &shm{node: node}
	return rc
}

// newShmNode opens the -shm of f and takes the DMS lock.
func newShmNode(f *File, id fileID) (*shmNode, int) {
	if err := unix.Fstat(f.fd, &f.st); err != nil {
		return nil, IOErrFstat
	}
	mode := uint32(f.st.Mode) & 0777
	node := &shmNode{id: id, path: f.path + "-shm", fd: -1}

	// As os_unix, readonly_shm=1 opens it for reading only at once, and
	// one that cannot be written is opened for reading only.
	fd, err := -1, error(unix.EACCES)
	if !f.roShm {
		fd, err = openFile(node.path, unix.O_RDWR|unix.O_CREAT|unix.O_NOFOLLOW|unix.O_CLOEXEC, mode)
	}
	if err != nil {
		fd, err = openFile(node.path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, mode)
		if err != nil {
			return nil, CantOpen
		}
		node.readOnly = true
	}
	node.fd = fd
	// A -shm made by root is given to the owner of the database.
	if os.Geteuid() == 0 {
		unix.Fchown(fd, int(f.st.Uid), int(f.st.Gid))
	}

	rc := node.lockDMS()
	if rc != OK && rc != ReadOnlyCantInit {
		unix.Close(fd)
		return nil, rc
	}
	node.unlocked = rc == ReadOnlyCantInit
	node.timer = time.AfterFunc(time.Hour, node.releaseIdle)
	node.timer.Stop()
	return node, rc
}

// lockDMS takes a read lock on the DMS byte, as os_unix's
// unixLockSharedMemory. The DMS byte is read locked by every process that
// has the -shm open. When no one holds it, this process is the first, and
// the -shm may hold anything a crash left: it takes a write lock, truncates
// the file, and turns the lock into a read lock. A write lock held by
// another means that one is truncating: SQLITE_BUSY, and SQLite tries again.
func (node *shmNode) lockDMS() int {
	typ, err := testLock(node.fd, &node.fl, shmDMS, 1)
	if err != nil {
		return IOErrLock
	}
	node.others.Store(typ != unix.F_UNLCK)
	switch typ {
	case unix.F_WRLCK:
		return Busy
	case unix.F_UNLCK:
		if node.readOnly {
			return ReadOnlyCantInit
		}
		if err := setLock(node.fd, &node.fl, unix.F_WRLCK, shmDMS, 1); err != nil {
			return Busy
		}
		// To 3 bytes and not 0, as os_unix, so that a legitimate
		// truncation can be told from the work of a stray process.
		if err := retry(func() error { return unix.Ftruncate(node.fd, 3) }); err != nil {
			return IOErrShmOpen
		}
	}
	if err := setLock(node.fd, &node.fl, unix.F_RDLCK, shmDMS, 1); err != nil {
		return Busy
	}
	return OK
}

// ShmLock locks or unlocks n slots of the wal-index from offset, as
// os_unix's unixShmLock. flags is SQLITE_SHM_LOCK or SQLITE_SHM_UNLOCK with
// SQLITE_SHM_SHARED or SQLITE_SHM_EXCLUSIVE. A shared lock is on one slot.
// SQLite never asks for a lock it holds, nor moves a slot between shared
// and exclusive without unlocking it first. s.shared and s.excl are only
// read and written by the connection.
func (f *File) ShmLock(offset, n, flags int) int {
	s := f.shm
	if s == nil {
		return IOErrShmLock
	}
	mask := uint16(1<<(offset+n) - 1<<offset)
	if mask&(1<<shmCkptSlot) != 0 {
		// learn in wal_unix.go looks, with the mutex of the node, at who
		// has the checkpoint slot and at what noteCkpt took down.
		s.node.mu.Lock()
		defer s.node.mu.Unlock()
	}
	switch {
	case flags&ShmUnlock != 0:
		if (s.shared|s.excl)&mask == 0 {
			return OK
		}
		return s.unlock(offset, n, mask)
	case flags&ShmShared != 0:
		return s.lockShared(offset, mask)
	}
	return s.lockExcl(offset, n, mask)
}

// lockShared takes slot i shared.
func (s *shm) lockShared(i int, bit uint16) int {
	if s.shared&bit != 0 {
		return OK
	}
	node := s.node
	sl := &node.slots[i]
	for tries := 0; ; {
		v := sl.state.Load()
		if v&slotExcl != 0 {
			// Another connection of the process holds it exclusive. On a
			// read slot it is moving the read mark, which takes two system
			// calls: the slot is looked at again for about that long. After
			// a few refusals SQLite sleeps, and a goroutine that slept
			// waits long for a core when every core is busy.
			if tries++; bit&shmReadSlots == 0 || tries > shmExclTries {
				return Busy
			}
			continue
		}
		if v&slotRead != 0 {
			// The process has the read lock: one more holder.
			if sl.state.CompareAndSwap(v, v+1) {
				s.shared |= bit
				return OK
			}
			continue
		}
		// The slot is free, or idle and write locked since its exclusive
		// unlock failed: then the write lock becomes a read lock, so that
		// other processes can read, or the slot stays idle for the timer.
		// Its state changes only with the mutex held, but for the count of
		// a slot the process has read locked.
		sl.lock()
		v = sl.state.Load()
		for v&slotRead != 0 {
			// Another reader locked it meanwhile: one more holder, taken
			// here so that whoever finds the mutex held finds the slot
			// held after it.
			if sl.state.CompareAndSwap(v, v+1) {
				sl.mu.Unlock()
				s.shared |= bit
				return OK
			}
			v = sl.state.Load()
		}
		if v&slotExcl != 0 {
			sl.mu.Unlock()
			continue
		}
		err := setLock(node.fd, &sl.fl, unix.F_RDLCK, int64(shmBase+i), 1)
		if err == nil {
			sl.state.Store(slotRead | 1)
		}
		sl.mu.Unlock()
		if err != nil {
			return lockError(err, IOErrShmLock)
		}
		s.shared |= bit
		return OK
	}
}

// lockExcl takes the n slots of mask, from offset, exclusive. A slot is
// refused, as a slot another process holds, when a connection of the process
// holds it, or takes it or gives it back for longer than lockFree looks:
// with no mutex waited for, since a goroutine that waits for a mutex sleeps.
// lockExcl does wait for the timer and for an exclusive lock of several
// slots (see shmSlot), which hold no slot: SQLITE_BUSY because of them would
// fail a checkpoint with no reader and no writer, and a checkpoint with no
// busy handler does not ask again.
func (s *shm) lockExcl(offset, n int, mask uint16) int {
	node := s.node
	slots := node.slots[offset : offset+n]
	for i := range slots {
		if slots[i].state.Load()&(slotCount|slotExcl) != 0 {
			return Busy
		}
	}
	// Asked for several slots, it may be refused one and leave the others
	// free: a connection that asks for one of those waits for it. Each slot
	// is counted from before its mutex is asked for. The mutexes are taken
	// in the order of the slots, here and in unlockExcl.
	var own uint64
	if n > 1 {
		own = 1
	}
	for i := range slots {
		if own != 0 {
			slots[i].pass.Add(passBegin)
		}
		if slots[i].lockFree(own) {
			continue
		}
		for j := 0; j <= i; j++ {
			if j < i {
				slots[j].mu.Unlock()
			}
			if own != 0 {
				slots[j].pass.Add(passEnd)
			}
		}
		return Busy
	}
	// The slots are taken from the readers of the process before the kernel
	// is asked: an idle read lock is out of state, and a reader waits for
	// the mutex to learn what came of it.
	var was [ShmNLock]int32
	rc, taken := OK, 0
	for ; taken < n; taken++ {
		v := slots[taken].state.Load()
		if v&(slotCount|slotExcl) != 0 || !slots[taken].state.CompareAndSwap(v, 0) {
			rc = Busy
			break
		}
		was[taken] = v
	}
	if rc == OK {
		if err := setLock(node.fd, &slots[0].fl, unix.F_WRLCK, int64(shmBase+offset), int64(n)); err != nil {
			rc = lockError(err, IOErrShmLock)
		}
	}
	idle := false
	for i := range slots {
		switch {
		case rc == OK:
			// An idle lock of the process became this write lock.
			slots[i].state.Store(slotWrite | slotExcl)
		case i < taken:
			slots[i].state.Store(was[i])
			idle = idle || was[i] != 0
		}
		slots[i].mu.Unlock()
		if own != 0 {
			slots[i].pass.Add(passEnd)
		}
	}
	if idle {
		// A slot that was idle is idle again. The timer may have run while
		// it was out of state, and then it did not see the slot.
		node.armIdle()
	}
	if rc == OK {
		s.excl |= mask
		if mask&(1<<shmCkptSlot) != 0 {
			node.noteCkpt()
		}
	}
	return rc
}

// shmCkptSlot is WAL_CKPT_LOCK, the slot a checkpoint holds for writing.
const shmCkptSlot = 1

// noteCkpt takes down what the WAL index says the last checkpoint set out to
// copy. A checkpoint writes that mark before it syncs the WAL, so when the
// sync fails the mark may be of a checkpoint of the process that has copied
// nothing, and learn in wal_unix.go takes what was noted in its place. It
// is noted when a connection of the process takes the checkpoint lock or
// EXCLUSIVE on the database, before a checkpoint of it writes its own mark,
// and when a checkpoint of the process copies a page, as its mark counts
// from then on. node.mu is held.
func (node *shmNode) noteCkpt() {
	was := node.ckpt
	node.ckpt.ok = false
	if len(node.maps) == 0 || node.maps[0].size < 132 {
		return
	}
	// The salts are bytes 32 to 39 of each copy of the header, and
	// nBackfillAttempted is at byte 128. Only a checkpoint writes it, and a
	// writer that starts the WAL over, after the salts.
	idx := (*[132]byte)(node.maps[0].addr)
	for range 100 {
		salts := [8]byte(idx[32:40])
		attempted := binary.NativeEndian.Uint32(idx[128:])
		if salts == [8]byte(idx[80:88]) && salts == [8]byte(idx[32:40]) {
			if was.ok && was.salts == salts {
				// Each mark noted is of a checkpoint that may have copied.
				attempted = max(attempted, was.attempted)
			}
			node.ckpt.salts, node.ckpt.attempted, node.ckpt.ok = salts, attempted, true
			return
		}
		// A writer is starting the WAL over, and has written one copy.
		runtime.Gosched()
	}
}

// exclusive notes that a File of the process holds EXCLUSIVE on the
// database, or no longer does.
func (node *shmNode) exclusive(held bool) {
	node.mu.Lock()
	node.excl = held
	if held {
		node.noteCkpt()
	}
	node.mu.Unlock()
}

// copies is called before a File that has the WAL index open writes the
// database or sets its size, which in WAL mode a checkpoint does. It says
// whether it may. Once a sync of the WAL failed it may not: a checkpoint
// that had read a frame would copy it after the frame was cut. Before that,
// the mark of the checkpoint is noted, under the mutex that learn holds, so
// that the cut keeps every frame the checkpoint copies.
func (node *shmNode) copies() bool {
	node.mu.Lock()
	defer node.mu.Unlock()
	if node.stopped {
		return false
	}
	node.noteCkpt()
	return true
}

// unlock releases the slots of mask, n from offset, that s holds. s lets go
// of every one of them, as SQLite takes it to, also when the kernel fails
// to unlock: such a slot is still locked in the kernel with no holder, the
// node keeps it as an idle slot, which the next holder of the process takes
// over and the timer tries to unlock again.
func (s *shm) unlock(offset, n int, mask uint16) int {
	node := s.node
	var first error
	for i := offset; i < offset+n; i++ {
		if bit := uint16(1) << i; s.shared&bit != 0 && s.excl&bit == 0 {
			if err := node.unshare(i); err != nil && first == nil {
				first = err
			}
		}
	}
	if held := s.excl & mask; held != 0 {
		if err := node.unlockExcl(held); err != nil && first == nil {
			first = err
		}
	}
	s.shared &^= mask
	s.excl &^= mask
	if first != nil {
		return lockError(first, IOErrShmLock)
	}
	return OK
}

// unshare gives back a shared hold of slot i. The slot stays locked in the
// kernel while other connections of the process hold it, and a read slot
// stays locked a little longer while no other process has the -shm open
// (see shmReadSlots).
func (node *shmNode) unshare(i int) error {
	sl := &node.slots[i]
	kept := shmReadSlots&(1<<i) != 0
	for {
		v := sl.state.Load()
		last := v&slotCount == 1
		if last && (!kept || node.others.Load()) {
			break
		}
		if sl.state.CompareAndSwap(v, v-1) {
			if last {
				// Alone, the process keeps the slot, and the timer looks
				// for other processes. Another process may have come since
				// others was read: the timer finds the slot idle, at this
				// tick if it had not passed the slot, or at the tick set
				// here.
				node.armIdle()
			}
			return nil
		}
	}
	sl.mu.Lock()
	for {
		v := sl.state.Load()
		if v&slotCount > 1 {
			if sl.state.CompareAndSwap(v, v-1) {
				sl.mu.Unlock()
				return nil
			}
		} else if sl.state.CompareAndSwap(v, 0) {
			// The last holder: the read lock is out of state before the
			// kernel unlocks, and a reader that comes now waits for mu.
			break
		}
	}
	err := setLock(node.fd, &sl.fl, unix.F_UNLCK, int64(shmBase+i), 1)
	if err != nil {
		sl.state.Store(slotRead)
	}
	sl.mu.Unlock()
	if kept || err != nil {
		// The timer looks again for other processes, and unlocks again.
		node.armIdle()
	}
	return err
}

// unlockExcl unlocks the slots of mask, which a connection holds exclusive,
// in the kernel: one call for each run of slots next to each other, and
// everything SQLite asks is one run. A call that fails leaves its slots
// write locked and idle, and the result is the first failure.
//
// One read slot, with no other process there, goes back to a read lock and
// stays idle: a connection that moved a read mark takes the slot shared
// next, and so do the readers that come to the mark.
func (node *shmNode) unlockExcl(mask uint16) (first error) {
	arm := false
	for rest := mask; rest != 0; {
		lo := bits.TrailingZeros16(rest)
		hi := lo + bits.TrailingZeros16(^(rest >> lo))
		rest &^= uint16(1<<hi - 1<<lo)
		for i := lo; i < hi; i++ {
			node.slots[i].mu.Lock()
		}
		typ, state := int16(unix.F_UNLCK), int32(0)
		if hi-lo == 1 && shmReadSlots&(1<<lo) != 0 && !node.others.Load() {
			typ, state = unix.F_RDLCK, slotRead
		}
		if err := setLock(node.fd, &node.slots[lo].fl, typ, int64(shmBase+lo), int64(hi-lo)); err != nil {
			state = slotWrite
			if first == nil {
				first = err
			}
		}
		for i := lo; i < hi; i++ {
			node.slots[i].state.Store(state)
			node.slots[i].mu.Unlock()
		}
		arm = arm || state != 0
	}
	if arm {
		node.armIdle()
	}
	return first
}

// The read slots, WAL_READ_LOCK(0) to WAL_READ_LOCK(4), are released late
// while no other process has the -shm open.
//
// A read transaction takes one of them shared and gives it back. When the
// process's last reader gives it back, the kernel lock is kept and the slot
// marked idle, so that the next reader of the process takes it again with
// no system call. A timer releases the idle slots shmIdleDelay later, and
// runs at most once per delay, however many transactions there are.
//
// Keeping the lock only stops aReadMark of the slot from changing, the
// guarantee a live reader holds, so the check walTryBeginRead makes after
// taking the lock stays valid, and no one reads wrong data. Connections of
// this process are not slowed at all: an exclusive lock of the process on
// an idle slot turns the kept read lock into a write lock.
//
// Another process is slowed. With readers here less than shmIdleDelay
// apart, a slot would be locked nearly all the time: the checkpoints of
// the other process would rarely be complete and its WAL would grow
// without bound. So the slots are kept only while no other process has the
// -shm open. Every process that has it open, this package or os_unix,
// holds a read lock on the DMS byte, and a test of that byte on the node's
// descriptor sees the locks of every other owner, OFD or POSIX, but not
// its own. The test is made when the -shm is opened and at every tick of
// the timer, which also runs after the last reader of a slot leaves while
// other processes are there. With another process, read slots are released
// at once, as os_unix does.
//
// A process that opens the -shm just after a test sees an idle slot of
// this process locked until the next tick at the latest: shmIdleDelay
// after the slot went idle, and later only by how late the Go runtime runs
// the timer. From that tick on, this process releases read slots at once.
// A reader that left its slot while that tick ran may still have kept it:
// it set the timer again, and the slot goes at the tick after.
//
// WRITE, CKPT and RECOVER (slots 0 to 2) are always released at once.
const shmReadSlots = uint16(1<<ShmNLock - 1<<3)

// shmIdleDelay is how long idle read slots stay locked. Tests change it.
var shmIdleDelay = time.Millisecond

// armIdle sets the timer to release the idle slots, if it is not set.
func (node *shmNode) armIdle() {
	if !node.armed.Load() && node.armed.CompareAndSwap(false, true) {
		node.timer.Reset(shmIdleDelay)
	}
}

// releaseIdle looks whether other processes have the -shm open, and unlocks
// in the kernel the slots no connection of the process holds. The timer
// runs it.
func (node *shmNode) releaseIdle() {
	node.mu.Lock()
	defer node.mu.Unlock()
	// Before the slots are looked at: a slot that goes idle after its turn
	// here sets the timer again.
	node.armed.Store(false)
	if node.closed {
		return
	}
	// When the test fails, act as if another process were there.
	typ, err := testLock(node.fd, &node.fl, shmDMS, 1)
	node.others.Store(err != nil || typ != unix.F_UNLCK)
	// An idle slot has no holder. It is taken out of state first: a reader
	// that takes it before that keeps it, and one that comes after waits
	// for the mutex. Idle slots next to each other are unlocked with one
	// call. A call that fails leaves its slots locked and idle, and the
	// next tick tries again.
	again := false
	var was [ShmNLock]int32
	for i := 0; i < ShmNLock; {
		lo := i
		for ; i < ShmNLock; i++ {
			sl := &node.slots[i]
			v := sl.state.Load()
			if v != slotRead && v != slotWrite {
				break // held, or not locked
			}
			// The timer holds no slot: it is counted in pass, so that an
			// exclusive lock waits for it and is not refused.
			sl.pass.Add(passBegin)
			if !sl.mu.TryLock() {
				sl.pass.Add(passEnd)
				again = true
				break
			}
			if !sl.state.CompareAndSwap(v, 0) {
				sl.mu.Unlock()
				sl.pass.Add(passEnd)
				break
			}
			was[i] = v
		}
		if i == lo {
			i++
			continue
		}
		failed := setLock(node.fd, &node.slots[lo].fl, unix.F_UNLCK, int64(shmBase+lo), int64(i-lo)) != nil
		for j := lo; j < i; j++ {
			if failed {
				node.slots[j].state.Store(was[j])
				again = true
			}
			node.slots[j].mu.Unlock()
			node.slots[j].pass.Add(passEnd)
		}
	}
	if again {
		node.armIdle()
	}
}

// ShmUnmap detaches the File from the wal-index and releases the slots it
// still holds. The last File of the process unmaps the memory and closes
// the -shm, which drops the DMS lock. remove then deletes the file: SQLite
// asks for that only while it holds EXCLUSIVE on the database and no other
// connection has the -shm open. Without a wal-index it does nothing.
func (f *File) ShmUnmap(remove bool) int {
	s := f.shm
	if s == nil {
		return OK
	}
	f.shm = nil
	node := s.node
	node.mu.Lock()
	if held := s.shared | s.excl; held != 0 {
		s.unlock(0, ShmNLock, held)
	}
	if f.lock >= LockExclusive {
		node.excl = false
	}
	node.mu.Unlock()

	shmNodes.Lock()
	defer shmNodes.Unlock()
	node.nRef--
	if node.nRef > 0 {
		return OK
	}
	delete(shmNodes.m, node.id)
	// Closing the descriptor drops the idle read locks with the rest.
	node.mu.Lock()
	node.closed = true
	if node.timer != nil {
		node.timer.Stop()
	}
	node.mu.Unlock()
	for _, m := range node.maps {
		unix.MunmapPtr(m.addr, m.size)
	}
	if node.alone {
		// To 3 bytes, as the first process to open it does (see lockDMS):
		// a process that was about to take the DMS lock for reading, having
		// seen this process hold it, would otherwise use the index as it is.
		if faultOn(node.path, OpTruncate) != OK || retry(func() error { return unix.Ftruncate(node.fd, 3) }) != nil {
			// A header of zeros is not that of an index that was built, so
			// SQLite builds it again all the same.
			unix.Pwrite(node.fd, make([]byte, 136), 0)
		}
	}
	if remove {
		unix.Unlink(node.path)
	}
	unix.Close(node.fd)
	return OK
}

// shmShared is os_unix's unixIsSharingShmNode: whether another connection,
// in this process or another, has the -shm of this File open. Then the
// database must not be locked RESERVED or EXCLUSIVE in WAL mode, or a
// connection closing would delete the WAL and -shm under the other.
func (f *File) shmShared() bool {
	if f.shm == nil {
		return false
	}
	node := f.shm.node
	shmNodes.Lock()
	others := node.nRef > 1
	shmNodes.Unlock()
	if others {
		return true
	}
	node.mu.Lock()
	typ, err := testLock(node.fd, &node.fl, shmDMS, 1)
	node.mu.Unlock()
	return err == nil && typ != unix.F_UNLCK
}
