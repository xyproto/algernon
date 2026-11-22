// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64))

package csqlite

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"math"
	"strings"
)

// Module makes the table of a virtual table module for a connection: for
// CREATE VIRTUAL TABLE t USING name(args...), when a connection first uses
// such a table, and for the module's name used as a table. args are the
// arguments after the name, as SQL wrote them.
type Module func(args []string) (VTable, error)

// VTable is a table of a module on one connection. It only reads.
type VTable interface {
	// Schema is the CREATE TABLE statement of its columns. Its name is not
	// used. A column marked HIDDEN is an argument of the module's name used
	// as a function.
	Schema() string
	// BestIndex says how a scan with the constraints of info would go. It
	// returns ErrConstraint when the scan cannot go with the constraints
	// that are usable.
	BestIndex(info *IndexInfo) error
	// Open makes a cursor for a scan.
	Open() (VCursor, error)
	// Disconnect ends the table on its connection.
	Disconnect() error
}

// VCursor is a scan of a VTable.
type VCursor interface {
	// Filter starts the scan, with the plan of BestIndex: idxNum and idxStr
	// as it set them, and the values of the constraints it gave an
	// ArgvIndex, in that order.
	Filter(idxNum int, idxStr string, args []driver.Value) error
	Next() error
	// EOF says the scan is past its last row.
	EOF() bool
	// Column is column i of the row, as a Func result.
	Column(i int) (driver.Value, error)
	RowID() (int64, error)
	Close() error
}

// Flags of CreateModule.
const (
	// ModEponymousOnly: the module is only used by its name, as a function
	// whose arguments are its hidden columns. CREATE VIRTUAL TABLE fails.
	ModEponymousOnly = 1 << iota
	// ModDirectOnly: its tables may not be used by triggers and views.
	ModDirectOnly
	// ModInnocuous: its tables are safe wherever SQL comes from.
	ModInnocuous
	// ModWritable: its tables take INSERT, UPDATE and DELETE when they are
	// VUpdaters, and those statements never merge. Not with ModEponymousOnly.
	ModWritable
)

// ErrConstraint is what BestIndex returns for a plan that cannot work.
var ErrConstraint = errors.New("no usable plan for the virtual table")

// The operators of IndexConstraint (SQLITE_INDEX_CONSTRAINT_*).
const (
	IndexEQ        = 2
	IndexGT        = 4
	IndexLE        = 8
	IndexLT        = 16
	IndexGE        = 32
	IndexMatch     = 64
	IndexLike      = 65
	IndexGlob      = 66
	IndexRegexp    = 67
	IndexNE        = 68
	IndexIsNot     = 69
	IndexIsNotNull = 70
	IndexIsNull    = 71
	IndexIs        = 72
	IndexLimit     = 73
	IndexOffset    = 74
	IndexFunction  = 150
)

// IndexConstraint is a term of the WHERE clause on one column. A term that
// is not usable may not get an ArgvIndex.
type IndexConstraint struct {
	Column int // -1 for the rowid
	Op     int
	Usable bool
}

// IndexOrderBy is a term of the ORDER BY clause.
type IndexOrderBy struct {
	Column int
	Desc   bool
}

// IndexUsage is what BestIndex says of a constraint: ArgvIndex > 0 gives
// its value to Filter at args[ArgvIndex-1], and Omit says SQLite need not
// check it again.
type IndexUsage struct {
	ArgvIndex int
	Omit      bool
}

// IndexInfo is the question of xBestIndex and its answer.
type IndexInfo struct {
	Constraints []IndexConstraint
	OrderBy     []IndexOrderBy
	ColUsed     uint64 // bit i: column i is used, bit 63: a column from 63 up is

	Usage           []IndexUsage // as long as Constraints
	IdxNum          int
	IdxStr          string
	OrderByConsumed bool
	EstimatedCost   float64
	EstimatedRows   int64
	Unique          bool // at most one row (SQLITE_INDEX_SCAN_UNIQUE)
}

type moduleEntry struct {
	connect Module
	flags   int
}

// modules, vtables and vcursors hold what SQLite knows by handle.
var (
	modules  table[moduleEntry]
	vtables  table[VTable]
	vcursors table[VCursor]
)

