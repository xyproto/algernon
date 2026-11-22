// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build (linux || darwin) && (amd64 || arm64)

// Package vfs is the operating system layer of SQLite, in Go: the files,
// locks and shared memory that os_unix.c gives a normal build. SQLite is
// built with SQLITE_OS_OTHER, and a small C shim turns its sqlite3_vfs and
// sqlite3_io_methods calls into calls of this package.
//
// It behaves like os_unix.c, and keeps locks the same way: the Files of one
// database in the process share a node (see inode and shmNode) that counts
// in memory which File holds what, refuses conflicts between them without
// asking the kernel, and holds the process's locks on one descriptor. The
// kernel is asked only when the process's lock changes.
//
// The difference is the kind of kernel lock. os_unix uses POSIX record
// locks, which belong to the process, and closing any descriptor of a file
// drops all of them, so os_unix keeps the descriptors of closed connections
// open until the last lock goes. This package uses locks on open file
// descriptions (OFD locks), which belong to the node's own descriptor:
// closing a File's descriptor drops nothing, so it is closed at once. The
// byte ranges are those of os_unix, so the locks also work against SQLite
// built from C, as long as that SQLite runs in another process.
//
// One more difference: while no other process has the -shm open, the last
// reader of the process to leave one of the WAL read slots leaves the lock
// on the slot for about a millisecond, so that the next reader needs no
// system call. When another process opens the -shm, this process finds out
// within about that millisecond and from then on releases read slots at
// once, as os_unix, so that the other process's checkpoints are not held
// off (see shmReadSlots).
//
// Results are SQLite result codes, with the values of sqlite3.h.
//
// One File is used by one goroutine at a time. Different Files, also of the
// same path, may be used at the same time.
package vfs

