// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build darwin && (amd64 || arm64)

package vfs

import "golang.org/x/sys/unix"

// Locks on open file descriptions. Darwin has them since macOS 10.15 but does
// not put them in its public headers. The values are from sys/fcntl.h in
// apple-oss-distributions/xnu.
const (
	fOFDSetlk = 90
	fOFDGetlk = 92
)

// fullFsync is os_unix's full_fsync on Darwin: F_FULLFSYNC for
// SQLITE_SYNC_FULL, and fsync when that is not asked for or fails (a file
// system that does not support it).
func fullFsync(fd int, full bool) error {
	if full {
		if _, err := unix.FcntlInt(uintptr(fd), unix.F_FULLFSYNC, 0); err == nil {
			return nil
		}
	}
	return retry(func() error { return unix.Fsync(fd) })
}

// allocate makes the file at least size bytes long. Darwin has no
// posix_fallocate, so os_unix writes a byte into each block instead.
func allocate(fd int, from, size int64, blksize int64) error {
	return allocateByWrites(fd, from, size, blksize)
}

// openTempFile is O_TMPFILE on Linux. Darwin does not have it.
func openTempFile(dir string) (int, error) {
	return -1, unix.EOPNOTSUPP
}

// corruptFS reports the read errors os_unix turns into SQLITE_IOERR_CORRUPTFS.
func corruptFS(err error) bool {
	switch err {
	case unix.ERANGE, unix.EIO, unix.ENXIO, unix.EDEVERR:
		return true
	}
	return false
}
