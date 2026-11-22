// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64))

package vfs

import "sync/atomic"

// ChangeKind is what a Change did.
type ChangeKind int

const (
	// ChangeCreate: the file was opened to be made if it is not there.
	ChangeCreate ChangeKind = iota
	// ChangeWrite: Data was written at Off.
	ChangeWrite
	// ChangeTruncate: the size of the file was set to Off.
	ChangeTruncate
	// ChangeSync: the file is about to be synced. Once the sync returns,
	// every write and truncate of the file told before this is on disk.
	ChangeSync
	// ChangeDirSync: the directory of the file is about to be synced. Once
	// the sync returns, the files made in it and deleted from it are.
	// Never told on Windows, which has no directory to sync.
	ChangeDirSync
	// ChangeDelete: the file was deleted.
	ChangeDelete
)

// Change is one thing the VFS did to a file on disk (SetTrace).
type Change struct {
	Kind ChangeKind
	Path string // the name the file was opened or deleted by
	Off  int64
	Data []byte // a copy, the receiver may keep it
}

// trace is the function SetTrace set, nil when none is.
var trace atomic.Pointer[func(Change)]

// SetTrace has the VFS tell f what it does to the files with a name on
// disk: the files it makes and deletes, and each write, truncate and sync.
// A write, a truncate and a delete are told once done, a sync before it
// starts. So what a file holds after a power loss is what was told before
// its last sync, and any part of what was told after. It is for tests that
// play a power loss. f is called from every goroutine that writes. Not
// told: what a write or a truncate that failed did before it failed, what
// the VFS does after a sync failed (SetFault), temporary files, and the
// shared memory of a WAL, which is rebuilt after a crash. nil turns it off.
func SetTrace(f func(Change)) {
	if f == nil {
		trace.Store(nil)
		return
	}
	trace.Store(&f)
}

// tell gives a change of the file named path to SetTrace's function.
func tell(kind ChangeKind, path string, off int64, data []byte) {
	if path == "" {
		return // a temporary file
	}
	if p := trace.Load(); p != nil {
		(*p)(Change{Kind: kind, Path: path, Off: off, Data: append([]byte(nil), data...)})
	}
}
