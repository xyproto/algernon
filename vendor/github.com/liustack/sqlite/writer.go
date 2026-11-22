// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64))

package sqlite

import (
	"cmp"
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/liustack/sqlite/internal/csqlite"
)

// One database file has one writer inside the driver. A connection takes the
// file's write lock before a statement that may write, and gives it back once
// the connection is out of its transaction. Writers of one process wait for
// each other here, in order, and not in SQLite's busy handler, whose sleeps
// grow while the writer that holds the lock waits for a thread to run on.
// Readers never take it. Other processes still meet SQLite's own locks.
//
// Group commit: an autocommit statement that only changes the main database
// and depends on nothing of its connection (csqlite.Stmt.Mergeable) does not wait
// for the lock when another writer holds it. It joins the file's queue. The
// writer that gives the lock back runs the requests of the queue that its
// connection can run exactly as their own would (same key, neither
// connection changed since it opened: conn.canMerge) as one transaction,
// each statement in a SAVEPOINT of its own, and commits once. Each caller
// gets its own result after the commit. A request no writer can run that
// way gets the lock itself, in turn, and runs on its own connection. With no
// other writer a statement runs directly.
type writeLock struct {
	mu      sync.Mutex
	held    bool
	waiters []waiter   // writers that wait for the lock, in order
	queue   []*request // statements to merge, in order
	// seq counts the writers and statements that came to wait, so that the
	// lock goes to whichever of them came first.
	seq  uint64
	refs int         // connections of the file, and 1 while attached
	file os.FileInfo // the file, whatever name it is opened by
	// attached says the file is replicated (replhook.Attach). Guarded by
	// writeLocks.
	attached bool
	// committed is the replicator's, called as a writer gives the lock
	// back (replhook.Attach).
	committed atomic.Pointer[func(frames, salt uint32, limit int64)]
	stats     batchStats
	// walSkip is the length of the WAL in frames below which no connection
	// tries again to start it over, after a try that readers held off
	// (wallimit.go), and walPage the page size of the database read at the
	// last try, 0 before the first. Only the holder of the lock reads and
	// writes them.
	walSkip, walPage int64
	// walTries counts the tries to start the WAL over, and walRestarts
	// those that did.
	walTries, walRestarts atomic.Int64
	// walAt is the length of the WAL in frames at the last automatic
	// checkpoint the driver ran (checkpoint.go), and walSalt the salt of
	// the WAL then, which changes when the WAL starts over. Only the
	// holder of the lock reads and writes them. walCheckpoints counts
	// those checkpoints.
	walAt          int64
	walSalt        uint32
	walCheckpoints atomic.Int64
}

// batchStats counts what group commit does on one file, for the benchmarks
// that decide whether waiting a little before a batch would pay.
type batchStats struct {
	batches atomic.Int64 // batches taken from the queue
	full    atomic.Int64 // batches that took maxBatch statements
	commits atomic.Int64 // batches committed
	merged  atomic.Int64 // statements committed by those
	// commitTime is the time spent in the COMMIT of those, in nanoseconds.
	commitTime atomic.Int64
}

// A waiter is a writer that waits for the lock: ch is closed as the lock
// goes to it. seq is its place in line (writeLock.seq).
type waiter struct {
	ch  chan struct{}
	seq uint64
}

// A request is one statement of the queue.
type request struct {
	seq  uint64 // its place in line (writeLock.seq)
	ctx  context.Context
	key  string // the key of the caller's connection
	sql  string
	args []driver.NamedValue
	done chan struct{}
	// handed says the lock was handed to the caller, which runs the
	// statement itself.
	handed bool
	res    result
	// lastID is the caller's last_insert_rowid: what it was when the request
	// was made, then what the statement left once it ran, as it would on
	// the caller's connection. The connection running the batch starts the
	// statement from it, so no value has to stand for "nothing inserted".
	lastID int64
	err    error
	// effects says the statement called a Go function that may have side
	// effects (one not marked Innocuous): its batch commits after it.
	effects bool
	// state says which came first once the request left the queue: the
	// driver, which starts the statement or hands the lock to the caller
	// (reqTaken), or the caller, which leaves as its context ends (reqLeft).
	// A caller that left gets nothing more, and its statement never runs. A
	// caller that came second waits for its answer.
	state atomic.Int32
}