import (
	"math/rand/v2"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// File is an open file, as os_unix's unixFile.
type File struct {
	fd       int
	path     string // the name given to Open, "" for a temporary file
	lock     int    // the lock level held
	noLock   bool   // not a main database: Lock and Unlock do nothing, as os_unix
	readOnly bool   // opened for reading only
	roShm    bool   // readonly_shm=1
	dirSync  bool   // a new journal: sync its directory on the first Sync
	chunk    int64  // SQLITE_FCNTL_CHUNK_SIZE
	dev      uint64 // device and inode when opened, for HasMoved
	ino      uint64
	inode    *inode // the locks of the main database, shared in the process
	wal      *wal   // a WAL: what it shares with the other Files of the WAL
	journal  bool   // a rollback journal
	dirty    int64  // a rollback journal: the first byte written since its last good sync
	stopped  bool   // a rollback journal whose sync failed: see journal_unix.go
	shm      *shm

	// Space for the argument of fstat, so that the calls made for every
	// page do not allocate.
	st unix.Stat_t
}

// OpenWith opens name with SQLITE_OPEN_* flags and the options of its URI,
// as os_unix's unixOpen. An empty name is a temporary file, deleted when
// closed. outFlags are the flags the file was opened with: READONLY in
// place of READWRITE and CREATE when it could only be opened for reading.
func OpenWith(name string, flags int, o OpenOptions) (f *File, outFlags int, rc int) {
	eType := flags & openTypeMask
	isExclusive := flags&OpenExclusive != 0
	isDelete := flags&OpenDeleteOnClose != 0
	isCreate := flags&OpenCreate != 0
	isReadWrite := flags&OpenReadWrite != 0
	// A new journal or WAL: its directory entry is synced on the first Sync.
	isNewJrnl := isCreate && (eType == OpenSuperJournal || eType == OpenMainJournal || eType == OpenWAL)

	f = &File{fd: -1, path: name, noLock: eType != OpenMainDB, roShm: o.ReadOnlyShm}

	fileName := name
	if name == "" {
		dir, rc := tempDir()
		if rc != OK {
			return nil, 0, rc
		}
		if fd, err := openTempFile(dir); err == nil {
			f.fd = fd
			return f, flags, OK
		}
		if fileName, rc = tempName(dir); rc != OK {
			return nil, 0, rc
		}
	}

	oflags := unix.O_NOFOLLOW | unix.O_CLOEXEC
	if isReadWrite {
		oflags |= unix.O_RDWR
	} else {
		oflags |= unix.O_RDONLY
	}
	if isCreate {
		oflags |= unix.O_CREAT
	}
	if isExclusive {
		oflags |= unix.O_EXCL
	}

	mode, uid, gid, rc := createMode(fileName, flags, o.ModeOf)
	if rc != OK {
		return nil, 0, rc
	}
	fd, err := openFile(fileName, oflags, mode)
	if err != nil {
		switch {
		case isNewJrnl && err == unix.EACCES && !exists(fileName):
			// The directory cannot be written.
			return nil, 0, ReadOnlyDirectory
		case err != unix.EISDIR && isReadWrite && !(isExclusive && err == unix.EEXIST):
			// Open for reading only. os_unix also does this when an
			// exclusive create finds the file there. This package does
			// not: EXCLUSIVE means the file must be new.
			flags = flags&^(OpenReadWrite|OpenCreate) | OpenReadOnly
			oflags = oflags&^(unix.O_RDWR|unix.O_CREAT|unix.O_EXCL) | unix.O_RDONLY
			fd, err = openFile(fileName, oflags, mode)
		}
		if err != nil {
			return nil, 0, CantOpen
		}
	}
	f.fd = fd
	if isCreate && !isDelete {
		tell(ChangeCreate, name, 0, nil)
	}

	// A journal or WAL made by root is given to the owner of the database,
	// or the owner could no longer open the database.
	if mode != 0 && flags&(OpenWAL|OpenMainJournal) != 0 && os.Geteuid() == 0 {
		unix.Fchown(fd, uid, gid)
	}
	if isDelete {
		unix.Unlink(fileName)
	}
	if !f.noLock {
		if err := unix.Fstat(fd, &f.st); err != nil {
			unix.Close(fd)
			return nil, 0, IOErrFstat
		}
		f.dev, f.ino = uint64(f.st.Dev), uint64(f.st.Ino)
		f.readOnly = flags&OpenReadOnly != 0
		if rc := f.attachInode(fileName); rc != OK {
			unix.Close(fd)
			return nil, 0, rc
		}
	}
	if eType == OpenWAL {
		if rc := f.attachWAL(); rc != OK {
			unix.Close(fd)
			return nil, 0, rc
		}
	}
	if eType == OpenMainJournal {
		if rc := f.openJournal(); rc != OK {
			unix.Close(fd)
			return nil, 0, rc
		}
		f.journal, f.dirty = true, clean
	}
	f.dirSync = isNewJrnl
	return f, flags, OK
}

// createMode is os_unix's findCreateFileMode: the mode to create a file
// with, 0 for the default. A journal or WAL takes the mode, owner and group
// of its database file, and a main database opened through a URI those of
// modeof, when given.
func createMode(name string, flags int, modeOf string) (mode uint32, uid, gid int, rc int) {
	switch {
	case flags&(OpenWAL|OpenMainJournal) != 0:
		// "<db>-journal", "<db>-wal", "<db>-journalNN" or "<db>-walNN".
		for i := len(name) - 1; i > 0 && name[i] != '.'; i-- {
			if name[i] == '-' {
				var st unix.Stat_t
				if err := unix.Stat(name[:i], &st); err != nil {
					return 0, 0, 0, IOErrFstat
				}
				return uint32(st.Mode) & 0777, int(st.Uid), int(st.Gid), OK
			}
		}
	case flags&OpenDeleteOnClose != 0:
		return 0600, 0, 0, OK
	case flags&OpenURI != 0 && modeOf != "":
		var st unix.Stat_t
		if err := unix.Stat(modeOf, &st); err != nil {
			return 0, 0, 0, IOErrFstat
		}
		return uint32(st.Mode) & 0777, int(st.Uid), int(st.Gid), OK
	}
	return 0, 0, 0, OK
}

// openFile is os_unix's robust_open: open, and when the file was made with
// a mode the umask changed, give it the mode asked for.
func openFile(name string, oflags int, mode uint32) (int, error) {
	m := mode
	if m == 0 {
		m = defaultFileMode
	}
	fd, err := openFD(name, oflags, m)
	if err != nil {
		return -1, err
	}
	if mode != 0 {
		var st unix.Stat_t
		if unix.Fstat(fd, &st) == nil && st.Size == 0 && uint32(st.Mode)&0777 != mode {
			unix.Fchmod(fd, mode)
		}
	}
	return fd, nil
}

// openFD opens name, again when a signal interrupts it, and never as
// descriptor 0, 1 or 2 (see aboveStdio).
func openFD(name string, oflags int, mode uint32) (int, error) {
	var fd int
	err := retry(func() (err error) {
		fd, err = unix.Open(name, oflags, mode)
		return err
	})
	if err != nil {
		return -1, err
	}
	return aboveStdio(fd)
}

// aboveStdio moves a descriptor that got number 0, 1 or 2 to a number
// above, as os_unix's robust_open. A program that closed its stdin, stdout
// or stderr gives that number to the next file opened, and what is then
// written to stdout or stderr (the Go runtime writes panics to 2) would go
// into the database. The old number is given /dev/null, so that later
// opens do not land there. os_unix closes the file and opens it again. A
// copy keeps the same open file, so that a file just made with O_EXCL is
// not lost.
func aboveStdio(fd int) (int, error) {
	if fd > 2 {
		return fd, nil
	}
	nfd, err := unix.FcntlInt(uintptr(fd), unix.F_DUPFD_CLOEXEC, 3)
	if err != nil {
		unix.Close(fd)
		return -1, err
	}
	// Without O_CLOEXEC: it stands for a standard stream, which children
	// inherit.
	null, err := unix.Open("/dev/null", unix.O_RDWR, 0)
	if err != nil {
		// The number is left free.
		unix.Close(fd)
		return nfd, nil
	}
	// Dup2 puts /dev/null in place of fd in one step. A null that itself
	// got a low number fills that one too and is kept.
	unix.Dup2(null, fd)
	if null > 2 {
		unix.Close(null)
	}
	return nfd, nil
}

// tempDir is os_unix's unixTempFileDir: the first directory that can be
// written of SQLITE_TMPDIR, TMPDIR, /var/tmp, /usr/tmp, /tmp and ".".
func tempDir() (string, int) {
	for _, dir := range [...]string{os.Getenv("SQLITE_TMPDIR"), os.Getenv("TMPDIR"), "/var/tmp", "/usr/tmp", "/tmp", "."} {
		if dir == "" {
			continue
		}
		var st unix.Stat_t
		if unix.Stat(dir, &st) == nil && st.Mode&unix.S_IFMT == unix.S_IFDIR &&
			unix.Access(dir, unix.W_OK|unix.X_OK) == nil {
			return dir, OK
		}
	}
	return "", IOErrGetTempPath
}

// tempName is os_unix's unixGetTempname: a random name not in use in dir.
func tempName(dir string) (string, int) {
	for range 11 {
		name := dir + "/" + tempFilePrefix + strconv.FormatUint(rand.Uint64(), 16)
		if !exists(name) {
			return name, OK
		}
	}
	return "", Error
}

func exists(name string) bool {
	return unix.Access(name, unix.F_OK) == nil
}

// retry calls fn again for as long as a signal interrupts it.
func retry(fn func() error) error {
	for {
		if err := fn(); err != unix.EINTR {
			return err
		}
	}
}

// Close closes the file, and its shared memory if ShmUnmap was not called.
// The locks the File holds are released, and those of other Files of the
// database are kept. Closing twice is harmless.
func (f *File) Close() int {
	if f.shm != nil {
		f.ShmUnmap(false)
	}
	if f.inode != nil {
		f.Unlock(LockNone)
		f.detachInode()
	}
	if f.wal != nil {
		f.detachWAL()
	}
	if f.stopped {
		f.closeJournal()
	}
	if f.fd >= 0 {
		// As os_unix, a failed close is not an error for SQLite.
		unix.Close(f.fd)
		f.fd = -1
	}
	return OK
}

// ReadAt reads len(p) bytes at off. When the file ends first, the rest of p
// is zeroed and the result is SQLITE_IOERR_SHORT_READ.
func (f *File) ReadAt(p []byte, off int64) int {
	if f.stopped || f.wal != nil && f.wal.failed.Load() {
		return IOErrRead
	}
	got := 0
	for got < len(p) {
		n, err := unix.Pread(f.fd, p[got:], off+int64(got))
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			if corruptFS(err) {
				return IOErrCorruptFS
			}
			return IOErrRead
		}
		if n == 0 {
			clear(p[got:])
			return IOErrShortRead
		}
		got += n
	}
	return OK
}