// CreateModule makes m the virtual table module name on the connection.
func (c *Conn) CreateModule(name string, m Module, flags int) error {
	if err := c.brokenErr(); err != nil {
		return err
	}
	if len(name) > maxName {
		return &Error{code: 21, msg: "module name too long"}
	}
	if flags&ModDirectOnly != 0 && flags&ModInnocuous != 0 {
		// SQLite keeps one risk level per table: the second would undo the
		// first.
		return &Error{code: 21, msg: "a module cannot be both DirectOnly and Innocuous"}
	}
	if flags&ModEponymousOnly != 0 && flags&ModWritable != 0 {
		return &Error{code: 21, msg: "a module cannot be both EponymousOnly and writable"}
	}
	h := modules.add(&moduleEntry{m, flags})
	a := c.args()
	z := a.str(name)
	// SQLite gives h to host_module_destroy when this fails as well.
	kind := uintptr(0) // GO_MODULE_PLAIN
	switch {
	case flags&ModEponymousOnly != 0:
		kind = 1 // GO_MODULE_EPONYMOUS
	case flags&ModWritable != 0:
		kind = 2 // GO_MODULE_WRITABLE
	}
	return c.err(int(int32(go_create_module(a.top, c.db, z, h, kind))))
}

const (
	sqliteError      = 1
	sqliteConstraint = 19
)

// run runs user code and turns a panic in it into an error.
func run(f func() error) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("panic in a Go virtual table: %v", e)
		}
	}()
	return f()
}

// host_vtab_connect connects a table of a module (engine/host.h's struct
// host_vtab_connect_args at a) and declares its schema.
func host_vtab_connect(sp, a uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteError)
	db := *(*uintptr)(at(a))
	m := modules.get(*(*uintptr)(at(a + 8)))
	argv := *(*uintptr)(at(a + 16))
	argc := int(*(*int32)(at(a + 24)))
	// argv holds the module, the schema and the table, then the arguments.
	margs := make([]string, 0, max(argc-3, 0))
	for i := 3; i < argc; i++ {
		margs = append(margs, cstring(*(*uintptr)(at(argv + uintptr(i)*8))))
	}
	var t VTable
	var schema string
	err := run(func() error {
		var err error
		if t, err = m.connect(margs); err != nil {
			t = nil // not ours to disconnect
			return err
		}
		if t == nil {
			return errors.New("the module made a nil table")
		}
		schema = t.Schema()
		if strings.IndexByte(schema, 0) >= 0 {
			return errors.New("the schema of the table has a NUL")
		}
		return nil
	})
	if err == nil {
		z, top, free, serr := cstr(sp, schema)
		if serr != nil {
			err = serr
		} else {
			r := int32(sqlite3_declare_vtab(top, db, z))
			free()
			if r != OK {
				err = fmt.Errorf("the schema of the table: %s", cstring(sqlite3_errmsg(sp, db)))
			}
		}
	}
	if err != nil {
		if t != nil {
			run(t.Disconnect)
		}
		s := args{sp}
		z := s.str(cutText(err))
		*(*uintptr)(at(a + 40)) = go_strdup(s.top, z)
		return sqliteError
	}
	config := 0
	if m.flags&ModDirectOnly != 0 {
		config |= 1 // GO_VTAB_DIRECTONLY
	}
	if m.flags&ModInnocuous != 0 {
		config |= 2 // GO_VTAB_INNOCUOUS
	}
	*(*int32)(at(a + 28)) = int32(config)
	*(*int64)(at(a + 32)) = int64(vtables.add(&t))
	return OK
}

// cutText is the text of err cut to 64 KiB, so that it fits on the shadow
// stack: SQLite makes its own copy.
func cutText(err error) string {
	const most = 1 << 16
	msg := errText(err)
	if len(msg) > most {
		msg = msg[:most]
	}
	return msg
}

// cstr puts s and a NUL after it where the C can read it: on the shadow
// stack at sp when it fits, in malloc'd memory when not. top is the free
// shadow stack above it, and free gives the memory back.
func cstr(sp uintptr, s string) (z, top uintptr, free func(), err error) {
	if len(s) < big {
		a := args{sp}
		z := a.str(s)
		return z, a.top, func() {}, nil
	}
	p := stageAlloc(uintptr(len(s)) + 1)
	if p == 0 {
		return 0, 0, nil, errNoMem()
	}
	b := bytesAt(p, len(s)+1)
	copy(b, s)
	b[len(s)] = 0
	return p, sp, func() { release(p) }, nil
}

// vtabError makes err the error of the table v.
func vtabError(sp, v uintptr, err error) uintptr {
	a := args{sp}
	z := a.str(cutText(err))
	go_vtab_error(a.top, v, z)
	return sqliteError
}