// The states of a request.
const (
	reqWaiting int32 = iota // in the queue, or in a batch before its turn
	reqTaken                // its statement started, or the lock was handed to its caller
	reqLeft                 // its caller left
)

// maxBatch is the most statements one transaction of the queue takes.
const maxBatch = 256

// writeLocks holds the write lock of each file open. A file is found by
// what it is, not by its name: one file has more than one name (case on a
// file system that ignores it, a short 8.3 name on windows).
var writeLocks struct {
	sync.Mutex
	all []*writeLock
}

// fileOf is the file named path, for writeLocks.
func fileOf(path string) (os.FileInfo, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	// On windows the identity of the file is read when first compared:
	// read it now, while path names it. A FileInfo whose identity could not
	// be read is the same as no file.
	if !os.SameFile(fi, fi) {
		return nil, fmt.Errorf("sqlite: %s: cannot read what file it is", path)
	}
	return fi, nil
}

// findWriteLock is the write lock of file, nil if none. writeLocks is held.
func findWriteLock(file os.FileInfo) *writeLock {
	for _, l := range writeLocks.all {
		if os.SameFile(l.file, file) {
			return l
		}
	}
	return nil
}

// openWriteLock is the write lock of path for a new connection, and whether
// the file is replicated.
func openWriteLock(path string) (l *writeLock, attached bool, err error) {
	file, err := fileOf(path)
	if err != nil {
		return nil, false, err
	}
	writeLocks.Lock()
	defer writeLocks.Unlock()
	if l = findWriteLock(file); l == nil {
		l = &writeLock{file: file}
		writeLocks.all = append(writeLocks.all, l)
	}
	l.refs++
	return l, l.attached, nil
}

func closeWriteLock(l *writeLock) {
	writeLocks.Lock()
	defer writeLocks.Unlock()
	l.refs--
	if l.refs == 0 {
		writeLocks.all = slices.DeleteFunc(writeLocks.all, func(x *writeLock) bool { return x == l })
	}
}

// errBusy is what a writer gets when the lock stays taken past the busy
// timeout, as SQLite would answer.
var errBusy = csqlite.NewError(5, "database is locked", nil)

// errForeignKey is what a merged statement that leaves a deferred foreign
// key unresolved gets: the error its own COMMIT would have given.
var errForeignKey = csqlite.NewError(787, "FOREIGN KEY constraint failed", nil)

