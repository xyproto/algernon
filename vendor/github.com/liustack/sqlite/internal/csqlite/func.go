// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64))

package csqlite

import (
	"database/sql/driver"
	"fmt"
	"math"
	"unsafe"
)

// Func is a SQL function written in Go. Its arguments are copies, of the
// types int64, float64, string, []byte and nil. It returns one of those, an
// int or a bool, or an error that becomes the error of the statement. A
// panic in it is recovered and becomes the error of the statement too.
type Func func(args []driver.Value) (driver.Value, error)

// Flags of CreateFunction (sqlite3.h).
const (
	FuncDeterministic = 0x000000800
	FuncDirectOnly    = 0x000080000
	FuncInnocuous     = 0x000200000
)

// FuncBorrowBlobs is a flag of CreateFunction that is not SQLite's: the
// function gets its BLOB arguments in place, in SQLite's memory, without a
// copy. They hold until the function returns, and it must not keep them.
// Only functions of this module use it (ext/vec), where copying two
// vectors per row costs more than the work.
const FuncBorrowBlobs = 1 << 30

const utf8 = 1 // SQLITE_UTF8

// Agg is the state of one group of an aggregate function written in Go:
// Step for each row of the group, then Final once. Step's arguments are
// copies, like a Func's.
type Agg interface {
	Step(args []driver.Value) error
	Final() (driver.Value, error)
}

// A funcEntry is a function SQLite knows by handle: a scalar function or the
// maker of an aggregate's states.
type funcEntry struct {
	fn     Func
	newAgg func() Agg
	borrow bool // FuncBorrowBlobs
	// effects says the function may have side effects: it is not marked
	// FuncInnocuous. Calling it marks the connection (markEffects).
	effects bool
}

// funcs holds the functions SQLite knows by handle. SQLite gives a handle
// back to host_func_destroy when it drops the function.
var funcs table[funcEntry]

// aggs holds the states of the groups being aggregated. A group's handle
// is in its sqlite3_aggregate_context, 0 until its first row.
var aggs table[Agg]

// collations holds the comparisons SQLite knows by handle.
var collations table[func(a, b string) int]

// CreateFunction makes f the SQL function name of nArg arguments (-1 for any
// number) on the connection, in place of one of the same name and nArg.
func (c *Conn) CreateFunction(name string, nArg int, flags int, f Func) error {
	borrow := flags&FuncBorrowBlobs != 0
	return c.createFunc(name, nArg, flags&^FuncBorrowBlobs, funcEntry{fn: f, borrow: borrow}, go_create_function)
}

// CreateAggregate makes an aggregate function name: newAgg makes the state
// of each group.
func (c *Conn) CreateAggregate(name string, nArg int, flags int, newAgg func() Agg) error {
	return c.createFunc(name, nArg, flags, funcEntry{newAgg: newAgg}, go_create_aggregate)
}

// WindowAgg is an Agg that is also an aggregate window function: Value gives
// the result for the rows of the frame so far, and Inverse takes out a row
// that left the frame, with the arguments Step had for it.
type WindowAgg interface {
	Agg
	Value() (driver.Value, error)
	Inverse(args []driver.Value) error
}

// CreateWindow makes an aggregate window function name: newWin makes the
// state of each group or partition.
func (c *Conn) CreateWindow(name string, nArg int, flags int, newWin func() WindowAgg) error {
	newAgg := func() Agg {
		w := newWin()
		if w == nil {
			return nil
		}
		return w
	}
	return c.createFunc(name, nArg, flags, funcEntry{newAgg: newAgg}, go_create_window)
}

func (c *Conn) createFunc(name string, nArg, flags int, e funcEntry, create func(sp, db, name, nArg, flags, h uintptr) uintptr) error {
	if err := c.brokenErr(); err != nil {
		return err
	}
	if len(name) > maxName {
		return &Error{code: 21, msg: "function name too long"}
	}
	e.effects = flags&FuncInnocuous == 0
	h := funcs.add(&e)
	a := c.args()
	z := a.str(name)
	// SQLite gives h to host_func_destroy when this fails as well.
	return c.err(int(int32(create(a.top, c.db, z, uintptr(nArg), uintptr(flags|utf8), h))))
}

// CreateCollation makes cmp the collation name. cmp returns a negative
// number, 0 or a positive number as a sorts before, with or after b.
func (c *Conn) CreateCollation(name string, cmp func(a, b string) int) error {
	if err := c.brokenErr(); err != nil {
		return err
	}
	if len(name) > maxName {
		return &Error{code: 21, msg: "collation name too long"}
	}
	h := collations.add(&cmp)
	a := c.args()
	z := a.str(name)
	r := int32(go_create_collation(a.top, c.db, z, h))
	if r != OK {
		// SQLite drops a collation it refused without a word.
		collations.remove(h)
	}
	return c.err(int(r))
}

