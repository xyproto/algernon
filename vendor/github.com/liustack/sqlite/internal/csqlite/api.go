// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64))

package csqlite

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"unsafe"

	// A Go newer than the one the engine was checked with stops the build.
	_ "github.com/liustack/sqlite/internal/goversion"
)

// The C API of SQLite for the driver. A Conn and its statements are used by
// one goroutine at a time. Strings and out-parameters of a call go on the
// connection's shadow stack, below the shadow pointer the call gets, so a
// call allocates no C memory.

// Result codes and flags of sqlite3.h.
const (
	OK   = 0
	Row  = 100
	Done = 101

	OpenReadOnly     = 0x00000001
	OpenReadWrite    = 0x00000002
	OpenCreate       = 0x00000004
	OpenURI          = 0x00000040
	OpenMemory       = 0x00000080
	OpenNoMutex      = 0x00008000
	OpenFullMutex    = 0x00010000
	OpenSharedCache  = 0x00020000
	OpenPrivateCache = 0x00040000
	OpenExResCode    = 0x02000000

	TypeInteger = 1
	TypeFloat   = 2
	TypeText    = 3
	TypeBlob    = 4
	TypeNull    = 5
)

// Error is a result code of SQLite with its message.
type Error struct {
	code int // the extended result code
	msg  string
	err  error // what is under it, or nil
}

// NewError is an error of the extended result code code with the message
// msg, and cause under it (nil for none).
func NewError(code int, msg string, cause error) *Error {
	return &Error{code: code, msg: msg, err: cause}
}

func (e *Error) Error() string { return fmt.Sprintf("sqlite: %s (%d)", e.msg, e.code) }

// Unwrap is what is under the error: the error that made it, or nil.
func (e *Error) Unwrap() error { return e.err }

// Code is the primary result code, as SQLITE_CONSTRAINT.
func (e *Error) Code() int { return e.code & 0xff }

// ExtendedCode is the extended result code, as SQLITE_CONSTRAINT_UNIQUE. For
// an error that has no extended code it is the primary code.
func (e *Error) ExtendedCode() int { return e.code }

// ErrCollationPanic is under the error of a statement in which a Go collation
// panicked.
var ErrCollationPanic = errors.New("panic in a Go collation")

var initOnce sync.Once
var initErr error

func start() error {
	initOnce.Do(func() {
		s, err := newStack()
		if err != nil {
			initErr = err
			return
		}
		defer s.free()
		if r := int32(go_init(s.sp())); r != OK {
			initErr = &Error{code: int(r), msg: errstr(s.sp(), int(r))}
		}
	})
	return initErr
}

func errstr(sp uintptr, code int) string { return cstring(sqlite3_errstr(sp, uintptr(code))) }

// LibVersion is the version of the SQLite in the engine.
func LibVersion() (string, error) {
	if err := start(); err != nil {
		return "", err
	}
	s, err := newStack()
	if err != nil {
		return "", err
	}
	defer s.free()
	return cstring(sqlite3_libversion(s.sp())), nil
}

// Conn is a connection to a database.
type Conn struct {
	st *stack
	sp uintptr
	db uintptr
	// changed says a statement prepared on the connection changes it
	// (ChangesConn). SQLite applies some of them as it prepares them, such as
	// a PRAGMA that sets a flag, so this holds whatever became of the
	// statement.
	changed bool
	// justChanged says the last Prepare was of such a statement, whether it
	// failed or not.
	justChanged bool
	// spent says a call on the connection failed with SQLITE_NOMEM,
	// SQLITE_IOERR or SQLITE_FULL (Spent).
	spent bool
}

// args builds the strings and out-parameters of one call on the shadow
// stack.
type args struct{ top uintptr }

func (c *Conn) args() args { return args{c.sp} }

// str copies s with a NUL after it.
func (a *args) str(s string) uintptr {
	p := a.top
	b := bytesAt(p, len(s)+1)
	copy(b, s)
	b[len(s)] = 0
	a.top = (p + uintptr(len(s)) + 1 + 15) &^ 15
	return p
}

// bytes copies b.
func (a *args) bytes(b []byte) uintptr {
	p := a.top
	copy(bytesAt(p, len(b)), b)
	a.top = (p + uintptr(len(b)) + 15) &^ 15
	return p
}

