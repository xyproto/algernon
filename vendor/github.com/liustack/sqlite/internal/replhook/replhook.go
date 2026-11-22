// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

// Package replhook gives the replicator (replica/internal/capture) what it
// needs of the driver and the driver does not offer its users. Package
// sqlite fills it in as it is initialized.
package replhook

import (
	"context"
	"time"
)

// WALWait is how long a checkpoint that starts the WAL over, once it is
// past the limit of a connection, waits for the readers: by the driver for
// a file that is not replicated, by the replicator for one that is.
const WALWait = 100 * time.Millisecond

// Attach makes every connection the driver opens to the database file path
// from then on ready for replication: no automatic checkpoints, and the
// PRAGMAs that would checkpoint refused (csqlite.Conn.SetGuard). The file
// must exist. It is found by what it is, so a connection that names it
// another way (case, a short windows name) is covered too. It fails when the
// driver has a connection to the file open, or the file is attached already.
// detach ends it: connections opened after it are as before.
//
// committed, if not nil, is called each time a connection of the driver that
// held the file's write lock ends its transaction, before the lock goes to
// the next writer, with the frames in the WAL as it is then and a number
// that changes each time the WAL starts over (csqlite.Conn.WALNow). A
// connection that never read the WAL does not call it: it wrote nothing
// there. It runs on the writer's goroutine and holds the lock meanwhile: the
// writers of the file wait for it, and it must not take the lock (Lock).
// limit is the length of the WAL in bytes past which that connection wants
// it started over, 0 for never (its _wal_limit): on an attached file the
// driver leaves that to committed.
var Attach func(path string, committed func(frames, salt uint32, limit int64)) (detach func(), err error)

// Lock takes the driver's write lock of an attached file, as a writer does,
// waiting until ctx is done: until unlock, no connection of this process
// writes to the file.
var Lock func(ctx context.Context, path string) (unlock func(), err error)

// TryLock takes the driver's write lock of an attached file only if no one
// holds it: ok is false when a writer does.
var TryLock func(path string) (unlock func(), ok bool, err error)