// host_func runs the Go function of a call. Everything it hands the C is
// written from sp, above the frames of the C that called it.
func host_func(sp, ctx, argc, argv uintptr) {
	defer failQuiet(sp)
	e := funcs.get(sqlite3_user_data(sp, ctx))
	if e.effects {
		markEffects(sp)
	}
	f := e.fn
	args := values(sp, argc, argv, e.borrow)
	v, err := safely(func() (driver.Value, error) { return f(args) })
	if err == nil {
		err = result(sp, ctx, v)
	}
	if err != nil {
		resultError(sp, ctx, err)
	}
}

// host_step runs Step of the group's state, made at its first row.
func host_step(sp, ctx, argc, argv uintptr) {
	defer failQuiet(sp)
	a, ok := groupState(sp, ctx)
	if !ok {
		return
	}
	args := values(sp, argc, argv, false)
	if _, err := safely(func() (driver.Value, error) { return nil, a.Step(args) }); err != nil {
		resultError(sp, ctx, err)
	}
}

// host_value runs Value of the state of a window function's group. A frame
// with no rows yet gets a state of its own, which host_final drops at the
// end of the partition.
func host_value(sp, ctx uintptr) {
	defer failQuiet(sp)
	a, ok := groupState(sp, ctx)
	if !ok {
		return
	}
	v, err := safely(a.(WindowAgg).Value)
	if err == nil {
		err = result(sp, ctx, v)
	}
	if err != nil {
		resultError(sp, ctx, err)
	}
}

// host_inverse runs Inverse of the state of a window function's group.
func host_inverse(sp, ctx, argc, argv uintptr) {
	defer failQuiet(sp)
	a, ok := groupState(sp, ctx)
	if !ok {
		return
	}
	args := values(sp, argc, argv, false)
	if _, err := safely(func() (driver.Value, error) { return nil, a.(WindowAgg).Inverse(args) }); err != nil {
		resultError(sp, ctx, err)
	}
}

// groupState is the state of the group of ctx, made at its first call. When
// it cannot give one it sets the error of ctx.
func groupState(sp, ctx uintptr) (Agg, bool) {
	p := sqlite3_aggregate_context(sp, ctx, 8)
	if p == 0 {
		sqlite3_result_error_nomem(sp, ctx)
		return nil, false
	}
	slot := (*uintptr)(at(p))
	if *slot == 0 {
		a, err := newState(sp, ctx)
		if err != nil {
			resultError(sp, ctx, err)
			return nil, false
		}
		*slot = aggs.add(&a)
	}
	return *aggs.get(*slot), true
}

// host_final runs Final of the group's state and drops it. SQLite calls it
// for every group that had a step, also when the statement stops early,
// and for a group of no rows, which gets a state of its own.
func host_final(sp, ctx uintptr) {
	defer failQuiet(sp)
	var a Agg
	if p := sqlite3_aggregate_context(sp, ctx, 0); p != 0 && *(*uintptr)(at(p)) != 0 {
		slot := (*uintptr)(at(p))
		a = *aggs.get(*slot)
		aggs.remove(*slot)
		*slot = 0
	} else {
		var err error
		if a, err = newState(sp, ctx); err != nil {
			resultError(sp, ctx, err)
			return
		}
	}
	v, err := safely(a.Final)
	if err == nil {
		err = result(sp, ctx, v)
	}
	if err != nil {
		resultError(sp, ctx, err)
	}
}

// newState makes the state of a group.
func newState(sp, ctx uintptr) (Agg, error) {
	e := funcs.get(sqlite3_user_data(sp, ctx))
	if e.effects {
		markEffects(sp) // every group of a call gets a state
	}
	newAgg := e.newAgg
	var a Agg
	_, err := safely(func() (driver.Value, error) { a = newAgg(); return nil, nil })
	if err == nil && a == nil {
		err = fmt.Errorf("the state of an aggregate is nil")
	}
	return a, err
}

func host_func_destroy(sp, h uintptr) {
	defer failQuiet(sp)
	funcs.remove(h)
}

// host_collate compares two strings with a Go collation. A comparison has no
// way to fail: a panic in one is kept on the stack of the connection, the
// comparison gives 0, and SQLite is told to stop the statement as it does
// when memory runs out (collatePanicked).
func host_collate(sp, h, n1, p1, n2, p2 uintptr) (ret uintptr) {
	defer failWith(sp, &ret, 0)
	cmp := *collations.get(h)
	r, err := compare(cmp, string(bytesAt(p1, int(int32(n1)))), string(bytesAt(p2, int(int32(n2)))))
	if err != nil {
		collatePanicked(sp, err)
		return 0
	}
	switch {
	case r < 0:
		return ^uintptr(0) // -1
	case r > 0:
		return 1
	}
	return 0
}

