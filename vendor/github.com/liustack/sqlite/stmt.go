// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64))

package sqlite

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/liustack/sqlite/internal/csqlite"
)

type stmt struct {
	c       *conn
	s       *csqlite.Stmt
	sql     string
	nparams int
	// readOnly says the statement writes nothing (sqlite3_stmt_readonly).
	readOnly bool
	// changesConn says the statement changes its connection (a PRAGMA that
	// sets something, ATTACH, a temporary object).
	changesConn bool
	// explain says the statement is an EXPLAIN: see prepareAgain.
	explain bool
	// once is a statement of conn.QueryContext: its rows finalize it.
	once bool
	// kept is the query a statement of conn.QueryContext is cached for, used
	// when it last ran, and keptAt its Reprepares when it was kept (cache.go).
	kept   string
	used   uint64
	keptAt int
	// busy says rows of the statement are open.
	busy bool
	// cols, decl and isTime describe the columns of its rows. They are read
	// again when SQLite has prepared the statement again after a change of
	// the schema (prepared counts those).
	cols     []string
	decl     []string
	isTime   []bool
	prepared int
}

// columns reads the names and declared types of the columns, once for each
// time SQLite prepared the statement.
func (s *stmt) columns() {
	p := s.s.Reprepares()
	if s.cols != nil && p == s.prepared {
		return
	}
	s.prepared = p
	n := s.s.ColumnCount()
	s.cols = make([]string, n)
	s.decl = make([]string, n)
	s.isTime = make([]bool, n)
	for i := range n {
		s.cols[i] = s.s.ColumnName(i)
		s.decl[i] = strings.ToUpper(s.s.ColumnDeclType(i))
		switch s.decl[i] {
		case "DATE", "DATETIME", "TIMESTAMP":
			s.isTime[i] = true
		}
	}
}

var (
	_ driver.Stmt             = (*stmt)(nil)
	_ driver.StmtExecContext  = (*stmt)(nil)
	_ driver.StmtQueryContext = (*stmt)(nil)
)

func (s *stmt) Close() error {
	if s.s == nil {
		return nil
	}
	err := s.s.Finalize()
	s.s = nil
	return err
}

func (s *stmt) NumInput() int { return -1 }

func (s *stmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.ExecContext(context.Background(), named(args))
}

func (s *stmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.QueryContext(context.Background(), named(args))
}

func named(args []driver.Value) []driver.NamedValue {
	n := make([]driver.NamedValue, len(args))
	for i, v := range args {
		n[i] = driver.NamedValue{Ordinal: i + 1, Value: v}
	}
	return n
}

func (s *stmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	return s.exec(ctx, args)
}