// WriteAt writes p at off. A full disk is SQLITE_FULL.
func (f *File) WriteAt(p []byte, off int64) int {
	if f.stopped || f.shm != nil && !f.shm.node.copies() {
		return IOErrWrite
	}
	if f.journal && off < f.dirty {
		f.dirty = off
	}
	if w := f.wal; w != nil {
		w.io.RLock()
		defer w.io.RUnlock()
		if w.failed.Load() {
			return IOErrWrite
		}
		w.wrote(off)
	}
	if rc := f.faultOf(OpWrite); rc != OK {
		return rc
	}
	all, at := p, off
	for len(p) > 0 {
		n, err := unix.Pwrite(f.fd, p, off)
		if err == unix.EINTR {
			continue
		}
		if err == unix.ENOSPC || err == nil && n == 0 {
			return Full
		}
		if err != nil {
			return IOErrWrite
		}
		p = p[n:]
		off += int64(n)
	}
	tell(ChangeWrite, f.path, at, all)
	return OK
}

// Truncate sets the size of the file, rounded up to whole chunks when
// SetChunkSize was called.
func (f *File) Truncate(size int64) int {
	if f.stopped || f.shm != nil && !f.shm.node.copies() {
		return IOErrTruncate
	}
	if w := f.wal; w != nil {
		w.io.RLock()
		defer w.io.RUnlock()
		if w.failed.Load() {
			return IOErrTruncate
		}
	}
	if rc := f.faultOf(OpTruncate); rc != OK {
		return rc
	}
	if f.chunk > 0 {
		size = (size + f.chunk - 1) / f.chunk * f.chunk
	}
	if err := retry(func() error { return unix.Ftruncate(f.fd, size) }); err != nil {
		return IOErrTruncate
	}
	if f.journal && size < f.dirty {
		f.dirty = size
	}
	tell(ChangeTruncate, f.path, size, nil)
	return OK
}

