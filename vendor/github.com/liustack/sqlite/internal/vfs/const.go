// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64))

package vfs

// Result codes, with the values of sqlite3.h.
const (
	OK       = 0
	Error    = 1
	Perm     = 3
	Busy     = 5
	NoMem    = 7
	ReadOnly = 8
	IOErr    = 10
	Full     = 13
	CantOpen = 14

	IOErrRead              = IOErr | 1<<8
	IOErrShortRead         = IOErr | 2<<8
	IOErrWrite             = IOErr | 3<<8
	IOErrFsync             = IOErr | 4<<8
	IOErrDirFsync          = IOErr | 5<<8
	IOErrTruncate          = IOErr | 6<<8
	IOErrFstat             = IOErr | 7<<8
	IOErrUnlock            = IOErr | 8<<8
	IOErrRdlock            = IOErr | 9<<8
	IOErrDelete            = IOErr | 10<<8
	IOErrNoMem             = IOErr | 12<<8
	IOErrAccess            = IOErr | 13<<8
	IOErrCheckReservedLock = IOErr | 14<<8
	IOErrLock              = IOErr | 15<<8
	IOErrClose             = IOErr | 16<<8
	IOErrShmOpen           = IOErr | 18<<8
	IOErrShmSize           = IOErr | 19<<8
	IOErrShmLock           = IOErr | 20<<8
	IOErrShmMap            = IOErr | 21<<8
	IOErrDeleteNoent       = IOErr | 23<<8
	IOErrGetTempPath       = IOErr | 25<<8
	IOErrCorruptFS         = IOErr | 33<<8

	CantOpenIsDir = CantOpen | 2<<8

	ReadOnlyCantInit  = ReadOnly | 5<<8
	ReadOnlyDirectory = ReadOnly | 6<<8

	// OKSymlink is what FullPathname returns when it followed a symbolic
	// link. SQLite turns it into OK, or into SQLITE_CANTOPEN_SYMLINK when the
	// database was opened with SQLITE_OPEN_NOFOLLOW.
	OKSymlink = OK | 2<<8
)

// Open flags (SQLITE_OPEN_*).
const (
	OpenReadOnly      = 0x00000001
	OpenReadWrite     = 0x00000002
	OpenCreate        = 0x00000004
	OpenDeleteOnClose = 0x00000008
	OpenExclusive     = 0x00000010
	OpenAutoProxy     = 0x00000020
	OpenURI           = 0x00000040
	OpenMemory        = 0x00000080
	OpenMainDB        = 0x00000100
	OpenTempDB        = 0x00000200
	OpenTransientDB   = 0x00000400
	OpenMainJournal   = 0x00000800
	OpenTempJournal   = 0x00001000
	OpenSubjournal    = 0x00002000
	OpenSuperJournal  = 0x00004000
	OpenWAL           = 0x00080000
	OpenNoFollow      = 0x01000000

	openTypeMask = 0x0FFF00
)

// Lock levels (SQLITE_LOCK_*).
const (
	LockNone      = 0
	LockShared    = 1
	LockReserved  = 2
	LockPending   = 3
	LockExclusive = 4
)

// Shared memory lock flags (SQLITE_SHM_*).
const (
	ShmUnlock    = 1
	ShmLock      = 2
	ShmShared    = 4
	ShmExclusive = 8
	ShmNLock     = 8
)

// Sync flags (SQLITE_SYNC_*).
const (
	SyncNormal   = 0x00002
	SyncFull     = 0x00003
	SyncDataOnly = 0x00010
)

// Access flags (SQLITE_ACCESS_*).
const (
	AccessExists    = 0
	AccessReadWrite = 1
	AccessRead      = 2
)

// Device characteristics (SQLITE_IOCAP_*).
const (
	IOCapPowersafeOverwrite = 0x00001000
	IOCapSubpageRead        = 0x00008000
)

const (
	// MaxPathname is the size of the buffer FullPathname fills, as
	// os_unix's MAX_PATHNAME. Give it to SQLite as sqlite3_vfs.mxPathname.
	MaxPathname = 512

	// sectorSize is os_unix's SQLITE_DEFAULT_SECTOR_SIZE.
	sectorSize = 4096

	// maxSymlinks is SQLITE_MAX_SYMLINK, the most links FullPathname follows.
	maxSymlinks = 200

	// The byte ranges of the database file locks, as os.h.
	pendingByte  = 0x40000000
	reservedByte = pendingByte + 1
	sharedFirst  = pendingByte + 2
	sharedSize   = 510

	// The byte ranges of the shm locks, as os_unix's UNIX_SHM_BASE and
	// UNIX_SHM_DMS.
	shmBase = (22 + ShmNLock) * 4
	shmDMS  = shmBase + ShmNLock

	// shmRegionSize is the size of one region of the wal-index.
	shmRegionSize = 32 * 1024

	// defaultFileMode is SQLITE_DEFAULT_FILE_PERMISSIONS.
	defaultFileMode = 0644

	tempFilePrefix = "etilqs_"
)