// space is room for n bytes.
func (a *args) space(n int) uintptr {
	p := a.top
	a.top = (p + uintptr(n) + 15) &^ 15
	return p
}

// word is room for one out-parameter, zeroed.
func (a *args) word() uintptr {
	p := a.top
	*(*uintptr)(at(p)) = 0
	a.top += 16
	return p
}

// big is the largest string or blob passed on the shadow stack: a larger one
// is passed from malloc'd memory.
const big = 1 << 20

// maxName is the longest file name, VFS name or schema name passed on the
// shadow stack. Real paths are a few KiB at most.
const maxName = 1 << 16

// Open opens a connection. It does not take OpenFullMutex: a connection has
// one shadow stack, so two goroutines cannot run on it at once, and a mutex
// of SQLite could not tell them apart.
func Open(name string, flags int, vfs string) (*Conn, error) {
	if flags&OpenFullMutex != 0 {
		return nil, &Error{code: 21, msg: "OpenFullMutex is not supported: a connection is used by one goroutine at a time"}
	}
	if len(name) > maxName || len(vfs) > maxName {
		return nil, &Error{code: 18, msg: "file name too long"}
	}
	if err := start(); err != nil {
		return nil, err
	}
	st, err := newStack()
	if err != nil {
		return nil, err
	}
	c := &Conn{st: st, sp: st.sp()}
	a := c.args()
	pdb := a.word()
	zName := a.str(name)
	var zVfs uintptr
	if vfs != "" {
		zVfs = a.str(vfs)
	}
	r := int32(sqlite3_open_v2(a.top, zName, pdb, uintptr(flags|OpenExResCode), zVfs))
	c.db = *(*uintptr)(at(pdb))
	c.st.db = c.db
	if r == OK {
		r = int32(go_classify(c.sp, c.db, c.classifyWord()))
	}
	if r == OK {
		r = int32(go_count_rollbacks(c.sp, c.db, c.st.base+rollbackOffset))
	}
	if r != OK {
		err := c.err(int(r))
		if c.db != 0 {
			sqlite3_close(c.sp, c.db)
		}
		st.free()
		return nil, err
	}
	return c, nil
}

// classifyWord is where goClassify writes the kind of the statement being
// prepared.
func (c *Conn) classifyWord() uintptr { return c.st.base + classifyOffset }

// The values of goClassifyWord.merging (engine/os.c).
const (
	merging = 1
	refused = 2
)

// SetMerging says whether the connection runs statements of other
// connections' callers (group commit). While it does, a statement that
// SQLite prepares again after a change of the schema and that is no longer
// mergeable fails before it does anything. Turned off, SetMerging reports
// whether that happened since it was turned on: the statement must run on
// its caller's connection, and Reclassify sorts it again.
func (c *Conn) SetMerging(on bool) (wasRefused bool) {
	w := (*int32)(at(c.classifyWord() + 4))
	wasRefused = *w == refused
	*w = 0
	if on {
		*w = merging
		*(*int32)(at(c.st.base + effectsOffset)) = 0
	}
	return wasRefused
}

// RanEffects reports whether a Go function that may have side effects (one
// not marked FuncInnocuous) ran on the connection since SetMerging(true).
func (c *Conn) RanEffects() bool {
	return *(*int32)(at(c.st.base + effectsOffset)) != 0
}

// The bits goClassify writes (engine/os.c).
const (
	writesMain  = 1
	private     = 2
	changesConn = 4
	// 8 is guarded (replica.go).
	readsWALAuto = 16 // the statement reads PRAGMA wal_autocheckpoint
	// mayWrite: the statement may write though sqlite3_stmt_readonly says it
	// does not (PRAGMA optimize, which runs ANALYZE inside it).
	mayWrite = 32
)

// Spent says SQLite ran out of memory, failed to read or write a file or
// found the disk full on the connection. These are the errors that make
// SQLite roll a transaction back at once, also when the statement that
// failed is one an extension runs inside a statement of the caller, which is
// still running. FTS5 then goes on with what it had read before the rollback
// and keeps it for its next write: the count of rows it failed to read, as
// zero. SQLite tells no one that this happened, so a connection that gave
// one of these errors is not to start anything more.
//
// A checkpoint the driver runs itself (Checkpoint) does not count: it runs
// no statement, so no extension is in the middle of one.
func (c *Conn) Spent() bool { return c.spent }

