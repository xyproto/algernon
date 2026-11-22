// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64))

package csqlite

import "errors"

// A Go panic in a host callback would unwind through the frames of the C and
// leave SQLite half way, its locks held. So a callback never lets one out
// when it has a way to fail:
//
//   - A callback with a result code (the VFS calls, malloc, realloc, mutex
//     try) recovers, marks the connection broken and returns an error code.
//     SQLite then unwinds by itself, releases its own locks and returns the
//     error to the caller.
//   - A callback with no result (random bytes, sleep, free, chunk size, the
//     shm barrier) recovers, marks the connection broken and returns. What it
//     skipped cannot matter to a connection that is never used again.
//   - A callback that cannot fail cleanly (mutex enter and leave: a lock half
//     taken is worse than a panic, and host_trap, which must not return)
//     marks the connection broken and lets the panic go on.
//
// A broken connection runs no more C. Its calls return ErrBroken, and the
// driver must drop it. SQLite's memory for it, and any mutex it held, are
// lost, so a broken connection is the sign of a bug of the host, not
// something to retry.

// ErrBroken is the error of a call on a connection a host callback panicked
// under. Test for it with errors.Is.
var ErrBroken = errors.New("sqlite: connection is broken")

const (
	sqliteIOErr = 10
	sqliteNoMem = 7
	sqliteFull  = 13
)

// failWith is deferred by a callback that returns a result: it recovers a
// panic, marks the stack of sp broken and sets the result to code.
func failWith(sp uintptr, r *uintptr, code uintptr) {
	if e := recover(); e != nil {
		poison(sp, e)
		*r = code
	}
}

// failQuiet is deferred by a callback that returns nothing.
func failQuiet(sp uintptr) {
	if e := recover(); e != nil {
		poison(sp, e)
	}
}

// failAndGoOn is deferred by a callback that cannot fail cleanly: it marks
// the stack of sp broken and lets the panic go on.
func failAndGoOn(sp uintptr) {
	if e := recover(); e != nil {
		poison(sp, e)
		panic(e)
	}
}

// Broken says a host callback panicked under the connection. It is safe to
// call from any goroutine.
func (c *Conn) Broken() bool { return c.st.broken.Load() }

// brokenErr is the error of a call on a broken connection.
func (c *Conn) brokenErr() error {
	if !c.st.broken.Load() {
		return nil
	}
	if cause, _ := c.st.cause.Load().(string); cause != "" {
		return &brokenError{cause}
	}
	return ErrBroken
}

type brokenError struct{ cause string }

func (e *brokenError) Error() string { return ErrBroken.Error() + ": " + e.cause }
func (e *brokenError) Unwrap() error { return ErrBroken }
