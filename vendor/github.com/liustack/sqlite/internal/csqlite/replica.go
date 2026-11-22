// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64))

package csqlite

import (
	"encoding/binary"
	"errors"
	"math"
)

// What the replicator (replica/internal/capture) needs of a connection.

// ErrGuarded is under the error of a PRAGMA refused on a connection to a
// replicated database (SetGuard).
var ErrGuarded = errors.New("sqlite: the replicator runs the checkpoints of this database")

// guardedMsg is the message of that error.
const guardedMsg = "the replicator runs the checkpoints of this database: PRAGMA wal_checkpoint, " +
	"wal_autocheckpoint, journal_mode other than WAL and locking_mode=EXCLUSIVE are refused"

// The bit goClassify adds when it refuses one (engine/os.c).
const guarded = 8

// SetGuard makes the connection refuse the PRAGMAs that would take the
// checkpoints of its main database from the replicator: wal_checkpoint,
// wal_autocheckpoint, journal_mode other than WAL and locking_mode other
// than NORMAL.
func (c *Conn) SetGuard(on bool) {
	w := (*int32)(at(c.classifyWord() + 16))
	*w = 0
	if on {
		*w = 1
	}
}

// guardErr is the error of a statement goClassify refused, or nil.
func (c *Conn) guardErr(r int32) error {
	if r == OK || *(*int32)(at(c.classifyWord()))&guarded == 0 {
		return nil
	}
	return &Error{code: int(r), msg: guardedMsg, err: ErrGuarded}
}

// WALHeader is the header of the WAL a read transaction reads: struct
// WalIndexHdr of SQLite's wal.c, also the header of the -shm file.
type WALHeader struct {
	Change    uint32 // counts the transactions written
	BigEndian bool   // the frame checksums are computed on big-endian words
	PageSize  int
	MaxFrame  uint32    // the last frame of the last transaction the read transaction sees
	Pages     uint32    // the size of the database in pages
	Checksum  [2]uint32 // the checksum of frame MaxFrame
	Salt      [8]byte   // the salts of the WAL, as its header and its frames hold them
}

// WALHeader is the WAL header the open read transaction of the connection
// reads.
func (c *Conn) WALHeader() (WALHeader, error) {
	if err := c.brokenErr(); err != nil {
		return WALHeader{}, err
	}
	a := c.args()
	out := a.space(48)
	if r := int32(go_wal_header(a.top, c.db, out)); r != OK {
		return WALHeader{}, &Error{code: int(r), msg: "no read transaction on a database in WAL mode"}
	}
	b := bytesAt(out, 48)
	ne := binary.NativeEndian
	h := WALHeader{
		Change:    ne.Uint32(b[8:]),
		BigEndian: b[13] != 0,
		PageSize:  int(ne.Uint16(b[14:])),
		MaxFrame:  ne.Uint32(b[16:]),
		Pages:     ne.Uint32(b[20:]),
		Checksum:  [2]uint32{ne.Uint32(b[24:]), ne.Uint32(b[28:])},
	}
	if h.PageSize == 1 {
		h.PageSize = 65536
	}
	copy(h.Salt[:], b[32:40])
	return h, nil
}

// WALNow is the WAL of the main database as it is now: its length in frames,
// and a number that changes each time the WAL starts over. ok is false when
// the connection has not read that WAL yet, has none, or is broken.
//
// It needs no transaction, and it is not what the connection last saw: a
// connection that has not touched the main database for a while, as one
// that writes only its temporary tables, still gets the WAL of now.
func (c *Conn) WALNow() (frames, salt uint32, ok bool) {
	if c.brokenErr() != nil {
		return 0, 0, false
	}
	a := c.args()
	v := uint64(go_wal_now(a.top, c.db))
	if v == math.MaxUint64 {
		return 0, 0, false
	}
	return uint32(v), uint32(v >> 32), true
}

// PageCount is the size of the main database in pages, as the open read
// transaction sees it.
func (c *Conn) PageCount() (uint32, error) {
	if err := c.brokenErr(); err != nil {
		return 0, err
	}
	a := c.args()
	n := a.word()
	if r := int32(go_page_count(a.top, c.db, n)); r != OK {
		return 0, &Error{code: int(r), msg: "no read transaction"}
	}
	return *(*uint32)(at(n)), nil
}

// ReadPage reads page pgno of the main database, as the open read
// transaction sees it, into buf, which is one page long. The lock page
// cannot be read.
func (c *Conn) ReadPage(pgno uint32, buf []byte) error {
	if err := c.brokenErr(); err != nil {
		return err
	}
	if len(buf) > 65536 {
		return &Error{code: 1, msg: "not a page size"}
	}
	a := c.args()
	out := a.space(len(buf))
	if r := int32(go_read_page(a.top, c.db, uintptr(pgno), out, uintptr(len(buf)))); r != OK {
		return c.err(int(r))
	}
	copy(buf, bytesAt(out, len(buf)))
	return nil
}

// The modes of Checkpoint (SQLITE_CHECKPOINT_*).
const (
	CheckpointPassive  = 0
	CheckpointFull     = 1
	CheckpointRestart  = 2
	CheckpointTruncate = 3
)

// Checkpoint checkpoints the main database. log is the number of frames in
// the WAL, ckpt the number of them now in the database file: -1 each when
// the database is not in WAL mode.
func (c *Conn) Checkpoint(mode int) (log, ckpt int, err error) {
	if err := c.brokenErr(); err != nil {
		return 0, 0, err
	}
	a := c.args()
	pLog, pCkpt := a.word(), a.word()
	z := a.str("main")
	r := int32(sqlite3_wal_checkpoint_v2(a.top, c.db, z, uintptr(mode), pLog, pCkpt))
	log, ckpt = int(*(*int32)(at(pLog))), int(*(*int32)(at(pCkpt)))
	return log, ckpt, c.errOf(int(r)) // not c.err: see Spent
}