// err is the error of result code r, with the connection's message. It
// marks the connection spent for the codes of Spent.
func (c *Conn) err(r int) error {
	switch r & 0xff {
	case sqliteNoMem, sqliteIOErr, sqliteFull:
		c.spent = true
	}
	return c.errOf(r)
}

// errOf is the error of result code r, with the connection's message.
func (c *Conn) errOf(r int) error {
	if msg := c.st.collatePanic; msg != "" {
		// go_collate_failed made SQLite stop with SQLITE_NOMEM and roll
		// back what the statement did.
		c.st.collatePanic = ""
		if r == OK || r == Row || r == Done {
			// SQLite went on after the panic: nothing it does on this
			// connection can be trusted any more.
			poison(c.st.base, "a Go collation panicked and SQLite went on: "+msg)
		}
		return &Error{code: r, msg: "panic in a Go collation: " + msg, err: ErrCollationPanic}
	}
	if r == OK || r == Row || r == Done {
		return nil
	}
	if c.db == 0 {
		return &Error{code: r, msg: errstr(c.sp, r)}
	}
	return &Error{code: r, msg: cstring(sqlite3_errmsg(c.sp, c.db))}
}

// Close closes the connection. Its statements must be finalized first: with
// one open, Close returns SQLITE_BUSY and the connection stays open and usable,
// since SQLite would otherwise keep a half-closed connection that runs on a
// shadow stack already given back. A broken connection runs no C: Close frees
// its stack and returns the error that says so.
func (c *Conn) Close() error {
	if c.db == 0 {
		return errors.New("sqlite: connection already closed")
	}
	if err := c.brokenErr(); err != nil {
		c.db = 0
		c.st.free()
		return err
	}
	if r := int32(sqlite3_close(c.sp, c.db)); r != OK {
		return c.err(int(r))
	}
	c.db = 0
	return c.st.free()
}

// Exec runs one or more statements that return no rows.
func (c *Conn) Exec(sql string) error {
	if err := c.brokenErr(); err != nil {
		return err
	}
	if len(sql) > big {
		p := stageAlloc(uintptr(len(sql) + 1))
		if p == 0 {
			return errNoMem()
		}
		defer release(p)
		copy(bytesAt(p, len(sql)), sql)
		*(*byte)(at(p + uintptr(len(sql)))) = 0
		*(*int32)(at(c.classifyWord())) = 0
		return c.execResult(int32(sqlite3_exec(c.sp, c.db, p, 0, 0, 0)))
	}
	a := c.args()
	z := a.str(sql)
	*(*int32)(at(c.classifyWord())) = 0
	return c.execResult(int32(sqlite3_exec(a.top, c.db, z, 0, 0, 0)))
}

func (c *Conn) execResult(r int32) error {
	if err := c.guardErr(r); err != nil {
		return err
	}
	return c.err(int(r))
}

// Prepare compiles the first statement of sql. tail is the rest of sql. A
// nil statement with no error is sql with no statement in it.
func (c *Conn) Prepare(sql string) (s *Stmt, tail string, err error) {
	c.justChanged = false
	if err := c.brokenErr(); err != nil {
		return nil, "", err
	}
	if noStatement(sql) {
		// What follows the last statement of an Exec. SQLite would copy it
		// to find that out, and a copy that got no memory would fail the
		// Exec after its statements committed.
		return nil, "", nil
	}
	return c.prepare(sql)
}

