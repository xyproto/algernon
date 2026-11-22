// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64))

package vfs

// OpenOptions are what the VFS reads from the URI of a database, as
// os_unix and os_win do.
type OpenOptions struct {
	// ModeOf is modeof=: a main database opened through a URI is made with
	// the mode of this file (unix only).
	ModeOf string
	// ReadOnlyShm is readonly_shm=1: the -shm is opened for reading only,
	// and can be used only while another process has it set up.
	ReadOnlyShm bool
}

// Open opens name with SQLITE_OPEN_* flags and no URI options.
func Open(name string, flags int) (f *File, outFlags int, rc int) {
	return OpenWith(name, flags, OpenOptions{})
}
