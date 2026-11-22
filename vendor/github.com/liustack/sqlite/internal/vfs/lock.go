// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build (linux || darwin) && (amd64 || arm64)

package vfs

import (
	"sync"

	"golang.org/x/sys/unix"
)

// The lock levels are made of byte locks, as os_unix's unixLock:
//
//	SHARED     read lock on the shared range, taken while holding a read
//	           lock on the pending byte
//	RESERVED   SHARED, and a write lock on the reserved byte
//	PENDING    RESERVED, and a write lock on the pending byte: no new SHARED
//	EXCLUSIVE  PENDING (or SHARED), and a write lock on the shared range
//
// The byte locks belong to the process, held on the descriptor of the
// inode. Between the Files of the process, the inode decides in memory, as
// os_unix's unixInodeInfo. Nothing ever waits: a conflict is SQLITE_BUSY.

// inode is the main database file of one or more Files of the process, as
// os_unix's unixInodeInfo. Its descriptor holds the process's locks on the
// file, and it counts the Files that hold SHARED.
type inode struct {
	id   fileID
	nRef int // Files of the inode, under inodes

	mu       sync.Mutex
	fd       int          // holds the locks
	readOnly bool         // fd cannot take write locks
	lock     int          // the highest lock a File of the process holds
	nShared  int          // Files holding SHARED or more
	fl       unix.Flock_t // space for the fcntl argument
}

// inodes are the inodes of open main databases. The mutex is held to find,
// make, count and close them.
var inodes struct {
	sync.Mutex
	m map[fileID]*inode
}

// attachInode gives the File the inode of its main database, made if this
// is the first File of it in the process. name is the path it was opened
// with.
func (f *File) attachInode(name string) int {
	id := fileID{f.dev, f.ino}
	inodes.Lock()
	defer inodes.Unlock()
	node := inodes.m[id]
	if node == nil {
		fd, readOnly, err := lockDescriptor(f, name)
		if err != nil {
			return CantOpen
		}
		node = &inode{id: id, fd: fd, readOnly: readOnly}
		if inodes.m == nil {
			inodes.m = make(map[fileID]*inode)
		}
		inodes.m[id] = node
	} else if node.readOnly && !f.readOnly {
		// Only read-only Files came before.
		node.mu.Lock()
		node.takeWritable(f.fd)
		node.mu.Unlock()
	}
	node.nRef++
	f.inode = node
	return OK
}

// takeWritable gives a read-only inode a copy of fd, a read-write
// descriptor of the same file, so that write locks can be taken. When
// Files of the process hold SHARED, the read lock moves to the new
// descriptor. A read-only descriptor never holds more than that, since
// every write lock fails on it. Two open file descriptions of the process
// may both hold a read lock for a moment, and no other process can hold a
// write lock on the shared range while the process reads, so the move
// cannot be refused for a conflict. If it fails all the same, the inode
// keeps its descriptor and only write locks fail, as before. node.mu is
// held.
func (node *inode) takeWritable(fd int) {
	nfd, err := dupDescriptor(fd)
	if err != nil {
		return
	}
	if node.lock == LockShared {
		if setLock(nfd, &node.fl, unix.F_RDLCK, sharedFirst, sharedSize) != nil {
			unix.Close(nfd)
			return
		}
		// The old descriptor may be a copy of a read-only File's, whose
		// description outlives it: closing it would not drop its lock.
		// The same holds for nfd and the new File.
		if setLock(node.fd, &node.fl, unix.F_UNLCK, 0, 0) != nil {
			setLock(nfd, &node.fl, unix.F_UNLCK, 0, 0)
			unix.Close(nfd)
			return
		}
	}
	unix.Close(node.fd)
	node.fd, node.readOnly = nfd, false
}