// prepare is Prepare once the connection is known to work and sql is given
// to SQLite.
func (c *Conn) prepare(sql string) (s *Stmt, tail string, err error) {
	a := c.args()
	pStmt := a.word()
	pTail := a.word()
	var z uintptr
	if len(sql) > big {
		z = stageAlloc(uintptr(len(sql) + 1))
		if z == 0 {
			return nil, "", errNoMem()
		}
		defer release(z)
		copy(bytesAt(z, len(sql)), sql)
	} else {
		z = a.bytes([]byte(sql))
	}
	w := (*int32)(at(c.classifyWord()))
	*w = 0
	r := int32(sqlite3_prepare_v2(a.top, c.db, z, uintptr(len(sql)), pStmt, pTail))
	kind := *w
	if kind&readsWALAuto != 0 && int32(go_wal_give(a.top, c.db)) != 0 {
		// SQLite codes what PRAGMA wal_autocheckpoint reads into the
		// statement as it prepares it, and reads 0 while the automatic
		// checkpoints are taken from it (TakeAutoCheckpoint). So the
		// statement is prepared once more with SQLite's own in place.
		if p := *(*uintptr)(at(pStmt)); r == OK && p != 0 {
			sqlite3_finalize(a.top, p)
		}
		*w = 0
		r = int32(sqlite3_prepare_v2(a.top, c.db, z, uintptr(len(sql)), pStmt, pTail))
		kind = *w
		go_wal_take(a.top, c.db)
	}
	c.justChanged = kind&changesConn != 0
	if c.justChanged {
		c.changed = true
	}
	if err := c.guardErr(r); err != nil {
		return nil, "", err
	}
	if r != OK {
		return nil, "", c.err(int(r))
	}
	if t := *(*uintptr)(at(pTail)); t != 0 {
		tail = sql[t-z:]
	}
	p := *(*uintptr)(at(pStmt))
	if p == 0 {
		return nil, tail, nil
	}
	writes := int32(sqlite3_stmt_readonly(a.top, p)) == 0 || kind&mayWrite != 0
	if int32(sqlite3_stmt_isexplain(a.top, p)) != 0 {
		// EXPLAIN lists the program of the statement and runs none of it.
		// goClassify saw the INSERT of an EXPLAIN INSERT, and SQLite counts
		// it as a write, but it writes nothing: it neither merges nor takes
		// the write lock.
		kind &^= writesMain
		writes = false
	}
	return &Stmt{c: c, p: p, kind: kind, writes: writes}, tail, nil
}

// noStatement says that sql holds nothing but white space, semicolons and
// whole comments, as SQLite's tokenizer reads them: SQLite would prepare no
// statement from it. Anything it is not sure of is left to SQLite.
func noStatement(sql string) bool {
	for i := 0; i < len(sql); i++ {
		switch sql[i] {
		case ' ', '\t', '\n', '\f', '\r', ';':
		case '-':
			if i+1 == len(sql) || sql[i+1] != '-' {
				return false
			}
			n := strings.IndexByte(sql[i:], '\n')
			if n < 0 {
				return true
			}
			i += n
		case '/':
			if i+1 == len(sql) || sql[i+1] != '*' {
				return false
			}
			n := strings.Index(sql[i+2:], "*/")
			if n < 0 {
				return false
			}
			i += n + 3
		default:
			return false
		}
	}
	return true
}

// rollbacks is how many transactions SQLite has rolled back on the
// connection (go_count_rollbacks).
func (c *Conn) rollbacks() uint32 { return *(*uint32)(at(c.st.base + rollbackOffset)) }

// ErrRolledBack is the error of a statement that wrote, at its end, when
// SQLite rolled back its transaction while it ran: another statement of the
// connection failed in a way that ends the transaction (out of memory, an
// I/O error, a full disk, a panic in a Go collation), or ROLLBACK ran.
var ErrRolledBack = errors.New("sqlite: the transaction of the statement was rolled back while it ran: what it wrote is gone")

// Changes is the number of rows the last statement changed.
func (c *Conn) Changes() int64 { return int64(sqlite3_changes64(c.sp, c.db)) }

// LastInsertRowID is the rowid of the last row inserted.
func (c *Conn) LastInsertRowID() int64 { return int64(sqlite3_last_insert_rowid(c.sp, c.db)) }

// SetLastInsertRowID sets what LastInsertRowID and last_insert_rowid() give.
func (c *Conn) SetLastInsertRowID(id int64) { sqlite3_set_last_insert_rowid(c.sp, c.db, uintptr(id)) }

// VirtualTablesInTx says virtual tables take part in the open transaction:
// one was written since it began.
func (c *Conn) VirtualTablesInTx() bool { return int32(go_vtabs_in_tx(c.sp, c.db)) != 0 }

// DeferredFKs says foreign key constraints of the transaction are not
// resolved yet (SQLITE_DBSTATUS_DEFERRED_FKS).
func (c *Conn) DeferredFKs() bool {
	a := c.args()
	cur, hi := a.word(), a.word()
	sqlite3_db_status(c.sp, c.db, 10, cur, hi, 0) // SQLITE_DBSTATUS_DEFERRED_FKS
	return *(*int32)(at(cur)) != 0
}