// Sync writes the file to disk, and the directory entry of a new journal
// the first time. See fullFsync for what each system does. A failed sync of
// a WAL stops the WAL: see wal_unix.go. A failed sync of a rollback journal
// stops the File: see journal_unix.go.
func (f *File) Sync(flags int) int {
	w := f.wal
	if w == nil {
		if f.stopped {
			return IOErrFsync
		}
		rc := f.sync(flags)
		switch {
		case !f.journal:
		case rc == OK:
			f.dirty = clean
		default:
			f.stopJournal()
		}
		return rc
	}
	w.io.Lock()
	defer w.io.Unlock()
	if w.failed.Load() {
		return IOErrFsync
	}
	dirty := w.dirty.Swap(clean)
	rc := f.sync(flags)
	if rc != OK {
		w.fail(f, dirty)
	}
	return rc
}

// sync is Sync without what a WAL needs.
func (f *File) sync(flags int) int {
	if rc := f.faultOf(OpSync); rc != OK {
		return rc
	}
	tell(ChangeSync, f.path, 0, nil)
	if err := fullFsync(f.fd, flags&0x0F == SyncFull); err != nil {
		return IOErrFsync
	}
	if f.dirSync {
		// As os_unix, a directory that cannot be opened or synced is
		// not an error: some systems cannot sync a directory.
		if dir, err := openDirectory(f.path); err == nil {
			tell(ChangeDirSync, f.path, 0, nil)
			fullFsync(dir, false)
			unix.Close(dir)
		}
		f.dirSync = false
	}
	return OK
}

