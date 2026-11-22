// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build windows && (amd64 || arm64)

package vfs

import (
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
)

// shmNode is the wal-index of one database in this process, as os_win's
// winShmNode: the "-shm" file next to the database, opened once and mapped
// in regions, shared by all the Files of that database. Its handle takes
// the DMS lock and makes the mappings. The slot locks are taken on a
// handle of each File (shm.h), so that Files of the process exclude each
// other through Windows as processes do. On a network drive, as os_win,
// they are taken on the node's handle instead, and the node sorts out the
// Files of the process.
type shmNode struct {
	key    string // the path in lower case, as os_win compares names
	path   string
	h      windows.Handle
	shared bool // slot locks on h: a network drive
	nRef   int  // Files using the node, under shmNodes

	mu       sync.Mutex
	readOnly bool // the -shm could only be opened for reading
	unlocked bool // the DMS lock is not held yet
	users    []*shm
	left     []slotRange  // ranges whose unlock failed, on any handle of the node
	leftN    atomic.Int32 // len(left), read without mu
	regions  []uintptr    // address of each region
	// mapped is regions as ShmMap may read it without mu: a copy of the
	// slice, stored under mu each time regions grows. Regions are only
	// added, and unmapped when the last File goes.
	mapped atomic.Pointer[[]uintptr]
	maps   []shmMapping
	ov     windows.Overlapped
}

type shmMapping struct {
	h    windows.Handle // of CreateFileMapping
	view uintptr        // of MapViewOfFile
}

// shmNodes are the open nodes, by path. The mutex is held to find, make,
// count and close nodes, as os_win's winShmEnterMutex.
var shmNodes struct {
	sync.Mutex
	m map[string]*shmNode
}

// shm is a File's use of the node, as os_win's winShm: its handle on the
// -shm and the slots it holds.
type shm struct {
	node   *shmNode
	h      windows.Handle
	shared uint16 // slots locked shared
	excl   uint16 // slots locked exclusive
	// The length of each exclusive lock, by its first slot. Windows
	// unlocks only the exact range that was locked.
	exclN [ShmNLock]uint8
	ov    windows.Overlapped
}

// slotRange is a range of slots locked in one call on a handle. A range
// whose unlock failed is still locked with no holder. It is unlocked again,
// as it was locked since Windows unlocks only the exact range, before any
// File of the process locks a slot, and the handle's close drops it at the
// latest.
type slotRange struct {
	h         windows.Handle
	offset, n int
}

// unlockFileEx is windows.UnlockFileEx, which tests make fail.
var unlockFileEx = windows.UnlockFileEx

// addLeft keeps a range whose unlock failed. node.mu is held.
func (node *shmNode) addLeft(r slotRange) {
	node.left = append(node.left, r)
	node.leftN.Store(int32(len(node.left)))
}

// retryLeft unlocks again the ranges whose unlock failed, on whichever
// handle, and keeps those that fail. It says whether none of those kept is
// on h with a slot of mask: h would stack its lock on such a range. A range
// on another handle needs no check, as the lock on h then fails or may be
// shared. node.mu is held.
func (node *shmNode) retryLeft(h windows.Handle, mask uint16) bool {
	ok := true
	kept := node.left[:0]
	for _, r := range node.left {
		ov := windows.Overlapped{Offset: uint32(shmBase + r.offset)}
		if unlockFileEx(r.h, 0, uint32(r.n), 0, &ov) != nil {
			kept = append(kept, r)
			if r.h == h && uint16(1<<(r.offset+r.n)-1<<r.offset)&mask != 0 {
				ok = false
			}
		}
	}
	clear(node.left[len(kept):])
	node.left = kept
	node.leftN.Store(int32(len(kept)))
	return ok
}

// forgetLeft drops the ranges of h, which is about to be closed: the close
// drops their locks, and the handle may be reused. node.mu is held.
func (node *shmNode) forgetLeft(h windows.Handle) {
	node.left = slices.DeleteFunc(node.left, func(r slotRange) bool { return r.h == h })
	node.leftN.Store(int32(len(node.left)))
}

// allocationGranularity is what a view of a file must start on.
var allocationGranularity = sync.OnceValue(func() int64 {
	var si struct {
		arch, _       uint16
		pageSize      uint32
		minApp        uintptr
		maxApp        uintptr
		activeMask    uintptr
		nProcessors   uint32
		processorType uint32
		granularity   uint32
		level, rev    uint16
	}
	windows.NewLazySystemDLL("kernel32.dll").NewProc("GetSystemInfo").Call(uintptr(unsafe.Pointer(&si)))
	if si.granularity == 0 {
		panic("vfs: GetSystemInfo gave no allocation granularity")
	}
	return int64(si.granularity)
})