// ReadOnly says the main database is open read-only.
func (c *Conn) ReadOnly() bool {
	a := c.args()
	return int32(sqlite3_db_readonly(c.sp, c.db, a.str("main"))) == 1
}

// AutoCommit says whether no transaction is open.
func (c *Conn) AutoCommit() bool { return int32(sqlite3_get_autocommit(c.sp, c.db)) != 0 }

// InTx says SQLite has a transaction open on the connection, a read one
// too: in autocommit, a query not done holds one, with the snapshot it read.
func (c *Conn) InTx() bool { return int32(sqlite3_txn_state(c.sp, c.db, 0)) != 0 }

// BusyTimeout sets how long a statement waits for a lock.
func (c *Conn) BusyTimeout(ms int) error {
	if err := c.brokenErr(); err != nil {
		return err
	}
	return c.err(int(int32(sqlite3_busy_timeout(c.sp, c.db, uintptr(ms)))))
}

// BusyTimeoutNow is the busy timeout of the connection in milliseconds, 0
// for none, as PRAGMA busy_timeout reads it. ok is false when the
// connection is broken. It runs no statement, so it also tells after a
// statement that failed, when SQLite may refuse another: an interrupt stays
// while a query of the connection is open.
func (c *Conn) BusyTimeoutNow() (ms int, ok bool) {
	if c.brokenErr() != nil {
		return 0, false
	}
	return int(int32(go_busy_timeout(c.sp, c.db))), true
}

// TakeAutoCheckpoint takes the automatic checkpoints of the connection from
// SQLite, which runs none of them from then on: the caller does. n is the
// wal_autocheckpoint the connection had, 0 for none. ok is false when they
// were taken already. PRAGMA wal_autocheckpoint=N gives them back to SQLite
// with N, as soon as it is prepared, and a call after it takes them again
// and tells N. PRAGMA wal_autocheckpoint reads the same either way.
func (c *Conn) TakeAutoCheckpoint() (n int, ok bool) {
	if c.brokenErr() != nil {
		return 0, false
	}
	n = int(int32(go_wal_take(c.sp, c.db)))
	return n, n >= 0
}

// GiveAutoCheckpoint gives the automatic checkpoints of the connection back
// to SQLite, with the wal_autocheckpoint they were taken with. It says
// whether they were taken.
func (c *Conn) GiveAutoCheckpoint() bool {
	return c.brokenErr() == nil && int32(go_wal_give(c.sp, c.db)) != 0
}

// Attached says the connection has a database attached.
func (c *Conn) Attached() bool {
	return c.brokenErr() == nil && int32(go_attached(c.sp, c.db)) != 0
}

// Stmt is a prepared statement.
type Stmt struct {
	c    *Conn
	p    uintptr
	kind int32 // what goClassify saw as it was prepared
	// sorted is the count of Reprepares when kind was read, or -1 when the
	// statement must be sorted again.
	sorted int
	// writes says the statement is not read-only. paused says it stopped
	// at a row, when the connection had rolled back rollbacks
	// transactions.
	writes    bool
	paused    bool
	rollbacks uint32
}

// lost says whether a statement that writes lost its transaction while it
// stopped at a row: another statement ended it. A rollback inside its own
// step is its own error, already in the result, or SQLite running it again
// after a change of the schema, after which it goes on and commits.
func (s *Stmt) lost() bool {
	lost := s.paused && s.c.rollbacks() != s.rollbacks
	s.paused = false
	return lost
}

// Mergeable says the statement changes tables of the main database and
// touches nothing only its connection has: run on another connection of
// the same file, it does the same. After SQLite prepared the statement again
// (a change of the schema can add a trigger or make a table a virtual one),
// it is sorted again first.
func (s *Stmt) Mergeable() bool {
	if p := s.Reprepares(); p != s.sorted {
		s.kind, s.sorted = s.classify(), p
	}
	return s.kind == writesMain
}

// Reclassify makes the next Mergeable sort the statement again: SQLite
// failed to prepare it again while the connection merged (SetMerging).
func (s *Stmt) Reclassify() { s.sorted = -1 }

