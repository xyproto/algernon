// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64))

package sqlite

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/liustack/sqlite/internal/replhook"
)

func init() {
	replhook.Attach = attach
	replhook.Lock = lockFile
	replhook.TryLock = tryLockFile
}

// attach is replhook.Attach.
func attach(path string, committed func(frames, salt uint32, limit int64)) (detach func(), err error) {
	file, err := fileOf(path)
	if err != nil {
		return nil, err
	}
	writeLocks.Lock()
	defer writeLocks.Unlock()
	switch l := findWriteLock(file); {
	case l != nil && l.attached:
		return nil, errors.New("sqlite: " + path + " is replicated already")
	case l != nil:
		return nil, errors.New("sqlite: " + path + " is open: replication starts before the database is opened")
	}
	l := &writeLock{refs: 1, attached: true, file: file}
	if committed != nil {
		l.committed.Store(&committed)
	}
	writeLocks.all = append(writeLocks.all, l)
	return func() {
		writeLocks.Lock()
		l.attached = false
		l.committed.Store(nil)
		writeLocks.Unlock()
		closeWriteLock(l)
	}, nil
}

// attachment is a file attached to a connection (ATTACH), and the reference
// to its write lock that makes the file open to replhook.Attach.
type attachment struct {
	schema, name string
	l            *writeLock
}

// syncAttached brings c.attached in line with the files attached to c,
// after each statement that changes the connection, failed or not. A file
// attached anew takes a reference to its write lock, so that it cannot be
// replicated while attached, and a file detached gives its reference back.
//
// A replicated file is detached again, and the statement fails: writes to
// it from c would take neither its write lock nor the replicator's
// committed call, and would checkpoint it, so the replica would miss them.
// That covers a connection of the file itself, which writes to the file
// under the attached name through a pager of its own.
func (c *conn) syncAttached() error {
	// A broken connection runs nothing more, and keeps its references
	// until it closes.
	if c.c.Broken() || len(c.attached) == 0 && !c.c.Attached() {
		return nil
	}
	schemas, names := c.c.Attachments()
	kept := make([]attachment, 0, len(schemas))
	stays := make([]bool, len(c.attached))
	var add []int // indexes in schemas
	for i := range schemas {
		if j := slices.IndexFunc(c.attached, func(a attachment) bool { return a.schema == schemas[i] && a.name == names[i] }); j >= 0 {
			stays[j] = true
			kept = append(kept, c.attached[j])
		} else {
			add = append(add, i)
		}
	}
	var drop []attachment
	for j, a := range c.attached {
		if !stays[j] {
			drop = append(drop, a)
		}
	}
	var errs []error
	var detach []string
	for _, i := range add {
		// ATTACH opened the file a moment ago.
		file, err := fileOf(names[i])
		if err == nil {
			writeLocks.Lock()
			l := findWriteLock(file)
			switch {
			case l != nil && l.attached:
				err = fmt.Errorf("sqlite: %s is replicated: only a connection that opens it can write to it, ATTACH cannot", names[i])
			case l == nil:
				l = &writeLock{file: file}
				writeLocks.all = append(writeLocks.all, l)
				fallthrough
			default:
				l.refs++
				kept = append(kept, attachment{schemas[i], names[i], l})
			}
			writeLocks.Unlock()
		}
		if err != nil {
			errs = append(errs, err)
			detach = append(detach, schemas[i])
		}
	}
	for _, a := range drop {
		closeWriteLock(a.l)
	}
	c.attached = kept
	for _, schema := range detach {
		if err := c.c.Exec(`DETACH "` + strings.ReplaceAll(schema, `"`, `""`) + `"`); err != nil {
			// Right after ATTACH, which only read the schema of the file,
			// only a failure of SQLite itself (memory, I/O) stops DETACH,
			// and that retires c.
			c.bad = true
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// lockFile is replhook.Lock.
func lockFile(ctx context.Context, path string) (unlock func(), err error) {
	l, err := attachedLock(path)
	if err != nil {
		return nil, err
	}
	if err := l.acquire(ctx, math.MaxInt64); err != nil {
		return nil, err
	}
	return func() { l.release(nil) }, nil
}

// tryLockFile is replhook.TryLock.
func tryLockFile(path string) (unlock func(), ok bool, err error) {
	l, err := attachedLock(path)
	if err != nil {
		return nil, false, err
	}
	// With no wait, acquire fails only when the lock is held.
	if l.acquire(context.Background(), 0) != nil {
		return nil, false, nil
	}
	return func() { l.release(nil) }, true, nil
}

// attachedLock is the write lock of path, which must be attached.
func attachedLock(path string) (*writeLock, error) {
	file, err := fileOf(path)
	if err != nil {
		return nil, err
	}
	writeLocks.Lock()
	l := findWriteLock(file)
	writeLocks.Unlock()
	if l == nil || !l.attached {
		return nil, errors.New("sqlite: " + path + " is not replicated")
	}
	return l, nil
}