// host_vtab_best_index asks the table h of v for a plan.
func host_vtab_best_index(sp, v, h, p uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteError)
	t := *vtables.get(h)
	n := int(*(*int32)(at(p)))
	cons := *(*uintptr)(at(p + 8))
	nOrder := int(*(*int32)(at(p + 16)))
	order := *(*uintptr)(at(p + 24))
	usage := *(*uintptr)(at(p + 32))
	info := &IndexInfo{
		Constraints:   make([]IndexConstraint, n),
		OrderBy:       make([]IndexOrderBy, nOrder),
		ColUsed:       *(*uint64)(at(p + 88)),
		Usage:         make([]IndexUsage, n),
		EstimatedCost: *(*float64)(at(p + 64)),
		EstimatedRows: *(*int64)(at(p + 72)),
	}
	for i := range info.Constraints {
		e := cons + uintptr(i)*12
		info.Constraints[i] = IndexConstraint{
			Column: int(*(*int32)(at(e))),
			Op:     int(*(*uint8)(at(e + 4))),
			Usable: *(*uint8)(at(e + 5)) != 0,
		}
	}
	for i := range info.OrderBy {
		e := order + uintptr(i)*8
		info.OrderBy[i] = IndexOrderBy{Column: int(*(*int32)(at(e))), Desc: *(*uint8)(at(e + 4)) != 0}
	}
	noPlan := false
	err := run(func() error {
		err := t.BestIndex(info)
		noPlan = errors.Is(err, ErrConstraint)
		return err
	})
	if noPlan {
		return sqliteConstraint
	}
	if err != nil {
		return vtabError(sp, v, err)
	}
	if strings.IndexByte(info.IdxStr, 0) >= 0 {
		return vtabError(sp, v, errors.New("BestIndex set an IdxStr with a NUL"))
	}
	if len(info.Usage) != n {
		return vtabError(sp, v, fmt.Errorf("BestIndex left %d usages for %d constraints", len(info.Usage), n))
	}
	if info.IdxNum < math.MinInt32 || info.IdxNum > math.MaxInt32 {
		return vtabError(sp, v, fmt.Errorf("BestIndex set IdxNum %d, beyond 32 bits", info.IdxNum))
	}
	for i, u := range info.Usage {
		if u.ArgvIndex < 0 || u.ArgvIndex > n {
			return vtabError(sp, v, fmt.Errorf("BestIndex gave constraint %d ArgvIndex %d", i, u.ArgvIndex))
		}
		e := usage + uintptr(i)*8
		*(*int32)(at(e)) = int32(u.ArgvIndex)
		*(*uint8)(at(e + 4)) = b2u8(u.Omit)
	}
	*(*int32)(at(p + 40)) = int32(info.IdxNum)
	if info.IdxStr != "" {
		z, top, free, err := cstr(sp, info.IdxStr)
		if err != nil {
			return sqliteNoMem
		}
		s := go_strdup(top, z)
		free()
		if s == 0 {
			return sqliteNoMem
		}
		*(*uintptr)(at(p + 48)) = s
		*(*int32)(at(p + 56)) = 1
	}
	*(*int32)(at(p + 60)) = int32(b2u8(info.OrderByConsumed))
	*(*float64)(at(p + 64)) = info.EstimatedCost
	*(*int64)(at(p + 72)) = info.EstimatedRows
	if info.Unique {
		*(*int32)(at(p + 80)) |= 1 // SQLITE_INDEX_SCAN_UNIQUE
	}
	return OK
}

func b2u8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

func host_vtab_disconnect(sp, h uintptr) (ret uintptr) {
	defer failWith(sp, &ret, OK)
	t := *vtables.get(h)
	vtables.remove(h)
	run(t.Disconnect) // SQLite has nothing to do with an error here
	return OK
}

// host_vtab_open opens a cursor of the table h of v into *out.
func host_vtab_open(sp, v, h, out uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteError)
	t := *vtables.get(h)
	var cur VCursor
	err := run(func() error {
		var err error
		cur, err = t.Open()
		if err == nil && cur == nil {
			err = errors.New("the table made a nil cursor")
		}
		return err
	})
	if err != nil {
		return vtabError(sp, v, err)
	}
	*(*int64)(at(out)) = int64(vcursors.add(&cur))
	return OK
}

func host_vtab_close(sp, h uintptr) (ret uintptr) {
	defer failWith(sp, &ret, OK)
	cur := *vcursors.get(h)
	vcursors.remove(h)
	run(cur.Close)
	return OK
}

// cursorOf is the cursor of os.c's GoVcursor c and its table.
func cursorOf(c uintptr) (VCursor, uintptr) {
	return *vcursors.get(*(*uintptr)(at(c + 8))), *(*uintptr)(at(c))
}

// setEOF writes the cursor's eof after Filter or Next. A cursor that failed
// is at its end.
func setEOF(c uintptr, cur VCursor, err error) error {
	eof := true
	if err == nil {
		err = run(func() error { eof = cur.EOF(); return nil })
	}
	*(*int32)(at(c + 16)) = int32(b2u8(eof || err != nil))
	return err
}