// classify prepares the SQL of the statement once more and gives what
// goClassify sees now. SQLite prepares it again in sqlite3_step from the
// same SQL and the same schema. A statement that no longer prepares is not
// mergeable: run on its own connection, it gives its own error.
func (s *Stmt) classify() int32 {
	t, _, err := s.c.Prepare(cstring(sqlite3_sql(s.c.sp, s.p)))
	if err != nil || t == nil {
		return private
	}
	defer t.Finalize()
	return t.kind
}

// ChangesConn says the statement changes the connection itself: a PRAGMA
// that sets something, ATTACH, DETACH, or a temporary table, index, view or
// trigger.
func (s *Stmt) ChangesConn() bool { return s.kind&changesConn != 0 }

// Private says the statement, as prepared, is not plain DML or a query of the
// main database: a PRAGMA, ATTACH, a read of another schema, a function that
// reads the connection's state (goClassify's GO_PRIVATE).
func (s *Stmt) Private() bool { return s.kind&private != 0 }

// IsExplain says the statement is an EXPLAIN or EXPLAIN QUERY PLAN.
func (s *Stmt) IsExplain() bool { return int32(sqlite3_stmt_isexplain(s.c.sp, s.p)) != 0 }

// ChangedItself says a statement that changes the connection was prepared on
// it, run or not.
func (c *Conn) ChangedItself() bool { return c.changed }

// JustChanged says the last Prepare was of a statement that changes the
// connection, even one that failed after SQLite applied it (PRAGMA
// busy_timeout = 0 junk).
func (c *Conn) JustChanged() bool { return c.justChanged }

// Reprepares is how many times SQLite prepared the statement again, after a
// change of the schema.
func (s *Stmt) Reprepares() int { return int(int32(sqlite3_stmt_status(s.c.sp, s.p, 5, 0))) } // SQLITE_STMTSTATUS_REPREPARE

func (s *Stmt) err(r uintptr) error { return s.c.err(int(int32(r))) }

// Finalize frees the statement.
func (s *Stmt) Finalize() error {
	if err := s.c.brokenErr(); err != nil {
		s.p = 0
		return err
	}
	r := sqlite3_finalize(s.c.sp, s.p)
	s.p = 0
	return s.err(r)
}

// Reset makes the statement ready to run again. Bindings stay.
func (s *Stmt) Reset() error {
	if err := s.c.brokenErr(); err != nil {
		return err
	}
	lost := s.lost()
	// A statement stopped before its end finishes its write here.
	err := s.err(sqlite3_reset(s.c.sp, s.p))
	if lost && err == nil {
		err = ErrRolledBack
	}
	return err
}

// ClearBindings sets every parameter to NULL.
func (s *Stmt) ClearBindings() error {
	if err := s.c.brokenErr(); err != nil {
		return err
	}
	return s.err(sqlite3_clear_bindings(s.c.sp, s.p))
}

// Step runs the statement to its next row: true when there is one.
func (s *Stmt) Step() (bool, error) {
	if err := s.c.brokenErr(); err != nil {
		return false, err
	}
	if s.lost() {
		// The rows still to come are of a write that is gone.
		sqlite3_reset(s.c.sp, s.p)
		return false, ErrRolledBack
	}
	// A statement that reads PRAGMA wal_autocheckpoint is prepared again by
	// SQLite each time it runs, so it runs with SQLite's own automatic
	// checkpoints in place, as it was prepared (Prepare).
	given := s.kind&readsWALAuto != 0 && int32(go_wal_give(s.c.sp, s.c.db)) != 0
	r := int32(sqlite3_step(s.c.sp, s.p))
	// A callback that panicked during the step leaves no trace in r: the
	// row it gave, or its end, may be wrong.
	if err := s.c.brokenErr(); err != nil {
		return false, err
	}
	if given {
		go_wal_take(s.c.sp, s.c.db)
	}
	switch r {
	case Row:
		if s.writes {
			s.paused, s.rollbacks = true, s.c.rollbacks()
		}
		return true, nil
	case Done:
		return false, nil
	default:
		return false, s.c.err(int(r))
	}
}

// ReadOnly says whether the statement writes nothing: SQLite counts it as
// read-only, or it is an EXPLAIN.
func (s *Stmt) ReadOnly() bool { return !s.writes }