// acquire takes the lock, waiting up to timeout.
func (l *writeLock) acquire(ctx context.Context, timeout time.Duration) error {
	l.mu.Lock()
	if !l.held {
		l.held = true
		l.mu.Unlock()
		return nil
	}
	if timeout <= 0 {
		l.mu.Unlock()
		return errBusy
	}
	ch := make(chan struct{})
	l.seq++
	l.waiters = append(l.waiters, waiter{ch, l.seq})
	l.mu.Unlock()
	t := time.NewTimer(timeout)
	defer t.Stop()
	var err error
	select {
	case <-ch:
		return nil
	case <-t.C:
		err = errBusy
	case <-ctx.Done():
		err = ctx.Err()
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, w := range l.waiters {
		if w.ch == ch {
			l.waiters = append(l.waiters[:i], l.waiters[i+1:]...)
			return err
		}
	}
	return nil // handed the lock as the wait ended
}

// release gives the lock back from c, nil for the replicator. c first runs
// one batch of the requests it can run. Then the lock goes to whichever came
// first of the first writer that waits for it and the first request of the
// queue. A request no batch takes, because no connection that runs one can
// run it, still gets the lock in its turn, however many writers keep coming.
// A request whose caller left is dropped there: no one would take the lock.
func (l *writeLock) release(c *conn) {
	l.mu.Lock()
	if c != nil && c.canMerge() {
		if batch := l.take(c, nil); len(batch) > 0 {
			l.mu.Unlock()
			back := c.runBatch(batch)
			l.mu.Lock()
			l.requeue(back)
			if !c.c.AutoCommit() {
				// The transaction of the batch would not roll back
				// (conn.abandon): c keeps the lock, and Close gives it back.
				c.writing = true
				l.mu.Unlock()
				return
			}
		}
	}
	// The WAL as it is now, which c knows once it has read it. A
	// connection that never did wrote nothing there.
	if frames, salt, ok := walNow(c); ok {
		if f := l.committed.Load(); f != nil {
			l.mu.Unlock()
			(*f)(frames, salt, c.walLimit)
			l.mu.Lock()
		}
		if c.walOver(l, int64(frames)) {
			l.mu.Unlock()
			c.restartWAL(l)
			l.mu.Lock()
		}
		// Also after the WAL limit looked: it may have found the WAL under
		// the limit, and past it a checkpoint that waited for readers in
		// vain leaves this one little or nothing to copy.
		if c.walDue(l, int64(frames), salt) {
			l.mu.Unlock()
			c.checkpoint(l, int64(frames))
			l.mu.Lock()
		}
	}
	for {
		switch {
		case len(l.waiters) > 0 && (len(l.queue) == 0 || l.waiters[0].seq < l.queue[0].seq):
			ch := l.waiters[0].ch
			l.waiters = l.waiters[1:]
			close(ch) // the lock goes to it, still held
		case len(l.queue) > 0:
			r := l.queue[0]
			l.queue = l.queue[1:]
			if !r.state.CompareAndSwap(reqWaiting, reqTaken) {
				close(r.done) // its caller left: it ends here
				continue
			}
			r.handed = true
			close(r.done)
		default:
			l.held = false
		}
		break
	}
	l.mu.Unlock()
}

// requeue puts back requests a batch could not run, in their place in line:
// the queue stays in the order the requests came. l.mu is held.
func (l *writeLock) requeue(back []*request) {
	if len(back) == 0 {
		return
	}
	l.queue = append(back, l.queue...)
	slices.SortStableFunc(l.queue, func(a, b *request) int { return cmp.Compare(a.seq, b.seq) })
}

// take moves the requests of the queue that c can run, after batch, into
// batch, up to maxBatch. l.mu is held.
func (l *writeLock) take(c *conn, batch []*request) []*request {
	n := 0
	for _, r := range l.queue {
		if r.key == c.key {
			n++
		}
	}
	if n = min(n, maxBatch-len(batch)); n > 0 {
		out := make([]*request, len(batch), len(batch)+n)
		copy(out, batch)
		rest := l.queue[:0] // in place: it writes behind where it reads
		for _, r := range l.queue {
			if r.key == c.key && len(out) < cap(out) {
				out = append(out, r)
			} else {
				rest = append(rest, r)
			}
		}
		clear(l.queue[len(rest):])
		l.queue = rest
		batch = out
	}
	if len(batch) > 0 {
		l.stats.batches.Add(1)
		if len(batch) == maxBatch {
			l.stats.full.Add(1)
		}
	}
	return batch
}

// lead runs req, whose connection c has just been handed the lock, as the
// first of a batch with the requests of the queue c can run: a request
// handed the lock does not take a commit of its own. It reports false when
// c cannot run req that way, which then runs alone.
func (l *writeLock) lead(c *conn, req *request) bool {
	req.handed = false
	req.done = make(chan struct{}) // runBatch closes it again
	req.state.Store(reqWaiting)    // and starts it
	l.mu.Lock()
	batch := l.take(c, []*request{req})
	l.mu.Unlock()
	back := c.runBatch(batch)
	ran := true
	for i, r := range back {
		if r == req {
			back = append(back[:i:i], back[i+1:]...)
			ran = false
			break
		}
	}
	if len(back) > 0 {
		l.mu.Lock()
		l.requeue(back)
		l.mu.Unlock()
	}
	return ran
}

// submit runs a mergeable statement: directly when the lock is free, in the
// queue when not.
func (l *writeLock) submit(ctx context.Context, c *conn, s *stmt, args []driver.NamedValue) (*result, error) {
	l.mu.Lock()
	if !l.held {
		l.held = true
		l.mu.Unlock()
		return c.runHeld(ctx, s, args)
	}
	if c.busyTimeout <= 0 {
		l.mu.Unlock()
		return nil, errBusy
	}
	l.seq++
	req := &request{ctx: ctx, key: c.key, sql: s.sql, args: args, done: make(chan struct{}), lastID: c.c.LastInsertRowID(), seq: l.seq}
	l.queue = append(l.queue, req)
	l.mu.Unlock()
	t := time.NewTimer(c.busyTimeout)
	defer t.Stop()
	var err error
	ended := false // the context
	select {
	case <-req.done:
		return c.answered(ctx, s, req)
	case <-t.C:
		err = errBusy
	case <-ctx.Done():
		err, ended = ctx.Err(), true
	}
	l.mu.Lock()
	for i, q := range l.queue {
		if q == req {
			l.queue = append(l.queue[:i], l.queue[i+1:]...)
			l.mu.Unlock()
			return nil, err
		}
	}
	l.mu.Unlock()
	// Taken already: a batch has it, or it has the lock. The busy timeout
	// no longer counts, as the wait for the lock is over. The context does
	// until the statement starts: the caller then leaves, and the statement
	// never runs. Once it started, the caller waits for what became of it.
	if !ended {
		select {
		case <-req.done:
			return c.answered(ctx, s, req)
		case <-ctx.Done():
		}
	}
	if req.state.CompareAndSwap(reqWaiting, reqLeft) {
		return nil, ctx.Err()
	}
	<-req.done
	return c.answered(ctx, s, req)
}

// runHeld runs s on c, which has just been given the write lock, and gives
// the lock back.
func (c *conn) runHeld(ctx context.Context, s *stmt, args []driver.NamedValue) (*result, error) {
	c.writing = true
	r, err := s.run(ctx, args)
	c.endWrite()
	return r, err
}

// answered is the result of a request of c once it is done.
func (c *conn) answered(ctx context.Context, s *stmt, req *request) (*result, error) {
	if req.handed {
		if err := ctx.Err(); err != nil {
			c.writing = true
			c.endWrite()
			return nil, err
		}
		if !c.canMerge() || !c.wl.lead(c, req) {
			if !c.c.AutoCommit() {
				// The batch c led would not roll back (conn.abandon). req
				// never ran and must not run in that transaction: c keeps
				// the lock until it is closed, and database/sql runs req on
				// another connection.
				c.writing = true
				return nil, driver.ErrBadConn
			}
			if c.retired() {
				// The batch c led left c spent, and req never ran: it
				// gives the lock back, and database/sql runs req on
				// another connection.
				c.writing = true
				c.endWrite()
				return nil, driver.ErrBadConn
			}
			return c.runHeld(ctx, s, req.args)
		}
		c.writing = true
		c.endWrite()
	}
	c.c.SetLastInsertRowID(req.lastID)
	if req.err != nil {
		return nil, req.err
	}
	r := &req.res // req is done with: its result is the caller's
	r.id = req.lastID
	return r, nil
}

// canMerge says c may run statements of other connections of its key: it
// runs them exactly as theirs would, since no connection of the key that
// changed itself (conn.private) ever joins the queue.
func (c *conn) canMerge() bool {
	return !c.private() && !c.closed && !c.bad && !c.readOnlyTx && !c.c.Broken() && !c.c.Spent() && c.c.AutoCommit() && !c.c.InTx()
}

// beginWrite takes the write lock before a statement that may write. A
// connection in a read transaction does not wait for it, as SQLite does not
// call the busy handler then: in rollback journal mode the writer that
// holds the lock waits for this connection's read lock to go before it
// commits, so both would wait until the timeout. In WAL mode the writer's
// commit leaves this connection's snapshot old, so the write would fail
// once the wait ended.
func (c *conn) beginWrite(ctx context.Context, readOnly bool) error {
	if c.wl == nil || c.writing || readOnly || c.readOnlyTx {
		return nil
	}
	timeout := c.busyTimeout
	if c.c.InTx() {
		timeout = 0
	}
	if err := c.wl.acquire(ctx, timeout); err != nil {
		return err
	}
	c.writing = true
	return nil
}

// endWrite gives the lock back once the connection is out of its
// transaction.
func (c *conn) endWrite() {
	if c.writing && c.c.AutoCommit() {
		c.writing = false
		c.wl.release(c)
	}
}

// runBatch runs requests of the queue as one transaction on c, which holds
// the lock and is out of any transaction, and answers each after the
// commit. It returns the requests c turned out not to be able to run, and
// those it did not get to, for the queue.
//
// No statement runs twice: what a statement did beside the database (a Go
// function, a virtual table) cannot be undone. A statement that fails keeps
// what SQLite kept of it, as its own autocommit transaction would (nothing,
// or the rows before the failure for ON CONFLICT FAIL), and its caller gets
// the error after the commit. One that leaves a deferred foreign key
// unresolved is undone by its SAVEPOINT, as its own COMMIT would fail. When
// a failure ends the whole transaction (an IO error, an interrupt, ON
// CONFLICT ROLLBACK), the statements that had run are gone with it.
func (c *conn) runBatch(batch []*request) (back []*request) {
	// The statements set c's last_insert_rowid, which is c's own.
	own := c.c.LastInsertRowID()
	defer c.c.SetLastInsertRowID(own)
	live := batch[:0] // in place: batch is runBatch's own
	for _, r := range batch {
		if err := r.ctx.Err(); err != nil {
			r.err = err
			close(r.done)
			continue
		}
		live = append(live, r)
	}
	batch = live
	if len(batch) == 0 {
		return nil
	}
	if err := c.execCached("BEGIN IMMEDIATE"); err != nil {
		answer(batch, err)
		return nil
	}
	// ran holds the statements that ran, those that failed too: one that
	// failed may have kept rows (OR FAIL), which commit with the others, so
	// it is answered after the commit.
	ran := make([]*request, 0, len(batch))
	for i, r := range batch {
		out, cause := c.runOne(r)
		switch out {
		case skipped:
			close(r.done)
			continue
		case notHere:
			back = append(back, r)
			continue
		case ended:
			// r gets what ended the transaction, and those that ran before
			// it get SQLITE_ABORT_ROLLBACK, as SQLite gives the statements
			// of a transaction it rolled back. The statements after r never
			// reached c: they wait for the next batch.
			close(r.done)
			answer(ran, rolledBack(cause))
			return append(back, batch[i+1:]...)
		}
		ran = append(ran, r)
		if r.effects {
			// Its Go functions did something outside the database: commit
			// now, so that no later statement can roll it back. The rest
			// wait for the next batch.
			back = append(back, batch[i+1:]...)
			break
		}
	}
	start := time.Now()
	if err := c.execCached("COMMIT"); err != nil {
		answer(ran, c.abandon(err))
		return back
	}
	c.wl.stats.commits.Add(1)
	c.wl.stats.commitTime.Add(int64(time.Since(start)))
	var merged int64
	for _, r := range ran {
		if r.err == nil {
			merged++
		}
	}
	c.wl.stats.merged.Add(merged)
	answer(ran, nil)
	return back
}

// rolledBack is the error of a statement of a batch whose transaction cause
// ended after the statement ran.
func rolledBack(cause error) error {
	return csqlite.NewError(ABORT_ROLLBACK, "abort due to ROLLBACK: "+cause.Error(), cause)
}

// errNotHere is what cached gives for SQL that is not exactly one statement,
// which a request never is: it then runs on its caller's connection.
var errNotHere = errors.New("sqlite: not mergeable here")

// An outcome is what runOne did with a request.
type outcome int

const (
	// skipped: the statement never stepped, and the transaction is still
	// open. Its context ended, or its prepare or its bind failed: r.err
	// says which. It is answered alone.
	skipped outcome = iota
	// notHere: c does not run it as the caller's connection would. It did
	// nothing, and goes back to the queue.
	notHere
	// ran: it ran, and the transaction is still open. r.err is its own
	// error, or nil.
	ran
	// ended: the transaction is over. The statement, or a statement of the
	// batch around it, ended it with r.err.
	ended
)

// runOne runs one statement of a batch in a SAVEPOINT. When the transaction
// ended it also gives what ended it, as SQLite said it: an interrupt is not
// the error of the context of the others.
func (c *conn) runOne(r *request) (outcome, error) {
	r.effects = false
	if r.err = r.ctx.Err(); r.err != nil {
		return skipped, nil
	}
	s, err := c.cached(r.sql)
	if errors.Is(err, errNotHere) {
		return notHere, nil
	}
	if err != nil {
		r.err = err
		if c.c.AutoCommit() {
			// The prepare ended the transaction. SQLite runs statements of
			// its own while it prepares one (it reads the schema, and a
			// virtual table reads its tables as it is connected), and one
			// of those that runs out of memory rolls the transaction back.
			// Going on would run the rest of the batch with no transaction
			// around it, each statement committed by its RELEASE.
			return ended, err
		}
		return skipped, nil
	}
	if !s.s.Mergeable() {
		return notHere, nil
	}
	// From here the statement has started: its caller no longer leaves. The
	// arguments of a caller that left are not the request's any more.
	if !r.state.CompareAndSwap(reqWaiting, reqTaken) {
		r.err = context.Canceled // its caller left as its context ended
		return skipped, nil
	}
	if r.err = s.bind(r.args); r.err != nil {
		return skipped, nil
	}
	if err := c.execCached("SAVEPOINT merge"); err != nil {
		r.err = c.abandon(err)
		return ended, r.err
	}
	c.c.SetLastInsertRowID(r.lastID)
	c.c.SetMerging(true)
	res, stepped, err := s.step(r.ctx)
	r.effects = c.c.RanEffects()
	if c.c.SetMerging(false) {
		// SQLite prepared the statement again and it is no longer
		// mergeable: its prepare failed, so it did nothing, also when the
		// refusal fell on a statement a virtual table prepared meanwhile
		// (goClassify). It runs on its caller's connection.
		s.s.Reclassify()
		if e := c.execCached("ROLLBACK TO merge"); e != nil {
			r.err = c.abandon(e)
			return ended, r.err
		}
		if e := c.execCached("RELEASE merge"); e != nil {
			r.err = c.abandon(e)
			return ended, r.err
		}
		// It waits again: it did nothing, and its caller still waits.
		r.state.Store(reqWaiting)
		return notHere, nil
	}
	if !stepped {
		// Its context ended after its turn came.
		if e := c.execCached("RELEASE merge"); e != nil {
			r.err = c.abandon(e)
			return ended, r.err
		}
		r.err = err
		return skipped, nil
	}
	// SQLite keeps last_insert_rowid when it undoes a statement, so this is
	// the caller's whatever becomes of the statement.
	r.lastID = c.c.LastInsertRowID()
	if c.c.AutoCommit() {
		if err == nil {
			err = errBusy // cannot happen: a statement that succeeds keeps the transaction
		}
		r.err = ctxErr(r.ctx, err)
		return ended, err
	}
	// A statement that failed may have kept rows too (OR FAIL). SQLite
	// rolls a statement it undoes back to its count of deferred foreign
	// keys, so a count left here is this statement's.
	undo := false
	if c.c.DeferredFKs() {
		err, undo = errForeignKey, true
	}
	if c.c.VirtualTablesInTx() {
		// FTS5 writes the terms of an INSERT to its index only at the next
		// SAVEPOINT or at COMMIT, so a failure to write them (a broken
		// index, a full disk) would fall on the next statement of the batch
		// or on all of them. A SAVEPOINT of its own writes them now, inside
		// the statement's SAVEPOINT, also after the statement failed, which
		// may leave terms. When it fails, the statement is undone: alone, its
		// COMMIT would have failed.
		if ferr := c.flushVirtualTables(); ferr != nil {
			if c.c.AutoCommit() {
				r.err = ferr // the failure ended the transaction
				return ended, ferr
			}
			if err == nil {
				err = ferr
			}
			undo = true
		}
	}
	if undo {
		if e := c.execCached("ROLLBACK TO merge"); e != nil {
			r.err = c.abandon(e)
			return ended, r.err
		}
	}
	if e := c.execCached("RELEASE merge"); e != nil {
		r.err = c.abandon(e)
		return ended, r.err
	}
	r.err = ctxErr(r.ctx, err)
	if err == nil {
		r.res = *res
	}
	return ran, nil
}

// flushVirtualTables makes the virtual tables of the transaction write what
// they hold back: SQLite calls their xSavepoint as a SAVEPOINT begins.
func (c *conn) flushVirtualTables() error {
	if err := c.execCached("SAVEPOINT flush"); err != nil {
		return err
	}
	return c.execCached("RELEASE flush")
}

// abandon rolls the transaction of a batch back when a statement the batch
// runs itself (SAVEPOINT, ROLLBACK TO, RELEASE, COMMIT) fails. A transaction
// that will not roll back makes c bad: c keeps the lock until it is closed,
// which rolls it back (writeLock.release).
func (c *conn) abandon(err error) error {
	if !c.c.AutoCommit() {
		c.execCached("ROLLBACK")
	}
	if !c.c.AutoCommit() {
		c.bad = true
	}
	return err
}

// answer gives each request its result, with err for all when it is not nil.
func answer(reqs []*request, err error) {
	for _, r := range reqs {
		if err != nil {
			r.err = err
		}
		close(r.done)
	}
}

// cached is a statement of the connection's cache for the queue's SQL.
func (c *conn) cached(sql string) (*stmt, error) {
	if s := c.cache[sql]; s != nil {
		return s, nil
	}
	p, tail, err := c.c.Prepare(sql)
	if err != nil {
		return nil, err
	}
	if p == nil || tail != "" {
		if p != nil {
			p.Finalize()
		}
		return nil, errNotHere // a request holds exactly one statement
	}
	if c.cache == nil {
		c.cache = map[string]*stmt{}
	}
	if len(c.cache) >= maxCached {
		for k, s := range c.cache {
			s.Close()
			delete(c.cache, k)
			break
		}
	}
	s := newStmt(c, p, sql, false)
	c.cache[sql] = s
	return s, nil
}

// controlFault is nil but in tests (fault_test.go), which set it to make a
// statement of execCached fail as one could: a ROLLBACK the connection could
// not prepare for want of memory leaves the transaction open.
var controlFault func(c *conn, sql string) error

// maxCached is the most statements a connection keeps for the queue.
const maxCached = 64

// execCached runs a statement of a batch with no parameters, such as
// SAVEPOINT merge. These few are kept apart from the statements of the
// requests and never closed while c is open: making room in c.cache for a
// request then never closes a statement in use, such as the request's own
// before its SAVEPOINT.
func (c *conn) execCached(sql string) error {
	if controlFault != nil {
		if err := controlFault(c, sql); err != nil {
			return err
		}
	}
	s := c.control[sql]
	if s == nil {
		p, tail, err := c.c.Prepare(sql)
		if err != nil {
			return err
		}
		if p == nil || tail != "" {
			if p != nil {
				p.Finalize()
			}
			return fmt.Errorf("sqlite: %q is not one statement", sql)
		}
		if c.control == nil {
			c.control = map[string]*stmt{}
		}
		s = newStmt(c, p, sql, false)
		c.control[sql] = s
	}
	_, err := s.s.Step()
	if rerr := s.s.Reset(); err == nil {
		err = rerr
	}
	return err
}