// openDirectory opens the directory that holds name.
func openDirectory(name string) (int, error) {
	dir := "."
	if i := strings.LastIndexByte(name, '/'); i > 0 {
		dir = name[:i]
	} else if i == 0 {
		dir = "/"
	}
	return openFD(dir, unix.O_RDONLY|unix.O_CLOEXEC, 0)
}

// Size is the size of the file. As os_unix, a size of 1 is given as 0: a
// work around for the msdos file system of OS X, kept so that SQLite sees
// the same sizes it would see with os_unix.
func (f *File) Size() (int64, int) {
	if f.stopped {
		return 0, IOErrFstat
	}
	if err := unix.Fstat(f.fd, &f.st); err != nil {
		return 0, IOErrFstat
	}
	if f.st.Size == 1 {
		return 0, OK
	}
	return f.st.Size, OK
}

// SetChunkSize is SQLITE_FCNTL_CHUNK_SIZE: Truncate and SizeHint round the
// size up to whole chunks of n bytes. 0 turns it off.
func (f *File) SetChunkSize(n int) {
	f.chunk = int64(n)
}

// SizeHint is SQLITE_FCNTL_SIZE_HINT, as os_unix's fcntlSizeHint: with a
// chunk size set, make the file at least size bytes, rounded up to whole
// chunks. Without a chunk size it does nothing, as os_unix when memory
// mapped I/O is off.
func (f *File) SizeHint(size int64) int {
	if f.chunk <= 0 {
		return OK
	}
	if err := unix.Fstat(f.fd, &f.st); err != nil {
		return IOErrFstat
	}
	n := (size + f.chunk - 1) / f.chunk * f.chunk
	if n > f.st.Size {
		err := allocate(f.fd, f.st.Size, n, int64(f.st.Blksize))
		if err != nil && err != unix.EINVAL {
			return IOErrWrite
		}
		if err == nil {
			tell(ChangeTruncate, f.path, n, nil)
		}
	}
	return OK
}

// allocateByWrites makes the file size bytes long by writing a byte at the
// end of each block from from on, as os_unix does without posix_fallocate.
func allocateByWrites(fd int, from, size int64, blksize int64) error {
	if blksize <= 0 {
		blksize = 4096
	}
	zero := []byte{0}
	for i := from/blksize*blksize + blksize - 1; i < size+blksize-1; i += blksize {
		if i >= size {
			i = size - 1
		}
		var n int
		err := retry(func() (err error) {
			n, err = unix.Pwrite(fd, zero, i)
			return err
		})
		if err != nil {
			return err
		}
		if n != 1 {
			return unix.ENOSPC
		}
	}
	return nil
}

// HasMoved is SQLITE_FCNTL_HAS_MOVED: the path the main database was opened
// with no longer names this file (renamed, deleted or replaced).
func (f *File) HasMoved() bool {
	if f.noLock {
		return false
	}
	var st unix.Stat_t
	if unix.Stat(f.path, &st) != nil {
		return true
	}
	return uint64(st.Ino) != f.ino || uint64(st.Dev) != f.dev
}

// SectorSize is os_unix's SQLITE_DEFAULT_SECTOR_SIZE.
func (f *File) SectorSize() int {
	return sectorSize
}

