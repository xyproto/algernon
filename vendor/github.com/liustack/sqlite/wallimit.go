// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64))

package sqlite

import (
	"time"

	"github.com/liustack/sqlite/internal/csqlite"
	"github.com/liustack/sqlite/internal/replhook"
)

// SQLite starts the WAL over, from its first frame, when a writer finds
// every frame copied into the database and no reader on the WAL. While
// readers overlap there is no such moment: its checkpoints copy the frames,
// but the file grows as long as that lasts, and reads slow down with it.
//
// So a connection that gives the write lock back looks at the length of the
// WAL, and past the limit of its DSN (_wal_limit) it runs a RESTART
// checkpoint. That one waits: for the readers that hold frames back, then
// for those that still read the WAL, while new readers read the database
// file alone. It waits up to walWait, with the write lock held, so writers
// wait with it. When the readers do not end in time, as with a query no one
// closes, the next try comes once the WAL has grown by another quarter of
// the limit: the wait is paid once in that much writing, not at each commit.
//
// A file that is replicated is left alone here: its replicator runs the
// checkpoints (replica/internal/capture).

// defaultWALLimit is the length past which the driver starts the WAL over
// when the DSN has no _wal_limit.
const defaultWALLimit = 64 << 20

// walWait is how long the checkpoint that starts the WAL over waits for
// readers.
const walWait = replhook.WALWait

// The size of a frame in the WAL beside its page, and of the largest page.
const (
	walFrameHeader = 24
	maxPageSize    = 65536
)

// walNow is the WAL of the file of c as it is now (csqlite.Conn.WALNow). It
// is not ok for the replicator, which gives the lock back with no
// connection of the driver.
func walNow(c *conn) (frames, salt uint32, ok bool) {
	if c == nil {
		return 0, 0, false
	}
	return c.c.WALNow()
}

// walOver says the WAL, of frames frames now, may be past the limit of c,
// which holds the write lock l and has just committed: restartWAL then
// looks closer. l.mu is held.
func (c *conn) walOver(l *writeLock, frames int64) bool {
	if c.walLimit <= 0 || c.replicated || c.closed || c.bad {
		return false
	}
	size := l.walPage
	if size == 0 {
		// Not read yet: as with the largest page.
		size = maxPageSize
	}
	if frames*(size+walFrameHeader) < c.walLimit {
		// Not yet, or it started over, by this or without it.
		l.walSkip = 0
		return false
	}
	return frames >= l.walSkip
}

// restartWAL starts the WAL over when it is past the limit of c, which holds
// the write lock l and is out of any write transaction. It is called without
// l.mu.
func (c *conn) restartWAL(l *writeLock) {
	// The page size changes only outside WAL mode. It is read at each try,
	// which are few.
	size, err := pragmaInt(c.c, "page_size")
	if err != nil || size <= 0 {
		return
	}
	l.walPage = size
	now, _, ok := c.c.WALNow()
	frames, limit := int64(now), c.walLimit/(size+walFrameHeader)
	if !ok || frames < limit {
		return
	}
	// The wait of the checkpoint is SQLite's busy timeout of the
	// connection, set back to what it was after it.
	was, ok := c.c.BusyTimeoutNow()
	if !ok {
		return
	}
	l.walTries.Add(1)
	if c.c.BusyTimeout(int(walWait/time.Millisecond)) != nil {
		return
	}
	_, _, err = c.c.Checkpoint(csqlite.CheckpointRestart)
	c.c.BusyTimeout(was)
	if err != nil {
		// Readers held it off, or c reads itself (a query not read to its
		// end), or the checkpoint failed as a commit would.
		l.walSkip = frames + max(limit/4, 1)
		return
	}
	l.walRestarts.Add(1)
}
