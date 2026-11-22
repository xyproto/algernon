// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64))

package sqlite

import "github.com/liustack/sqlite/internal/csqlite"

// SQLite's automatic checkpoint runs after every commit that leaves the WAL
// at wal_autocheckpoint frames or more. With no reader that is once each so
// many frames: the checkpoint copies them all, and the next writer starts
// the WAL over. While reads overlap the WAL does not start over, so every
// commit runs a checkpoint, which syncs the WAL for the few frames the
// readers let it copy. Where a sync is slow the writer spends most of its
// time there.
//
// So the driver runs the automatic checkpoints of its connections itself:
// the connection that gives the write lock back runs one once the WAL has
// grown by its wal_autocheckpoint frames since the last one the driver ran
// (release in writer.go). It takes them from SQLite (takeCheckpoints), and
// PRAGMA wal_autocheckpoint reads and sets the number of frames as before.
//
// Not for a replicated file, whose replicator runs the checkpoints: its
// connections have wal_autocheckpoint at 0 and cannot set it. And not for a
// connection with an attached database: SQLite's own cover every database
// of a connection, the driver's only the main one, so that connection has
// SQLite's until it detaches.

// takeCheckpoints sets who runs the automatic checkpoints of c, and reads
// what wal_autocheckpoint the program gave it. It is called once c is set
// up, and after each statement that changes the connection: a PRAGMA, an
// ATTACH or a DETACH. It reads and sets fields of the connection, and
// cannot fail.
func (c *conn) takeCheckpoints() {
	if c.wl == nil {
		return
	}
	if c.c.Attached() {
		c.c.GiveAutoCheckpoint()
		c.walSQLite = true
		return
	}
	// SQLite has them at first, after a PRAGMA wal_autocheckpoint=N, which
	// sets its own, and after a DETACH.
	if n, ok := c.c.TakeAutoCheckpoint(); ok {
		c.walEvery = int64(n)
	}
	c.walSQLite = false
}

// walDue says an automatic checkpoint is due: the WAL, of frames frames
// now, has grown by the wal_autocheckpoint of c since the last one that
// ran. salt changes when the WAL starts over (csqlite.Conn.WALNow). c holds
// the write lock l and has just committed. l.mu is held.
func (c *conn) walDue(l *writeLock, frames int64, salt uint32) bool {
	if c.walEvery <= 0 || c.walSQLite || c.closed || c.bad {
		return false
	}
	if salt != l.walSalt || frames < l.walAt {
		// It started over: a checkpoint had copied all of it. The salt
		// tells, and so does a WAL shorter than it was, should a WAL made
		// anew have the salt of the one before.
		l.walSalt, l.walAt = salt, 0
	}
	return frames-l.walAt >= c.walEvery
}

// checkpoint runs the automatic checkpoint of c, which holds the write lock
// l and is out of any write transaction, on a WAL of frames frames. It is
// called without l.mu. As SQLite's own, it copies what the readers let it
// and waits for no one. One that could not run, as when c has a query open
// or another checkpoint runs, does not count: the next commit tries again,
// as SQLite does at each commit past wal_autocheckpoint.
func (c *conn) checkpoint(l *writeLock, frames int64) {
	if _, _, err := c.c.Checkpoint(csqlite.CheckpointPassive); err != nil {
		return
	}
	l.walAt = frames
	l.walCheckpoints.Add(1)
}