// openShmHandle opens the -shm at path, as os_win's winHandleOpen: for
// reading and writing, or for reading only when it cannot be written or
// readOnly asks for it (readonly_shm=1).
func openShmHandle(path string, readOnly bool) (h windows.Handle, ro bool, rc int) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return windows.InvalidHandle, false, CantOpen
	}
	if isDir(p) {
		return windows.InvalidHandle, false, CantOpenIsDir
	}
	share := uint32(windows.FILE_SHARE_READ | windows.FILE_SHARE_WRITE)
	if !readOnly {
		h, err = windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, share, nil, windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL, 0)
		if err == nil {
			return h, false, OK
		}
	}
	h, err = windows.CreateFile(p, windows.GENERIC_READ, share, nil, windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return windows.InvalidHandle, true, CantOpen
	}
	return h, true, OK
}

// sharedShmForTest makes every new node use the shared handle, as on a
// network drive, as os_win's sqlite3_win_test_unc_locking.
var sharedShmForTest bool

// remote tells whether path is on a network drive, as os_win's winIsUNCPath
// and winIsRemoteVolume.
func remote(path string) bool {
	if strings.HasPrefix(path, `\\`) {
		rest := path[2:]
		// \\?\C:\ is a local path written long.
		if !(len(rest) >= 5 && rest[:2] == `?\` && isDriveLetter(rest[2]) && rest[3] == ':' && (rest[4] == '\\' || rest[4] == '/')) {
			return true
		}
		path = rest[2:]
	}
	if len(path) < 3 || !isDriveLetter(path[0]) || path[1] != ':' || path[2] != '\\' && path[2] != '/' {
		return false
	}
	root, _ := windows.UTF16PtrFromString(path[:2] + `\`)
	return windows.GetDriveType(root) == windows.DRIVE_REMOTE
}

// openShm attaches the File to the node of its database, as os_win's
// winOpenSharedMemory. The first File of the process opens the -shm. Each
// File opens a handle of its own for its slot locks.
func (f *File) openShm() int {
	path := f.path + "-shm"
	key := shmKey(path)

	shmNodes.Lock()
	node := shmNodes.m[key]
	if node == nil {
		h, ro, rc := openShmHandle(path, f.roShm)
		if rc != OK {
			shmNodes.Unlock()
			return rc
		}
		node = &shmNode{key: key, path: path, h: h, shared: sharedShmForTest || remote(f.path), readOnly: ro, unlocked: true}
		if shmNodes.m == nil {
			shmNodes.m = make(map[string]*shmNode)
		}
		shmNodes.m[key] = node
	}
	node.nRef++
	s := &shm{node: node, h: windows.InvalidHandle}
	node.mu.Lock()
	node.users = append(node.users, s)
	node.mu.Unlock()
	f.shm = s
	shmNodes.Unlock()

	if !node.shared {
		h, _, rc := openShmHandle(path, f.roShm)
		if rc != OK {
			f.ShmUnmap(false)
			return rc
		}
		s.h = h
	}
	return OK
}

// lockDMS takes a read lock on the DMS byte, as os_win's
// winLockSharedMemory. The DMS byte is read locked by every process that
// has the -shm open. When this process can write lock it, no one else has
// the -shm open, and it may hold anything a crash left: it is truncated
// first. node.mu is held.
func (node *shmNode) lockDMS() int {
	ov := &node.ov
	*ov = windows.Overlapped{Offset: shmDMS}
	if windows.LockFileEx(node.h, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, ov) == nil {
		rc := OK
		if node.readOnly {
			rc = ReadOnlyCantInit
		} else if windows.Ftruncate(node.h, 0) != nil {
			// The truncate can fail while another program has the file
			// mapped. Zeroing the header is enough to make it invalid.
			var zero [8]byte
			var n uint32
			*ov = windows.Overlapped{}
			if windows.WriteFile(node.h, zero[:], &n, ov) != nil || n != uint32(len(zero)) {
				rc = IOErrWrite
			}
		}
		*ov = windows.Overlapped{Offset: shmDMS}
		windows.UnlockFileEx(node.h, 0, 1, 0, ov)
		if rc != OK {
			return rc
		}
	}
	*ov = windows.Overlapped{Offset: shmDMS}
	if windows.LockFileEx(node.h, windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, ov) != nil {
		return Busy
	}
	node.unlocked = false
	return OK
}

// ShmMap returns the address of region n of the wal-index, each region size
// bytes, as os_win's winShmMap. When the region is not there yet, extend
// makes it, and without extend the address is 0 and the result OK. The
// memory stays mapped until ShmUnmap. With a -shm that can only be read,
// the result is SQLITE_READONLY with the address, or
// SQLITE_READONLY_CANTINIT when no other connection has it open.
func (f *File) ShmMap(n, size int, extend bool) (uintptr, int) {
	if f.shm == nil {
		if rc := f.openShm(); rc != OK {
			return 0, rc
		}
	}
	node := f.shm.node
	// A region mapped already is found without the mutex (docs/busy-cores.md
	// section 7). Regions are mapped after the DMS lock is taken, which then
	// stays. A read-only -shm goes the long way for its result.
	if !node.readOnly {
		if r := node.mapped.Load(); r != nil && n < len(*r) {
			return (*r)[n], OK
		}
	}
	node.mu.Lock()
	defer node.mu.Unlock()
	if node.unlocked {
		if rc := node.lockDMS(); rc != OK {
			return 0, rc
		}
	}
	rc := OK
	if len(node.regions) <= n {
		rc = node.mapRegions(n+1, size, extend)
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
// larger first when extend is set. Each region has a view of its own,
// started on the allocation granularity. node.mu is held.
func (node *shmNode) mapRegions(want, size int, extend bool) int {
	defer func() {
		r := node.regions[:len(node.regions):len(node.regions)]
		node.mapped.Store(&r)
	}()
	need := int64(want) * int64(size)
	var info windows.ByHandleFileInformation
	if windows.GetFileInformationByHandle(node.h, &info) != nil {
		return IOErrShmSize
	}
	if int64(info.FileSizeHigh)<<32|int64(info.FileSizeLow) < need {
		if !extend {
			return OK
		}
		if windows.Ftruncate(node.h, need) != nil {
			return IOErrShmSize
		}
	}
	protect, access := uint32(windows.PAGE_READWRITE), uint32(windows.FILE_MAP_READ|windows.FILE_MAP_WRITE)
	if node.readOnly {
		protect, access = windows.PAGE_READONLY, windows.FILE_MAP_READ
	}
	gran := allocationGranularity()
	for len(node.regions) < want {
		hMap, err := windows.CreateFileMapping(node.h, nil, protect, uint32(need>>32), uint32(need), nil)
		if err != nil {
			return IOErrShmMap
		}
		off := int64(len(node.regions)) * int64(size)
		shift := off % gran
		view, err := windows.MapViewOfFile(hMap, access, uint32((off-shift)>>32), uint32(off-shift), uintptr(int64(size)+shift))
		if err != nil {
			windows.CloseHandle(hMap)
			return IOErrShmMap
		}
		node.maps = append(node.maps, shmMapping{h: hMap, view: view})
		node.regions = append(node.regions, view+uintptr(shift))
	}
	return OK
}

// ShmLock locks or unlocks n slots of the wal-index from offset, as os_win's
// winShmLock. flags is SQLITE_SHM_LOCK or SQLITE_SHM_UNLOCK with
// SQLITE_SHM_SHARED or SQLITE_SHM_EXCLUSIVE. A shared lock is on one slot.
func (f *File) ShmLock(offset, n, flags int) int {
	s := f.shm
	if s == nil {
		return IOErrShmLock
	}
	mask := uint16(1<<(offset+n) - 1<<offset)
	unlock := flags&ShmUnlock != 0
	excl := flags&ShmExclusive != 0
	switch {
	case unlock && (s.excl|s.shared)&mask == 0:
		return OK
	case !unlock && !excl && s.shared&mask != 0:
		return OK
	}
	node := s.node
	h := s.h
	if node.shared {
		// One handle for the process: what the other Files hold, the
		// lock on the handle and this File's bits change together.
		h = node.h
		node.mu.Lock()
		defer node.mu.Unlock()
	}
	if unlock {
		if node.shared && !excl && node.otherReads(s, mask) {
			// Another File of the process still reads the slot.
			h = windows.InvalidHandle
		}
		rc := OK
		if h != windows.InvalidHandle {
			s.ov = windows.Overlapped{Offset: uint32(shmBase + offset)}
			if unlockFileEx(h, 0, uint32(n), 0, &s.ov) != nil {
				// The File lets go of the slots all the same, as SQLite
				// takes it to: the range is kept to unlock again.
				if !node.shared {
					node.mu.Lock()
				}
				node.addLeft(slotRange{h, offset, n})
				if !node.shared {
					node.mu.Unlock()
				}
				rc = IOErrUnlock
			}
		}
		s.excl &^= mask
		s.shared &^= mask
		return rc
	}
	if node.shared {
		for _, o := range node.users {
			if excl && (o.shared|o.excl)&mask != 0 || !excl && o.excl&mask != 0 {
				return Busy
			}
			if !excl && o.shared&mask != 0 {
				h = windows.InvalidHandle
			}
		}
	}
	if h != windows.InvalidHandle {
		if node.leftN.Load() > 0 {
			if !node.shared {
				node.mu.Lock()
			}
			ok := node.retryLeft(h, mask)
			if !node.shared {
				node.mu.Unlock()
			}
			if !ok {
				return Busy
			}
		}
		lf := uint32(windows.LOCKFILE_FAIL_IMMEDIATELY)
		if excl {
			lf |= windows.LOCKFILE_EXCLUSIVE_LOCK
		}
		s.ov = windows.Overlapped{Offset: uint32(shmBase + offset)}
		if windows.LockFileEx(h, lf, 0, uint32(n), 0, &s.ov) != nil {
			return Busy
		}
	}
	if excl {
		s.excl |= mask
		s.exclN[offset] = uint8(n)
	} else {
		s.shared |= mask
	}
	return OK
}

// otherReads tells whether a File of the node other than s holds a slot of
// mask shared. node.mu is held.
func (node *shmNode) otherReads(s *shm, mask uint16) bool {
	for _, o := range node.users {
		if o != s && o.shared&mask != 0 {
			return true
		}
	}
	return false
}

// ShmUnmap detaches the File from the wal-index, as os_win's
// winShmUnmap. Its slot locks are released. The last File of the process
// unmaps the -shm and closes it, and deletes it when remove is set.
func (f *File) ShmUnmap(remove bool) int {
	s := f.shm
	if s == nil {
		return OK
	}
	f.shm = nil
	node := s.node
	// Locks still held are released here, each as it was taken: Windows
	// drops the locks of a closed handle only when it gets to it.
	f.shm = s
	for i := range ShmNLock {
		switch bit := uint16(1) << i; {
		case s.excl&bit != 0 && s.exclN[i] > 0:
			f.ShmLock(i, int(s.exclN[i]), ShmUnlock|ShmExclusive)
		case s.shared&bit != 0:
			f.ShmLock(i, 1, ShmUnlock|ShmShared)
		}
	}
	f.shm = nil
	if s.h != windows.InvalidHandle {
		// What could not be unlocked goes when Windows gets to the close.
		node.mu.Lock()
		node.retryLeft(s.h, 0)
		node.forgetLeft(s.h)
		node.mu.Unlock()
		windows.CloseHandle(s.h)
		s.h = windows.InvalidHandle
	}

	shmNodes.Lock()
	node.mu.Lock()
	if len(node.left) > 0 {
		node.retryLeft(node.h, 0)
	}
	for i, o := range node.users {
		if o == s {
			node.users = append(node.users[:i], node.users[i+1:]...)
			break
		}
	}
	node.mu.Unlock()
	node.nRef--
	if node.nRef > 0 {
		shmNodes.Unlock()
		return OK
	}
	delete(shmNodes.m, node.key)
	for _, m := range node.maps {
		windows.UnmapViewOfFile(m.view)
		windows.CloseHandle(m.h)
	}
	node.maps, node.regions = nil, nil
	if !node.unlocked {
		node.ov = windows.Overlapped{Offset: shmDMS}
		windows.UnlockFileEx(node.h, 0, 1, 0, &node.ov)
	}
	windows.CloseHandle(node.h)
	shmNodes.Unlock()
	// The delete can wait for a virus scanner: not with every open and
	// close of the process held up.
	if remove {
		deleteFile(node.path)
	}
	return OK
}

// shmKey is the key of the shared memory node of path. Like os_win, which
// compares the paths with sqlite3StrICmp, it folds ASCII letters only:
// strings.ToLower folds more, and would make one node of two files Windows
// tells apart, such as İ.db and i.db.
func shmKey(path string) string {
	b := []byte(path)
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}