func (s *stmt) exec(ctx context.Context, args []driver.NamedValue) (*result, error) {
	c := s.c
	if c.retired() {
		return nil, driver.ErrBadConn
	}
	if err := c.txEnded(); err != nil {
		return nil, err
	}
	if err := s.prepareAgain(); err != nil {
		return nil, err
	}
	// A connection with a query not done keeps the snapshot it read, which the
	// connection running a batch would not have.
	if c.wl != nil && !c.writing && !c.private() && c.c.AutoCommit() && !c.c.InTx() && s.s.Mergeable() {
		return c.wl.submit(ctx, c, s, args)
	}
	if err := c.beginWrite(ctx, s.readOnly); err != nil {
		return nil, err
	}
	r, err := s.run(ctx, args)
	c.endWrite()
	if s.changesConn {
		// DETACH changes the connection too: settings are read after it.
		if aerr := c.syncAttached(); err == nil {
			err = aerr
		}
		c.changed()
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

// prepareAgain prepares an EXPLAIN again before it runs. SQLite runs EXPLAIN
// without checking the schema, so one prepared before a change of the
// schema would list the plan of the old schema, and the names of tables it
// freed.
func (s *stmt) prepareAgain() error {
	if !s.explain {
		return nil
	}
	if s.busy {
		return errors.New("sqlite: the rows of the statement are still open")
	}
	p, _, err := s.c.prepareOne(s.sql)
	if err != nil {
		return err
	}
	s.s.Finalize()
	s.s, s.cols = p, nil
	return nil
}

// changed reads what the driver keeps of the settings of c, once c is set
// up and after each statement that changes the connection, failed or not:
// busy_timeout, as the file's write lock waits as long as SQLite would, and
// who runs the automatic checkpoints (checkpoint.go). It reads fields of
// the connection and runs no statement, which SQLite may refuse after one
// that failed.
func (c *conn) changed() {
	if ms, ok := c.c.BusyTimeoutNow(); ok {
		c.busyTimeout = time.Duration(ms) * time.Millisecond
	}
	c.takeCheckpoints()
}

// run binds args and steps the statement once, with the write lock held if
// it writes. A context done before the step stops it: SQLite forgets an
// interrupt that comes before a statement starts.
func (s *stmt) run(ctx context.Context, args []driver.NamedValue) (*result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := s.bind(args); err != nil {
		return nil, err
	}
	res, _, err := s.step(ctx)
	return res, ctxErr(ctx, err)
}

// step steps the bound statement once and resets it. stepped says it did
// step: it does not when ctx is done already (watch). The error is SQLite's,
// as an interrupt: ctxErr makes it the error of ctx.
func (s *stmt) step(ctx context.Context) (res *result, stepped bool, err error) {
	stop, err := s.c.watch(ctx)
	if err != nil {
		return nil, false, err
	}
	_, err = s.s.Step()
	stop()
	if rerr := s.s.Reset(); err == nil {
		err = rerr
	}
	if err != nil {
		return nil, true, err
	}
	return &result{id: s.c.c.LastInsertRowID(), n: s.c.c.Changes()}, true, nil
}

func (s *stmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	return s.query(ctx, args)
}

// query runs the statement to its first row, so that its columns are the
// ones of the schema it runs on (SQLite prepares it again at its first step
// after a change of the schema), and a context done before then stops it.
func (s *stmt) query(ctx context.Context, args []driver.NamedValue) (*rows, error) {
	c := s.c
	if c.retired() {
		return nil, driver.ErrBadConn
	}
	if err := c.txEnded(); err != nil {
		return nil, err
	}
	if s.busy {
		return nil, errors.New("sqlite: the rows of the statement are still open")
	}
	if err := s.prepareAgain(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := s.bind(args); err != nil {
		return nil, err
	}
	if err := c.beginWrite(ctx, s.readOnly); err != nil {
		return nil, err
	}
	stop, err := c.watch(ctx)
	if err != nil {
		c.endWrite()
		return nil, err
	}
	row, err := s.s.Step()
	// Also when it failed: SQLite prepares a PRAGMA again as it steps it,
	// and sets the value as it does.
	if s.changesConn {
		if err != nil || !row {
			// An ATTACH has no row: it is done, and DETACH may run.
			s.s.Reset()
			if aerr := c.syncAttached(); err == nil {
				err = aerr
			}
		}
		c.changed()
	}
	if err != nil {
		stop()
		s.s.Reset()
		c.endWrite()
		return nil, ctxErr(ctx, err)
	}
	s.busy = true
	s.columns()
	r := &rows{s: s, ctx: ctx, stop: stop, first: true, done: !row}
	if !row {
		r.end()
	}
	return r, nil
}

func (s *stmt) bind(args []driver.NamedValue) error {
	if err := s.s.ClearBindings(); err != nil {
		return err
	}
	named := false
	for _, a := range args {
		named = named || a.Name != ""
	}
	if !named && len(args) < s.nparams {
		// SQLite would read the parameters left over as NULL.
		return fmt.Errorf("sqlite: %d arguments for %d parameters", len(args), s.nparams)
	}
	// bound marks what the arguments bind when some have names: SQLite would
	// read a parameter left out as NULL.
	var small [64]bool
	var bound []bool
	if named {
		bound = small[:]
		if s.nparams > len(small) {
			bound = make([]bool, s.nparams)
		}
	}
	for _, a := range args {
		i := a.Ordinal
		if a.Name != "" {
			i = s.paramIndex(a.Name)
			if i == 0 {
				return fmt.Errorf("sqlite: no parameter named %s", a.Name)
			}
		} else if i > s.nparams {
			return fmt.Errorf("sqlite: %d arguments for %d parameters", len(args), s.nparams)
		}
		var err error
		switch v := a.Value.(type) {
		case nil:
			err = s.s.BindNull(i)
		case int64:
			err = s.s.BindInt64(i, v)
		case float64:
			err = s.s.BindDouble(i, v)
		case bool:
			var n int64
			if v {
				n = 1
			}
			err = s.s.BindInt64(i, n)
		case []byte:
			if v == nil {
				err = s.s.BindNull(i) // as mattn/go-sqlite3 does
			} else {
				err = s.s.BindBlob(i, v)
			}
		case string:
			err = s.s.BindText(i, v)
		case time.Time:
			err = s.s.BindText(i, v.Format(timeFormat))
		default:
			return fmt.Errorf("sqlite: cannot bind %T", v)
		}
		if err != nil {
			return err
		}
		if bound != nil {
			bound[i-1] = true
		}
	}
	if bound != nil {
		return s.allBound(bound)
	}
	return nil
}

// allBound says each parameter has an argument, given what bound marks. One
// name used twice in the SQL is one parameter. A number up to the largest
// needs an argument even when the SQL skips it (1 in "SELECT ?2, :x"), as
// it does with no names.
func (s *stmt) allBound(bound []bool) error {
	for i := range s.nparams {
		if bound[i] {
			continue
		}
		if n := s.s.ParamName(i + 1); n != "" {
			return fmt.Errorf("sqlite: no argument for parameter %s", n)
		}
		return fmt.Errorf("sqlite: no argument for parameter %d", i+1)
	}
	return nil
}

// paramIndex finds the parameter of a name given without its prefix.
func (s *stmt) paramIndex(name string) int {
	for i := 1; i <= s.nparams; i++ {
		n := s.s.ParamName(i)
		if len(n) > 1 && n[1:] == name {
			return i
		}
	}
	return 0
}

// watch interrupts the statement of the connection when ctx is done, until
// stop is called. stop waits for an interrupt that has started, so that it
// never reaches a connection that is closed or runs another statement.
// When ctx is already done it gives the error of ctx, and the statement must
// not step: SQLite forgets an interrupt that comes before a statement starts.
// A cancel that comes while the step starts may still let the statement
// finish, as sqlite3_interrupt says.
func (c *conn) watch(ctx context.Context) (func(), error) {
	if ctx.Done() == nil {
		return func() {}, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	done := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		c.c.Interrupt()
		close(done)
	})
	return func() {
		if !stop() {
			<-done
		}
	}, nil
}

// ctxErr is the error of the context when it ended the statement.
func ctxErr(ctx context.Context, err error) error {
	var e *csqlite.Error
	if errors.As(err, &e) && e.Code() == 9 && ctx.Err() != nil { // SQLITE_INTERRUPT
		return ctx.Err()
	}
	return err
}

type result struct{ id, n int64 }

func (r *result) LastInsertId() (int64, error) { return r.id, nil }
func (r *result) RowsAffected() (int64, error) { return r.n, nil }

type rows struct {
	s    *stmt
	ctx  context.Context
	stop func()
	// first says query stepped to the first row and NextRow has not yet
	// given it.
	first bool
	done  bool
	close bool
}

// end stops watching the context once the statement has ended, so that no
// interrupt reaches a later statement of the connection.
func (r *rows) end() {
	r.done = true
	r.stop()
	r.stop = func() {}
}

var (
	_ driver.Rows                           = (*rows)(nil)
	_ driver.RowsColumnTypeDatabaseTypeName = (*rows)(nil)
)

// Columns gives a copy: database/sql hands the slice to its caller, and the
// statement keeps its own for its next rows.
func (r *rows) Columns() []string { return slices.Clone(r.s.cols) }

func (r *rows) ColumnTypeDatabaseTypeName(i int) string { return r.s.decl[i] }

func (r *rows) Close() error {
	if r.close {
		return nil
	}
	r.close = true
	r.stop()
	r.s.busy = false
	err := r.s.s.Reset()
	r.s.c.endWrite()
	if r.s.once {
		if ferr := r.s.Close(); err == nil {
			err = ferr
		}
	} else if r.s.kept != "" {
		if kerr := r.s.c.keepQuery(r.s, err); err == nil {
			err = kerr
		}
	}
	if r.done {
		return nil // Next already gave the error of the statement
	}
	return err
}

// NextRow steps to the next row.
func (r *rows) NextRow() error {
	if r.first {
		r.first = false
		if r.done {
			return io.EOF
		}
		return nil
	}
	if r.done {
		return io.EOF
	}
	row, err := r.s.s.Step()
	if err != nil {
		r.end()
		return ctxErr(r.ctx, err)
	}
	if !row {
		r.end()
		return io.EOF
	}
	return nil
}

// Next is NextRow and the values of the row, for database/sql before Go
// 1.27 calls ScanColumn.
func (r *rows) Next(dest []driver.Value) error {
	if err := r.NextRow(); err != nil {
		return err
	}
	for i := range dest {
		v, err := r.value(i)
		if err != nil {
			return err
		}
		dest[i] = v
	}
	return nil
}

// value is column i of the row as a driver.Value. It fails when SQLite has
// no memory to give a text or a blob.
func (r *rows) value(i int) (driver.Value, error) {
	s := r.s.s
	switch s.ColumnType(i) {
	case csqlite.TypeInteger:
		v := s.ColumnInt64(i)
		if r.isTime(i) {
			return time.Unix(v, 0).UTC(), nil
		}
		return v, nil
	case csqlite.TypeFloat:
		return s.ColumnDouble(i), nil
	case csqlite.TypeText:
		v, err := s.ColumnText(i)
		if err != nil {
			return nil, err
		}
		if r.isTime(i) {
			if t, ok := parseTime(v); ok {
				return t, nil
			}
		}
		return v, nil
	case csqlite.TypeBlob:
		b, err := s.ColumnBlob(i)
		if err != nil {
			return nil, err
		}
		if b == nil {
			b = []byte{}
		}
		return b, nil
	}
	return nil, nil
}

func (r *rows) isTime(i int) bool { return r.s.isTime[i] }