// DeviceCharacteristics gives the SQLITE_IOCAP_* bits of the file as os_unix
// does, but powersafe overwrite: engine/os.c adds that one, which the psow
// URI parameter turns off. Subpage read lets SQLite read a large value from
// its overflow pages straight into its buffer.
func (f *File) DeviceCharacteristics() int {
	return IOCapSubpageRead
}

// Delete deletes name, and syncs its directory when syncDir is set.
func Delete(name string, syncDir bool) int {
	err := retry(func() error { return unix.Unlink(name) })
	if err == unix.ENOENT {
		return IOErrDeleteNoent
	}
	if err != nil {
		return IOErrDelete
	}
	tell(ChangeDelete, name, 0, nil)
	if syncDir {
		// As os_unix, a directory that cannot be opened is not an error.
		if dir, err := openDirectory(name); err == nil {
			defer unix.Close(dir)
			tell(ChangeDirSync, name, 0, nil)
			if fullFsync(dir, false) != nil {
				return IOErrDirFsync
			}
		}
	}
	return OK
}

// Access tells whether name exists (as os_unix, an empty regular file does
// not), or can be read and written, or read.
func Access(name string, flags int) (bool, int) {
	switch flags {
	case AccessExists:
		var st unix.Stat_t
		if unix.Stat(name, &st) != nil {
			return false, OK
		}
		return st.Mode&unix.S_IFMT != unix.S_IFREG || st.Size > 0, OK
	case AccessReadWrite:
		return unix.Access(name, unix.R_OK|unix.W_OK) == nil, OK
	default:
		return unix.Access(name, unix.R_OK) == nil, OK
	}
}

// FullPathname is os_unix's unixFullPathname: the absolute path of name,
// with ".", ".." and symbolic links resolved one part at a time. The result
// code is OKSymlink when a link was followed: give it to SQLite as it is.
// A path that does not fit in MaxPathname bytes is SQLITE_CANTOPEN.
func FullPathname(name string) (string, int) {
	p := fullPath{out: make([]byte, 0, MaxPathname)}
	if !strings.HasPrefix(name, "/") {
		wd, err := unix.Getwd()
		if err != nil {
			return "", CantOpen
		}
		p.appendAll(wd)
	}
	p.appendAll(name)
	if p.rc != OK || len(p.out) < 2 {
		return "", CantOpen
	}
	if p.nSymlink > 0 {
		return string(p.out), OKSymlink
	}
	return string(p.out), OK
}

// fullPath is os_unix's DbPath, a path being built.
type fullPath struct {
	out      []byte
	rc       int
	nSymlink int
}

func (p *fullPath) appendAll(path string) {
	for path != "" {
		elem, rest, _ := strings.Cut(path, "/")
		if elem != "" {
			p.appendOne(elem)
		}
		path = rest
	}
}

func (p *fullPath) appendOne(elem string) {
	switch elem {
	case ".":
		return
	case "..":
		if len(p.out) > 1 {
			p.out = p.out[:strings.LastIndexByte(string(p.out), '/')]
		}
		return
	}
	if len(p.out)+len(elem)+2 >= MaxPathname {
		p.rc = Error
		return
	}
	p.out = append(p.out, '/')
	p.out = append(p.out, elem...)
	if p.rc != OK {
		return
	}

	path := string(p.out)
	var st unix.Stat_t
	if err := unix.Lstat(path, &st); err != nil {
		if err != unix.ENOENT {
			p.rc = CantOpen
		}
		return
	}
	if st.Mode&unix.S_IFMT != unix.S_IFLNK {
		return
	}
	if p.nSymlink > maxSymlinks {
		p.rc = CantOpen
		return
	}
	p.nSymlink++
	var buf [4096 + 2]byte
	n, err := unix.Readlink(path, buf[:])
	if err != nil || n <= 0 || n >= len(buf)-2 {
		p.rc = CantOpen
		return
	}
	if buf[0] == '/' {
		p.out = p.out[:0]
	} else {
		p.out = p.out[:len(p.out)-len(elem)-1]
	}
	p.appendAll(string(buf[:n]))
}
