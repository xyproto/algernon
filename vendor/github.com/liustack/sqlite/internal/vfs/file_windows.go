// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build windows && (amd64 || arm64)

// The files and locks of SQLite on Windows, as os_win.c. Windows locks
// (LockFileEx) belong to the handle they are taken on, and a lock stops
// every other handle, of this process or another. So each File holds its
// own locks on its own handle, as each connection of os_win does, and two
// Files of one database in this process exclude each other the way two
// processes do. The byte ranges and the order they are taken in are those
// of os_win, so SQLite built from C can use the same database at the same
// time.
//
// Results are SQLite result codes, with the values of sqlite3.h. One File
// is used by one goroutine at a time.

package vfs

import (
	"math/rand/v2"
	"os"
	"strconv"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// File is an open file, as os_win's winFile.
type File struct {
	h        windows.Handle
	path     string // the name given to Open, "" for a temporary file
	lock     int    // the lock level held
	noLock   bool   // not a main database: Lock and Unlock do nothing, as os_unix
	readOnly bool   // opened for reading only
	roShm    bool   // readonly_shm=1
	chunk    int64  // SQLITE_FCNTL_CHUNK_SIZE
	shm      *shm

	// Space for the arguments of the calls made for every page and every
	// lock, so that they do not allocate.
	ov   windows.Overlapped
	n    uint32
	info windows.ByHandleFileInformation
}

// OpenWith opens name with SQLITE_OPEN_* flags and the options of its URI,
// as os_win's winOpen: of the options, only ReadOnlyShm. An empty name is a
// temporary file, deleted when closed. outFlags are the flags the file was
// opened with: READONLY in place of READWRITE and CREATE when it could only
// be opened for reading.
func OpenWith(name string, flags int, o OpenOptions) (f *File, outFlags int, rc int) {
	eType := flags & openTypeMask
	isExclusive := flags&OpenExclusive != 0
	isDelete := flags&OpenDeleteOnClose != 0
	isCreate := flags&OpenCreate != 0
	isReadWrite := flags&OpenReadWrite != 0

	fileName := name
	if name == "" {
		if fileName, rc = tempName(); rc != OK {
			return nil, 0, rc
		}
	}
	p, err := windows.UTF16PtrFromString(fileName)
	if err != nil {
		return nil, 0, CantOpen
	}
	if isDir(p) {
		return nil, 0, CantOpenIsDir
	}

	access := uint32(windows.GENERIC_READ)
	if isReadWrite {
		access |= windows.GENERIC_WRITE
	}
	var disposition uint32
	switch {
	case isExclusive:
		disposition = windows.CREATE_NEW
	case isCreate:
		disposition = windows.OPEN_ALWAYS
	default:
		disposition = windows.OPEN_EXISTING
	}
	attrs := uint32(windows.FILE_ATTRIBUTE_NORMAL)
	if isDelete {
		attrs = windows.FILE_ATTRIBUTE_TEMPORARY | windows.FILE_ATTRIBUTE_HIDDEN | windows.FILE_FLAG_DELETE_ON_CLOSE
	}

	// As os_win: no FILE_SHARE_DELETE, so that a file in use cannot be
	// deleted or renamed. A file that is there but cannot be opened for
	// writing is opened for reading, and a file that is busy (a virus
	// scanner, an indexer) is tried again.
	var h windows.Handle
	isRO := false
	for cnt := 0; ; {
		h, err = windows.CreateFile(p, access, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, disposition, attrs, 0)
		if err == nil {
			break
		}
		if isReadWrite {
			if _, aerr := windows.GetFileAttributes(p); aerr == nil {
				isRO = true
				break
			}
		}
		if !retryIO(&cnt, err) {
			break
		}
	}
	if err != nil {
		if isReadWrite && isRO && !isExclusive {
			return OpenWith(name, flags&^(OpenReadWrite|OpenCreate)|OpenReadOnly, o)
		}
		return nil, 0, CantOpen
	}
	if isCreate && !isDelete {
		tell(ChangeCreate, name, 0, nil)
	}
	f = &File{
		h:        h,
		path:     name,
		noLock:   eType != OpenMainDB,
		readOnly: flags&OpenReadOnly != 0,
		roShm:    o.ReadOnlyShm,
	}
	return f, flags, OK
}

// isDir tells whether p names a directory.
func isDir(p *uint16) bool {
	a, err := windows.GetFileAttributes(p)
	return err == nil && a&windows.FILE_ATTRIBUTE_DIRECTORY != 0
}

// tempName is os_win's winGetTempname: a random name in the directory for
// temporary files.
func tempName() (string, int) {
	dir := os.TempDir()
	if dir == "" {
		return "", IOErrGetTempPath
	}
	if dir[len(dir)-1] != '\\' && dir[len(dir)-1] != '/' {
		dir += `\`
	}
	return dir + tempFilePrefix + strconv.FormatUint(rand.Uint64(), 16), OK
}

// The errors os_win tries again (winIoerrCanRetry1): what a virus scanner or
// an indexer that has the file open for a moment causes, and network
// hiccups.
func retryable(err error) bool {
	switch err {
	case windows.ERROR_ACCESS_DENIED, windows.ERROR_SHARING_VIOLATION,
		windows.ERROR_LOCK_VIOLATION, windows.ERROR_DEV_NOT_EXIST,
		windows.ERROR_NETNAME_DELETED, windows.ERROR_SEM_TIMEOUT,
		windows.ERROR_NETWORK_UNREACHABLE:
		return true
	}
	return false
}

const (
	ioRetries    = 10                    // SQLITE_WIN32_IOERR_RETRY
	ioRetryDelay = 25 * time.Millisecond // SQLITE_WIN32_IOERR_RETRY_DELAY
)

// retryIO is os_win's winRetryIoerr: after an error err that may pass,
// wait a little longer each time and say to try again, up to ioRetries
// times.
func retryIO(cnt *int, err error) bool {
	if *cnt >= ioRetries || !retryable(err) {
		return false
	}
	*cnt++
	time.Sleep(ioRetryDelay * time.Duration(*cnt))
	return true
}

// Close closes the file, and its shared memory if ShmUnmap was not called.
// The locks the File holds are released first: Windows drops the locks of
// a closed handle only when it gets to it. Closing twice is harmless.
func (f *File) Close() int {
	if f.shm != nil {
		f.ShmUnmap(false)
	}
	if f.h == windows.InvalidHandle {
		return OK
	}
	f.Unlock(LockNone)
	// As os_win, three tries a tenth of a second apart.
	var err error
	for range 3 {
		if err = windows.CloseHandle(f.h); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	f.h = windows.InvalidHandle
	if err != nil {
		return IOErrClose
	}
	return OK
}

// at sets the offset of the next ReadFile, WriteFile or LockFileEx.
func (f *File) at(off int64) *windows.Overlapped {
	f.ov = windows.Overlapped{Offset: uint32(off), OffsetHigh: uint32(off >> 32)}
	return &f.ov
}

// ReadAt reads len(p) bytes at off. When the file ends first, the rest of p
// is zeroed and the result is SQLITE_IOERR_SHORT_READ.
func (f *File) ReadAt(p []byte, off int64) int {
	got, cnt := 0, 0
	for got < len(p) {
		err := windows.ReadFile(f.h, p[got:], &f.n, f.at(off+int64(got)))
		if err == windows.ERROR_HANDLE_EOF {
			f.n, err = 0, nil
		}
		if err != nil {
			if retryIO(&cnt, err) {
				continue
			}
			return IOErrRead
		}
		if f.n == 0 {
			clear(p[got:])
			return IOErrShortRead
		}
		got += int(f.n)
	}
	return OK
}

// WriteAt writes p at off. A full disk is SQLITE_FULL.
func (f *File) WriteAt(p []byte, off int64) int {
	if rc := f.faultOf(OpWrite); rc != OK {
		return rc
	}
	cnt := 0
	all, at := p, off
	for len(p) > 0 {
		err := windows.WriteFile(f.h, p, &f.n, f.at(off))
		if err != nil {
			if retryIO(&cnt, err) {
				continue
			}
			if err == windows.ERROR_DISK_FULL || err == windows.ERROR_HANDLE_DISK_FULL {
				return Full
			}
			return IOErrWrite
		}
		if f.n == 0 || int(f.n) > len(p) {
			return IOErrWrite
		}
		p = p[f.n:]
		off += int64(f.n)
	}
	tell(ChangeWrite, f.path, at, all)
	return OK
}

// Truncate sets the size of the file, rounded up to whole chunks when
// SetChunkSize was called.
func (f *File) Truncate(size int64) int {
	if rc := f.faultOf(OpTruncate); rc != OK {
		return rc
	}
	if f.chunk > 0 {
		size = (size + f.chunk - 1) / f.chunk * f.chunk
	}
	err := windows.Ftruncate(f.h, size)
	// As os_win: another program (SQLite with mmap_size set) has the file
	// mapped, and Windows does not shrink a mapped file. It stays longer,
	// which SQLite reads fine.
	if err != nil && err != windows.ERROR_USER_MAPPED_FILE {
		return IOErrTruncate
	}
	if err == nil {
		tell(ChangeTruncate, f.path, size, nil)
	}
	return OK
}

// Sync writes the file to disk. Windows has one kind of sync, and no
// directory to sync.
func (f *File) Sync(flags int) int {
	if rc := f.faultOf(OpSync); rc != OK {
		return rc
	}
	tell(ChangeSync, f.path, 0, nil)
	if err := windows.FlushFileBuffers(f.h); err != nil {
		return IOErrFsync
	}
	return OK
}

// Size is the size of the file.
func (f *File) Size() (int64, int) {
	if err := windows.GetFileInformationByHandle(f.h, &f.info); err != nil {
		return 0, IOErrFstat
	}
	return int64(f.info.FileSizeHigh)<<32 | int64(f.info.FileSizeLow), OK
}

// SetChunkSize is SQLITE_FCNTL_CHUNK_SIZE: Truncate and SizeHint round the
// size up to whole chunks of n bytes. 0 turns it off.
func (f *File) SetChunkSize(n int) {
	f.chunk = int64(n)
}

// SizeHint is SQLITE_FCNTL_SIZE_HINT, as os_win: with a chunk size set, make
// the file at least size bytes, rounded up to whole chunks.
func (f *File) SizeHint(size int64) int {
	if f.chunk <= 0 {
		return OK
	}
	cur, rc := f.Size()
	if rc != OK {
		return rc
	}
	if size > cur {
		return f.Truncate(size)
	}
	return OK
}

// HasMoved is SQLITE_FCNTL_HAS_MOVED. A file opened without
// FILE_SHARE_DELETE cannot be renamed or deleted, so it never has.
func (f *File) HasMoved() bool {
	return false
}

// SectorSize is os_win's SQLITE_DEFAULT_SECTOR_SIZE.
func (f *File) SectorSize() int {
	return sectorSize
}

// DeviceCharacteristics gives the SQLITE_IOCAP_* bits of the file as os_win
// does, but powersafe overwrite: engine/os.c adds that one, which the psow
// URI parameter turns off. Subpage read lets SQLite read a large value from
// its overflow pages straight into its buffer.
//
// os_win also says an open file cannot be deleted
// (SQLITE_IOCAP_UNDELETABLE_WHEN_OPEN). SQLite would then keep a truncated
// or kept journal open between transactions, and a connection with the
// journal mode delete, as the sqlite3 shell has by default, would fail to
// commit since it cannot delete that journal. Without the bit SQLite closes
// the journal when it unlocks the database, as on unix.
func (f *File) DeviceCharacteristics() int {
	return IOCapSubpageRead
}

// lockFile takes a lock on n bytes from off. It fails at once when another
// handle holds a lock in the way.
func (f *File) lockFile(flags uint32, off int64, n uint32) error {
	return windows.LockFileEx(f.h, flags|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, n, 0, f.at(off))
}

func (f *File) unlockFile(off int64, n uint32) error {
	return windows.UnlockFileEx(f.h, 0, n, 0, f.at(off))
}

// The locks of os_win: the PENDING byte, the RESERVED byte, and the shared
// range, read locked by each reader and write locked by the writer.
func (f *File) getReadLock() error {
	return f.lockFile(0, sharedFirst, sharedSize)
}

func (f *File) unlockReadLock() {
	f.unlockFile(sharedFirst, sharedSize)
}

// Lock raises the lock to level, as os_win's winLock, with the same
// intermediate states: SHARED -> RESERVED, RESERVED -> (PENDING) ->
// EXCLUSIVE, PENDING -> EXCLUSIVE, and SHARED -> EXCLUSIVE for a hot
// journal. A level that cannot be had now is SQLITE_BUSY, and the lock is
// then left at the highest state reached.
func (f *File) Lock(level int) int {
	if f.lock >= level {
		return OK
	}
	if f.noLock {
		f.lock = level
		return OK
	}
	if f.readOnly && level >= LockReserved {
		return IOErrLock
	}
	newLock := f.lock
	ok := true
	var err error

	// The PENDING byte, for good with EXCLUSIVE, for a moment with SHARED:
	// a writer that holds it keeps new readers out. As os_win, three tries
	// a millisecond apart, for virus scanners.
	gotPending := false
	if level == LockShared || level == LockExclusive && f.lock == LockReserved {
		flags := uint32(0)
		if level == LockExclusive {
			flags = windows.LOCKFILE_EXCLUSIVE_LOCK
		}
		for cnt := 3; cnt > 0; cnt-- {
			if err = f.lockFile(flags, pendingByte, 1); err == nil {
				break
			}
			if err == windows.ERROR_INVALID_HANDLE {
				return IOErrLock
			}
			if cnt > 1 {
				time.Sleep(time.Millisecond)
			}
		}
		ok = err == nil
		gotPending = ok
	}

	switch {
	case level == LockShared && ok:
		if ok = f.getReadLock() == nil; ok {
			newLock = LockShared
		}
	case level == LockReserved && ok:
		if ok = f.lockFile(windows.LOCKFILE_EXCLUSIVE_LOCK, reservedByte, 1) == nil; ok {
			newLock = LockReserved
		}
	case level == LockExclusive && ok:
		newLock = LockPending
		gotPending = false
		f.unlockReadLock()
		if ok = f.lockFile(windows.LOCKFILE_EXCLUSIVE_LOCK, sharedFirst, sharedSize) == nil; ok {
			newLock = LockExclusive
		} else {
			f.getReadLock()
		}
	}

	if gotPending && level == LockShared {
		f.unlockFile(pendingByte, 1)
	}
	f.lock = newLock
	if !ok {
		return Busy
	}
	return OK
}

// Unlock lowers the lock to level, LockShared or LockNone, as os_win's
// winUnlock.
func (f *File) Unlock(level int) int {
	if f.lock <= level {
		return OK
	}
	old := f.lock
	f.lock = level
	if f.noLock {
		return OK
	}
	rc := OK
	if old >= LockExclusive {
		f.unlockFile(sharedFirst, sharedSize)
		if level == LockShared && f.getReadLock() != nil {
			rc = IOErrUnlock
		}
	}
	if old >= LockReserved {
		f.unlockFile(reservedByte, 1)
	}
	if level == LockNone && old >= LockShared {
		f.unlockReadLock()
	}
	if old >= LockPending {
		f.unlockFile(pendingByte, 1)
	}
	return rc
}

// CheckReservedLock tells whether any File or process holds RESERVED or
// more, as os_win: by trying to lock the RESERVED byte.
func (f *File) CheckReservedLock() (bool, int) {
	if f.noLock {
		return false, OK
	}
	if f.lock >= LockReserved {
		return true, OK
	}
	if f.lockFile(0, reservedByte, 1) != nil {
		return true, OK
	}
	f.unlockFile(reservedByte, 1)
	return false, OK
}

// LockState is SQLITE_FCNTL_LOCKSTATE, the lock level held.
func (f *File) LockState() int {
	return f.lock
}

// Delete deletes name, as os_win's winDelete, trying again while another
// program has it open for a moment. syncDir does nothing: Windows has no
// directory to sync.
func Delete(name string, syncDir bool) int {
	rc := deleteFile(name)
	if rc == OK {
		tell(ChangeDelete, name, 0, nil)
	}
	return rc
}

// deleteFile is Delete without telling SetTrace, which is not told of the
// shared memory of a WAL.
func deleteFile(name string) int {
	p, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return IOErrDelete
	}
	for cnt := 0; ; {
		a, err := windows.GetFileAttributes(p)
		if err != nil {
			if err == windows.ERROR_FILE_NOT_FOUND || err == windows.ERROR_PATH_NOT_FOUND {
				return IOErrDeleteNoent
			}
			return IOErrDelete
		}
		if a&windows.FILE_ATTRIBUTE_DIRECTORY != 0 {
			return IOErrDelete
		}
		err = windows.DeleteFile(p)
		if err == nil {
			return OK
		}
		if !retryIO(&cnt, err) {
			return IOErrDelete
		}
	}
}

// Access tells whether name exists (as os_win, an empty file does not), or
// can be read and written, or read, as os_win's winAccess.
func Access(name string, flags int) (bool, int) {
	p, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return false, OK
	}
	var data windows.Win32FileAttributeData
	for cnt := 0; ; {
		err = windows.GetFileAttributesEx(p, windows.GetFileExInfoStandard, (*byte)(unsafe.Pointer(&data)))
		if err == nil || !retryIO(&cnt, err) {
			break
		}
	}
	if err != nil {
		if err == windows.ERROR_FILE_NOT_FOUND || err == windows.ERROR_PATH_NOT_FOUND {
			return false, OK
		}
		return false, IOErrAccess
	}
	switch flags {
	case AccessExists:
		return data.FileSizeHigh != 0 || data.FileSizeLow != 0, OK
	case AccessReadWrite:
		return data.FileAttributes&windows.FILE_ATTRIBUTE_READONLY == 0, OK
	default:
		return true, OK
	}
}

// FullPathname is os_win's winFullPathname: the absolute path of name, from
// GetFullPathNameW. As os_win, a leading "/" before a drive letter or a
// "\\?\" prefix is dropped: a URI gives "file:///C:/x.db" as "/C:/x.db". A
// path that does not fit in MaxPathname bytes is SQLITE_CANTOPEN.
func FullPathname(name string) (string, int) {
	if len(name) >= 3 && name[0] == '/' &&
		(isDriveLetter(name[1]) && name[2] == ':' || len(name) >= 5 && name[1:5] == `\\?\`) {
		name = name[1:]
	}
	p, err := windows.UTF16FromString(name)
	if err != nil {
		return "", CantOpen
	}
	buf := make([]uint16, MaxPathname)
	n, err := windows.GetFullPathName(&p[0], uint32(len(buf)), &buf[0], nil)
	if err != nil || n == 0 || int(n) >= len(buf) {
		return "", CantOpen
	}
	out := windows.UTF16ToString(buf[:n])
	if len(out)+1 > MaxPathname {
		return "", CantOpen
	}
	return out, OK
}

func isDriveLetter(c byte) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}