func host_collate_destroy(sp, h uintptr) {
	defer failQuiet(sp)
	collations.remove(h)
}

// values copies the arguments of a call into Go, but for BLOBs when borrow
// says the function reads them in place (FuncBorrowBlobs).
func values(sp, argc, argv uintptr, borrow bool) []driver.Value {
	args := make([]driver.Value, int(int32(argc)))
	for i := range args {
		args[i] = value(sp, *(*uintptr)(at(argv + uintptr(i)*8)), borrow)
	}
	return args
}

// safely runs user code and turns a panic in it into an error: the panic is
// the user's, and the C under the callback is still whole.
func safely(f func() (driver.Value, error)) (v driver.Value, err error) {
	defer func() {
		if e := recover(); e != nil {
			v, err = nil, fmt.Errorf("panic in a Go function: %v", e)
		}
	}()
	return f()
}

// SQLite's fundamental types (sqlite3.h).
const (
	typeInteger = 1
	typeFloat   = 2
	typeText    = 3
	typeBlob    = 4
	typeNull    = 5
)

// value copies a sqlite3_value into Go, but for a BLOB when borrow is set.
func value(sp, v uintptr, borrow bool) driver.Value {
	switch int32(sqlite3_value_type(sp, v)) {
	case typeInteger:
		return int64(sqlite3_value_int64(sp, v))
	case typeFloat:
		return sqlite3_value_double(sp, v)
	case typeText:
		p := sqlite3_value_text(sp, v)
		n := int(int32(sqlite3_value_bytes(sp, v)))
		if p == 0 || n == 0 {
			return ""
		}
		return string(bytesAt(p, n))
	case typeBlob:
		p := sqlite3_value_blob(sp, v)
		n := int(int32(sqlite3_value_bytes(sp, v)))
		if borrow && n > 0 {
			return bytesAt(p, n)
		}
		b := make([]byte, n)
		if n > 0 {
			copy(b, bytesAt(p, n))
		}
		return b
	}
	return nil
}

// result sets the result of a call to v.
func result(sp, ctx uintptr, v driver.Value) error {
	switch v := v.(type) {
	case nil:
		sqlite3_result_null(sp, ctx)
	case int64:
		sqlite3_result_int64(sp, ctx, uintptr(v))
	case int:
		sqlite3_result_int64(sp, ctx, uintptr(v))
	case bool:
		var i uintptr
		if v {
			i = 1
		}
		sqlite3_result_int64(sp, ctx, i)
	case float64:
		sqlite3_result_double(sp, ctx, v)
	case string:
		if len(v) > math.MaxInt32 {
			sqlite3_result_error_toobig(sp, ctx)
			return nil
		}
		z, done, err := stageAt(sp, unsafe.StringData(v), len(v))
		if err != nil {
			sqlite3_result_error_nomem(sp, ctx)
			return nil
		}
		defer done()
		sqlite3_result_text(sp+stageRoom(len(v)), ctx, z, uintptr(len(v)), transient)
	case []byte:
		if v == nil {
			sqlite3_result_null(sp, ctx)
			return nil
		}
		if len(v) > math.MaxInt32 {
			sqlite3_result_error_toobig(sp, ctx)
			return nil
		}
		z, done, err := stageAt(sp, unsafe.SliceData(v), len(v))
		if err != nil {
			sqlite3_result_error_nomem(sp, ctx)
			return nil
		}
		defer done()
		sqlite3_result_blob(sp+stageRoom(len(v)), ctx, z, uintptr(len(v)), transient)
	default:
		return fmt.Errorf("a Go function returned a %T", v)
	}
	return nil
}

// resultError makes msg the error of a call. SQLite copies it.
func resultError(sp, ctx uintptr, err error) {
	msg := errText(err)
	const most = 1 << 16
	if len(msg) > most {
		msg = msg[:most]
	}
	a := args{sp}
	z := a.str(msg)
	sqlite3_result_error(a.top, ctx, z, uintptr(len(msg)))
}

// compare runs a Go collation and turns a panic in it into an error.
func compare(cmp func(a, b string) int, a, b string) (r int, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()
	return cmp(a, b), nil
}

// collatePanicked keeps the first panic of a collation for the error of the
// call (Conn.err), and has SQLite stop the statement as if memory ran out.
func collatePanicked(sp uintptr, err error) {
	v, ok := stacks.Load(stackBase(sp))
	if !ok {
		panic("csqlite: a collation ran on a stack the host did not make")
	}
	s := v.(*stack)
	if s.collatePanic == "" {
		s.collatePanic = errText(err)
	}
	go_collate_failed(sp, s.db)
}

// errText is err.Error(), which is user code too.
func errText(err error) (s string) {
	defer func() {
		if e := recover(); e != nil {
			s = fmt.Sprintf("an error whose Error method panicked: %v", e)
		}
	}()
	return err.Error()
}