// lockDescriptor opens the descriptor the inode holds its locks on: a copy
// of the File's, so that it is the same file and closing the File's does
// not close it. For a read-only File it first tries to open the file for
// writing, so that read-write Files that come later can take write locks.
func lockDescriptor(f *File, name string) (fd int, readOnly bool, err error) {
	if f.readOnly {
		var st unix.Stat_t
		fd, err := openFile(name, unix.O_RDWR|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if err == nil {
			if unix.Fstat(fd, &st) == nil && uint64(st.Dev) == f.dev && uint64(st.Ino) == f.ino {
				return fd, false, nil
			}
			unix.Close(fd)
		}
	}
	fd, err = dupDescriptor(f.fd)
	return fd, f.readOnly, err
}

// dupDescriptor copies fd to a number above 2 (see aboveStdio).
func dupDescriptor(fd int) (int, error) {
	return unix.FcntlInt(uintptr(fd), unix.F_DUPFD_CLOEXEC, 3)
}

// detachInode is the end of attachInode. The last File closes the inode's
// descriptor. Its locks are gone by then: every File unlocks before.
func (f *File) detachInode() {
	node := f.inode
	f.inode = nil
	inodes.Lock()
	defer inodes.Unlock()
	node.nRef--
	if node.nRef == 0 {
		delete(inodes.m, node.id)
		unix.Close(node.fd)
	}
}

// Lock raises the lock to level: SHARED, RESERVED or EXCLUSIVE. SQLite
// never asks for PENDING. When EXCLUSIVE fails from RESERVED, the File keeps
// PENDING, so that the next try does not race new readers.
func (f *File) Lock(level int) int {
	if f.lock >= level || f.noLock {
		return OK
	}
	node := f.inode
	node.mu.Lock()
	defer node.mu.Unlock()

	// Another File of the process holds a lock that keeps this one out.
	if f.lock != node.lock && (node.lock >= LockPending || level > LockShared) {
		return Busy
	}
	// Other Files of the process read: join them.
	if level == LockShared && (node.lock == LockShared || node.lock == LockReserved) {
		f.lock = LockShared
		node.nShared++
		return OK
	}

	// SHARED, and EXCLUSIVE from RESERVED, first take the pending byte.
	if level == LockShared || level == LockExclusive && f.lock == LockReserved {
		typ := int16(unix.F_WRLCK)
		if level == LockShared {
			typ = unix.F_RDLCK
		}
		if err := setLock(node.fd, &node.fl, typ, pendingByte, 1); err != nil {
			return lockError(err, IOErrLock)
		}
		if level == LockExclusive {
			f.lock = LockPending
			node.lock = LockPending
		}
	}

	switch {
	case level == LockShared:
		// No File of the process holds a lock.
		rc := OK
		if err := setLock(node.fd, &node.fl, unix.F_RDLCK, sharedFirst, sharedSize); err != nil {
			rc = lockError(err, IOErrLock)
		}
		// The read lock on the pending byte was only needed to get in.
		if err := setLock(node.fd, &node.fl, unix.F_UNLCK, pendingByte, 1); err != nil && rc == OK {
			rc = IOErrUnlock
		}
		if rc != OK {
			return rc
		}
		node.nShared = 1

	case level == LockExclusive && node.nShared > 1:
		// Another File of the process still reads.
		return Busy

	case f.shmShared():
		// In WAL mode, another connection still has the -shm open.
		return Busy

	case f.readOnly:
		// What fcntl says of a write lock on a read-only descriptor.
		return IOErrLock

	default:
		start, n := int64(reservedByte), int64(1)
		if level == LockExclusive {
			start, n = sharedFirst, sharedSize
		}
		if err := setLock(node.fd, &node.fl, unix.F_WRLCK, start, n); err != nil {
			return lockError(err, IOErrLock)
		}
	}
	f.lock = level
	node.lock = level
	if level == LockExclusive && f.shm != nil {
		f.shm.node.exclusive(true)
	}
	return OK
}

// Unlock lowers the lock to level: SHARED or NONE. The process keeps its
// read lock while another of its Files holds SHARED.
func (f *File) Unlock(level int) int {
	if f.lock <= level || f.noLock {
		return OK
	}
	node := f.inode
	node.mu.Lock()
	defer node.mu.Unlock()

	if f.lock > LockShared {
		if f.shm != nil {
			// Before the lock goes: from then another process may come.
			f.shm.node.exclusive(false)
		}
		if level == LockShared {
			// Turn the write lock on the shared range back into a read
			// lock. This cannot conflict: another owner that does is not
			// keeping to the protocol.
			if err := setLock(node.fd, &node.fl, unix.F_RDLCK, sharedFirst, sharedSize); err != nil {
				return IOErrRdlock
			}
		}
		// The pending and reserved bytes.
		if err := setLock(node.fd, &node.fl, unix.F_UNLCK, pendingByte, 2); err != nil {
			return IOErrUnlock
		}
		f.lock = LockShared
		node.lock = LockShared
	}
	if level == LockNone {
		f.lock = LockNone
		node.nShared--
		if node.nShared == 0 {
			node.lock = LockNone
			if err := setLock(node.fd, &node.fl, unix.F_UNLCK, 0, 0); err != nil {
				return IOErrUnlock
			}
		}
	}
	return OK
}

// CheckReservedLock tells whether some File, of this process or another,
// holds RESERVED or more.
func (f *File) CheckReservedLock() (bool, int) {
	if f.noLock {
		return false, OK
	}
	node := f.inode
	node.mu.Lock()
	defer node.mu.Unlock()
	if node.lock > LockShared {
		return true, OK
	}
	// The locks of the inode's own descriptor are not seen here, which is
	// right: the process holds no more than SHARED.
	typ, err := testLock(node.fd, &node.fl, reservedByte, 1)
	if err != nil {
		return false, IOErrCheckReservedLock
	}
	return typ != unix.F_UNLCK, OK
}

// LockState is the lock level held, for SQLITE_FCNTL_LOCKSTATE.
func (f *File) LockState() int {
	return f.lock
}

// fcntlHook, when set by a test, is called before each lock fcntl with its
// lock type, -1 for a test of a lock, and its range. An error it returns is
// returned in place of the call, as if fcntl had failed.
var fcntlHook func(typ int16, start, n int64) error

// setLock sets a lock of typ (F_RDLCK, F_WRLCK or F_UNLCK) on n bytes from
// start, without waiting. n 0 is to the end of the file. fl is space for the
// argument.
func setLock(fd int, fl *unix.Flock_t, typ int16, start, n int64) error {
	if fcntlHook != nil {
		if err := fcntlHook(typ, start, n); err != nil {
			return err
		}
	}
	*fl = unix.Flock_t{Type: typ, Whence: 0, Start: start, Len: n}
	for {
		err := unix.FcntlFlock(uintptr(fd), fOFDSetlk, fl)
		if err != unix.EINTR {
			return err
		}
	}
}

// testLock tells which lock another owner holds that would stop a write
// lock on n bytes from start: F_UNLCK, F_RDLCK or F_WRLCK. The locks of
// this open file description do not count.
func testLock(fd int, fl *unix.Flock_t, start, n int64) (int16, error) {
	if fcntlHook != nil {
		if err := fcntlHook(-1, start, n); err != nil {
			return 0, err
		}
	}
	*fl = unix.Flock_t{Type: unix.F_WRLCK, Whence: 0, Start: start, Len: n}
	for {
		err := unix.FcntlFlock(uintptr(fd), fOFDGetlk, fl)
		if err == nil {
			return fl.Type, nil
		}
		if err != unix.EINTR {
			return 0, err
		}
	}
}

// lockError is os_unix's sqliteErrorFromPosixError: SQLITE_BUSY for a lock
// held by another owner, def for the rest.
func lockError(err error, def int) int {
	switch err {
	case unix.EACCES, unix.EAGAIN, unix.ETIMEDOUT, unix.EBUSY, unix.EINTR, unix.ENOLCK:
		return Busy
	case unix.EPERM:
		return Perm
	}
	return def
}
