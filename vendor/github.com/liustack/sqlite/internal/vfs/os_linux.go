// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build linux && (amd64 || arm64)

package vfs

import "golang.org/x/sys/unix"

// Locks on open file descriptions, Linux 3.15 and later.
const (
	fOFDSetlk = unix.F_OFD_SETLK
	fOFDGetlk = unix.F_OFD_GETLK
)

// fullFsync is os_unix's full_fsync as built on Linux (HAVE_FDATASYNC):
// fdatasync for every sync. It writes the file size too when that changed,
// which is all SQLite needs of the inode.
func fullFsync(fd int, full bool) error {
	return retry(func() error { return unix.Fdatasync(fd) })
}

// allocate makes the file at least size bytes long, as posix_fallocate.
func allocate(fd int, from, size int64, blksize int64) error {
	err := retry(func() error { return unix.Fallocate(fd, 0, from, size-from) })
	if err == unix.EOPNOTSUPP {
		return allocateByWrites(fd, from, size, blksize)
	}
	return err
}

// openTempFile opens an unnamed file in dir. It fails on a file system
// without O_TMPFILE, and the caller then makes a named file.
func openTempFile(dir string) (int, error) {
	return openFD(dir, unix.O_RDWR|unix.O_EXCL|unix.O_TMPFILE|unix.O_CLOEXEC, 0600)
}

// corruptFS reports the read errors os_unix turns into SQLITE_IOERR_CORRUPTFS.
func corruptFS(err error) bool {
	switch err {
	case unix.ERANGE, unix.EIO, unix.ENXIO:
		return true
	}
	return false
}