// ParamCount is the number of the largest parameter.
func (s *Stmt) ParamCount() int { return int(int32(sqlite3_bind_parameter_count(s.c.sp, s.p))) }

// ParamName is the name of parameter i (from 1), with its prefix, or "".
func (s *Stmt) ParamName(i int) string {
	return cstring(sqlite3_bind_parameter_name(s.c.sp, s.p, uintptr(i)))
}

func (s *Stmt) BindNull(i int) error {
	if err := s.c.brokenErr(); err != nil {
		return err
	}
	return s.err(sqlite3_bind_null(s.c.sp, s.p, uintptr(i)))
}

func (s *Stmt) BindInt64(i int, v int64) error {
	if err := s.c.brokenErr(); err != nil {
		return err
	}
	return s.err(sqlite3_bind_int64(s.c.sp, s.p, uintptr(i), uintptr(v)))
}

func (s *Stmt) BindDouble(i int, v float64) error {
	if err := s.c.brokenErr(); err != nil {
		return err
	}
	return s.err(sqlite3_bind_double(s.c.sp, s.p, uintptr(i), v))
}

// transient tells SQLite to copy a string or blob before the call returns.
const transient = ^uintptr(0)

func (s *Stmt) BindText(i int, v string) error {
	if err := s.c.brokenErr(); err != nil {
		return err
	}
	if len(v) > math.MaxInt32 {
		return &Error{code: 18, msg: "string or blob too big"}
	}
	z, done, err := s.c.stage(unsafe.StringData(v), len(v))
	if err != nil {
		return err
	}
	defer done()
	return s.err(sqlite3_bind_text(s.c.sp+stageRoom(len(v)), s.p, uintptr(i), z, uintptr(len(v)), transient))
}

func (s *Stmt) BindBlob(i int, v []byte) error {
	if err := s.c.brokenErr(); err != nil {
		return err
	}
	if len(v) > math.MaxInt32 {
		return &Error{code: 18, msg: "string or blob too big"}
	}
	if v == nil {
		v = []byte{}
	}
	z, done, err := s.c.stage(unsafe.SliceData(v), len(v))
	if err != nil {
		return err
	}
	defer done()
	return s.err(sqlite3_bind_blob(s.c.sp+stageRoom(len(v)), s.p, uintptr(i), z, uintptr(len(v)), transient))
}

// stage copies n bytes from Go memory to where the C can read them: the
// bottom of the shadow stack when they fit, malloc'd memory when not.
func (c *Conn) stage(b *byte, n int) (uintptr, func(), error) { return stageAt(c.sp, b, n) }

// stageAt is stage from the shadow pointer sp, for a host callback that
// calls the C again.
func stageAt(sp uintptr, b *byte, n int) (uintptr, func(), error) {
	src := unsafe.Slice(b, n)
	if n <= big {
		copy(bytesAt(sp, n), src)
		return sp, func() {}, nil
	}
	p := stageAlloc(uintptr(n))
	if p == 0 {
		return 0, nil, errNoMem()
	}
	copy(bytesAt(p, n), src)
	return p, func() { release(p) }, nil
}

// stageAlloc is where the Go side gets memory for an argument too big for the
// shadow stack. It is a variable so that a test can make it fail.
var stageAlloc = alloc

func errNoMem() error { return &Error{code: sqliteNoMem, msg: "out of memory"} }

// stageRoom is the shadow stack stage takes for n bytes.
func stageRoom(n int) uintptr {
	if n > big {
		return 0
	}
	return (uintptr(n) + 15) &^ 15
}

// ColumnCount is the number of columns of a row.
func (s *Stmt) ColumnCount() int { return int(int32(sqlite3_column_count(s.c.sp, s.p))) }

func (s *Stmt) ColumnName(i int) string {
	return cstring(sqlite3_column_name(s.c.sp, s.p, uintptr(i)))
}

// ColumnDeclType is the declared type of column i, or "".
func (s *Stmt) ColumnDeclType(i int) string {
	return cstring(sqlite3_column_decltype(s.c.sp, s.p, uintptr(i)))
}

func (s *Stmt) ColumnType(i int) int { return int(int32(sqlite3_column_type(s.c.sp, s.p, uintptr(i)))) }

func (s *Stmt) ColumnInt64(i int) int64 { return int64(sqlite3_column_int64(s.c.sp, s.p, uintptr(i))) }

