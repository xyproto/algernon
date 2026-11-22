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

// A commit in a rollback journal mode syncs the journal before it writes the
// database. When that sync fails, SQLite rolls back in the process by reading
// the journal: each page there goes back into the cache. But after a failed
// sync Linux may drop the pages it could not write (see wal_unix.go), and the
// journal then reads back as the disk holds it, without some of what was
// written:
//
//   - Without the header or the pages, SQLite finds nothing to restore, takes
//     the rollback as done, and keeps the pages of the failed transaction in
//     the cache, where the next commit writes them to the database.
//   - A journal that persists keeps the journal of an earlier transaction
//     past the end of this one, and SQLite zeroes the first byte of the old
//     header there. Without that byte, a rollback goes on into the old
//     journal and undoes pages of a transaction that was committed.
//
// So the first failed sync of a rollback journal cuts it at the first byte
// written since its last good sync, and stops its File: each read, write,
// truncate and sync of it fails from then on. The rollback in the process
// fails, and SQLite drops its whole cache, as after any I/O error it cannot
// undo. Once SQLite closed the File, what is left of the journal is rolled
// back as after a crash. That is all that must be undone: SQLite writes a
// page to the database only after a good sync of the journal that holds it,
// so the part cut covers no page of the database, and after the last sync of
// a commit, which only ends the journal, the database is already synced.
//
// The cut is a truncate, which holds in the page cache though it may not
// reach the disk either. A power loss before the journal is rolled back may
// then bring back what was cut, as it may any write that was not synced.
//
// When the cut itself fails, the journal does not open again in the process
// until the cut is made: each open tries it again. A journal that another
// process wrote or replaced since is let go as it is, since it is no longer
// the journal the sync failed on.
//
// SQLite closes the journal when it lets go of the lock on the database.
// With locking_mode=EXCLUSIVE it never does, so the driver drops a connection
// whose journal is stopped: see JournalStopped.

// journals are the stopped journals of the process, by the name of the
// journal: how many stopped Files are open, and the cut that failed.
var journals struct {
	sync.Mutex
	open  map[string]int
	uncut map[string]*uncut
}

// uncut is a journal whose cut failed.
type uncut struct {
	off int64 // where to cut
	// held is a descriptor of the journal. It makes the cut, and keeps the
	// inode from being given to a new file, which would be taken for this
	// one. -1 when the descriptor could not be made.
	held int
	// st is the journal when the cut failed: a journal written since is
	// another process's.
	st unix.Stat_t
}

// same says a and b are one file that was not written between.
func same(a, b *unix.Stat_t) bool {
	return a.Dev == b.Dev && a.Ino == b.Ino && a.Size == b.Size && a.Mtim == b.Mtim && a.Ctim == b.Ctim
}

// stopJournal stops f, a rollback journal, after its sync failed, and cuts
// it at the first byte written since its last good sync.
func (f *File) stopJournal() {
	f.stopped = true
	journals.Lock()
	defer journals.Unlock()
	if journals.open == nil {
		journals.open = make(map[string]int)
		journals.uncut = make(map[string]*uncut)
	}
	journals.open[f.path]++
	stoppedJournals.Add(1)
	if f.dirty == clean || f.cut(f.fd, f.dirty) {
		return
	}
	u := &uncut{off: f.dirty, held: -1}
	if fd, err := unix.FcntlInt(uintptr(f.fd), unix.F_DUPFD_CLOEXEC, 3); err == nil {
		u.held = fd
	}
	// Without the state of the journal, no later one is the same, and the
	// journal is let go at its next open.
	unix.Fstat(f.fd, &u.st)
	if old := journals.uncut[f.path]; old != nil && old.held >= 0 {
		unix.Close(old.held)
	}
	journals.uncut[f.path] = u
}

// cut truncates the journal of the descriptor fd at off. As the sync that
// failed, the cut may not reach the disk.
func (f *File) cut(fd int, off int64) bool {
	if f.faultOf(OpTruncate) != OK {
		return false
	}
	if retry(func() error { return unix.Ftruncate(fd, off) }) != nil {
		return false
	}
	fullFsync(fd, true)
	return true
}

// openJournal makes the cut of f, a rollback journal just opened, when the
// cut failed as a sync of the journal failed. The journal does not open
// until it is made.
func (f *File) openJournal() int {
	journals.Lock()
	defer journals.Unlock()
	u := journals.uncut[f.path]
	if u == nil {
		return OK
	}
	if err := unix.Fstat(f.fd, &f.st); err != nil {
		return IOErrFstat
	}
	if same(&f.st, &u.st) && (u.held < 0 || !f.cut(u.held, u.off)) {
		return IOErr
	}
	if u.held >= 0 {
		unix.Close(u.held)
	}
	delete(journals.uncut, f.path)
	return OK
}

// closeJournal forgets f, a stopped journal that is closing.
func (f *File) closeJournal() {
	f.stopped = false
	journals.Lock()
	if journals.open[f.path]--; journals.open[f.path] == 0 {
		delete(journals.open, f.path)
	}
	journals.Unlock()
	stoppedJournals.Add(-1)
}

// JournalStopped says a failed sync stopped the rollback journal of the
// database name, the name SQLite opened it by, and a File still has it
// open: a connection that holds it cannot use the database until it closes.
func JournalStopped(name string) bool {
	if !JournalsStopped() {
		return false
	}
	journals.Lock()
	defer journals.Unlock()
	return journals.open[name+"-journal"] > 0
}