func host_vtab_filter(sp, c, idxNum, idxStr, argc, argv uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteError)
	cur, v := cursorOf(c)
	s := ""
	if idxStr != 0 {
		s = cstring(idxStr)
	}
	vals := values(sp, argc, argv, false)
	err := run(func() error { return cur.Filter(int(int32(idxNum)), s, vals) })
	if err = setEOF(c, cur, err); err != nil {
		return vtabError(sp, v, err)
	}
	return OK
}

func host_vtab_next(sp, c uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteError)
	cur, v := cursorOf(c)
	if err := setEOF(c, cur, run(cur.Next)); err != nil {
		return vtabError(sp, v, err)
	}
	return OK
}

// host_vtab_column sets column i of the row as the result of ctx.
func host_vtab_column(sp, c, ctx, i uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteError)
	cur, _ := cursorOf(c)
	var val driver.Value
	err := run(func() error {
		var err error
		val, err = cur.Column(int(int32(i)))
		return err
	})
	if err == nil {
		err = result(sp, ctx, val)
	}
	if err != nil {
		resultError(sp, ctx, err)
	}
	return OK
}

func host_vtab_rowid(sp, c, out uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteError)
	cur, v := cursorOf(c)
	var id int64
	err := run(func() error {
		var err error
		id, err = cur.RowID()
		return err
	})
	if err != nil {
		return vtabError(sp, v, err)
	}
	*(*int64)(at(out)) = id
	return OK
}

func host_module_destroy(sp, h uintptr) {
	defer failQuiet(sp)
	modules.remove(h)
}

// VUpdater is a VTable that takes writes, made by a module with ModWritable.
// A rowid is an int64, or the PRIMARY KEY of a WITHOUT ROWID table.
type VUpdater interface {
	// Insert adds a row of the values cols. rowid is nil when the statement
	// gave none, and Insert chooses it. It returns the rowid of the row.
	Insert(rowid driver.Value, cols []driver.Value) (int64, error)
	// Update changes the row oldRowid to newRowid and cols.
	Update(oldRowid, newRowid driver.Value, cols []driver.Value) error
	Delete(rowid driver.Value) error
}

// VTransactor is a VUpdater that takes part in the transactions of its
// connection: Begin before the first write of a transaction and as the
// table is made (goWritableCreate), Sync and Commit at its commit, Rollback
// at its rollback.
type VTransactor interface {
	Begin() error
	Sync() error
	Commit() error
	Rollback() error
}

// VSavepointer is a VTransactor that keeps savepoints, so that a statement
// that fails inside a transaction is undone: Savepoint(n) marks the state,
// RollbackTo(n) goes back to the state at savepoint n (at Begin when the
// table joined after n), Release(n) forgets the marks from n on.
type VSavepointer interface {
	Savepoint(n int) error
	Release(n int) error
	RollbackTo(n int) error
}

// host_vtab_update runs a write on the table h of v: a delete when argc is
// 1, an insert when argv[0] is NULL, an update when not.
func host_vtab_update(sp, v, h, argc, argv, rowid uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteError)
	t := *vtables.get(h)
	u, ok := t.(VUpdater)
	if !ok {
		return vtabError(sp, v, errors.New("the table takes no writes"))
	}
	vals := values(sp, argc, argv, false)
	err := run(func() error {
		switch {
		case len(vals) == 1:
			return u.Delete(vals[0])
		case vals[0] == nil:
			id, err := u.Insert(vals[1], vals[2:])
			if err == nil {
				*(*int64)(at(rowid)) = id
			}
			return err
		}
		return u.Update(vals[0], vals[1], vals[2:])
	})
	if err != nil {
		return vtabError(sp, v, err)
	}
	return OK
}

// The transaction methods of host_vtab_tx (engine/host.h).
const (
	txBegin = iota
	txSync
	txCommit
	txRollback
	txSavepoint
	txRelease
	txRollbackTo
)

// host_vtab_tx runs a transaction method on the table h of v, when it has
// one.
func host_vtab_tx(sp, v, h, op, n uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteError)
	t := *vtables.get(h)
	var f func() error
	if x, ok := t.(VTransactor); ok {
		switch op {
		case txBegin:
			f = x.Begin
		case txSync:
			f = x.Sync
		case txCommit:
			f = x.Commit
		case txRollback:
			f = x.Rollback
		}
	}
	if x, ok := t.(VSavepointer); ok {
		k := int(int32(n))
		switch op {
		case txSavepoint:
			f = func() error { return x.Savepoint(k) }
		case txRelease:
			f = func() error { return x.Release(k) }
		case txRollbackTo:
			f = func() error { return x.RollbackTo(k) }
		}
	}
	var err error
	if f != nil {
		err = run(f)
	}
	if err != nil {
		return vtabError(sp, v, err)
	}
	return OK
}