func (s *Stmt) ColumnDouble(i int) float64 { return sqlite3_column_double(s.c.sp, s.p, uintptr(i)) }

// ColumnText is the text of column i, copied. It fails when SQLite has no
// memory for it: it allocates to give as UTF-8 a text it holds as UTF-16
// (a database in UTF-16) or a value that is no text.
func (s *Stmt) ColumnText(i int) (string, error) {
	p := sqlite3_column_text(s.c.sp, s.p, uintptr(i))
	if p == 0 {
		return "", s.columnErr()
	}
	n := int(int32(sqlite3_column_bytes(s.c.sp, s.p, uintptr(i))))
	if n == 0 {
		return "", nil
	}
	return string(bytesAt(p, n)), nil
}

// ColumnBlob is the bytes of column i, copied: nil for NULL and for a blob
// of no bytes. It fails when SQLite has no memory for it.
func (s *Stmt) ColumnBlob(i int) ([]byte, error) {
	p := sqlite3_column_blob(s.c.sp, s.p, uintptr(i))
	if p == 0 {
		return nil, s.columnErr()
	}
	n := int(int32(sqlite3_column_bytes(s.c.sp, s.p, uintptr(i))))
	b := make([]byte, n)
	copy(b, bytesAt(p, n))
	return b, nil
}

// columnErr is the error of a call that gave no pointer for a column: out
// of memory, or nil when the value is NULL or empty. SQLite gives NULL for
// both and tells them apart by the connection's error code only, which a
// step that gave a row left at SQLITE_ROW.
func (s *Stmt) columnErr() error {
	if int32(sqlite3_errcode(s.c.sp, s.c.db))&0xff == sqliteNoMem {
		return s.c.err(sqliteNoMem)
	}
	return nil
}

// sideStacks are shadow stacks for calls made beside a connection's own,
// from another goroutine. They are kept, not unmapped: there are as many as
// such calls ever ran at once.
var sideStacks struct {
	sync.Mutex
	free []*stack
}

func getSideStack() (*stack, error) {
	sideStacks.Lock()
	defer sideStacks.Unlock()
	if n := len(sideStacks.free); n > 0 {
		s := sideStacks.free[n-1]
		sideStacks.free = sideStacks.free[:n-1]
		return s, nil
	}
	return newStack()
}

func putSideStack(s *stack) {
	sideStacks.Lock()
	sideStacks.free = append(sideStacks.free, s)
	sideStacks.Unlock()
}

// Interrupt stops the statement running on the connection, from any
// goroutine. The statement returns SQLITE_INTERRUPT.
func (c *Conn) Interrupt() error {
	s, err := getSideStack()
	if err != nil {
		return err
	}
	sqlite3_interrupt(s.sp(), c.db)
	putSideStack(s)
	return nil
}

// Filenames are the absolute names of the files of the databases of the
// connection, main and attached, without memory and temporary ones.
func (c *Conn) Filenames() []string {
	if c.brokenErr() != nil {
		return nil
	}
	var names []string
	for i := 0; ; i++ {
		a := c.args()
		z := sqlite3_db_name(a.top, c.db, uintptr(i))
		if z == 0 {
			return names
		}
		if name := cstring(sqlite3_db_filename(a.top, c.db, z)); name != "" {
			names = append(names, name)
		}
	}
}

// Attachments are the databases attached to the connection (ATTACH) that
// are files: their schema names, and the absolute names of their files.
func (c *Conn) Attachments() (schemas, files []string) {
	if c.brokenErr() != nil {
		return nil, nil
	}
	// 0 is main and 1 is temp.
	for i := 2; ; i++ {
		a := c.args()
		z := sqlite3_db_name(a.top, c.db, uintptr(i))
		if z == 0 {
			return schemas, files
		}
		if name := cstring(sqlite3_db_filename(a.top, c.db, z)); name != "" {
			schemas = append(schemas, cstring(z))
			files = append(files, name)
		}
	}
}

// Filename is the absolute name of the file of database schema ("main"), or
// "" for a memory or temporary database.
func (c *Conn) Filename(schema string) string {
	if len(schema) > maxName || c.brokenErr() != nil {
		return ""
	}
	a := c.args()
	z := a.str(schema)
	return cstring(sqlite3_db_filename(a.top, c.db, z))
}
