// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64))

package vfs

import "sync/atomic"

// Op is a file operation a test can make fail (SetFault).
type Op int

const (
	OpWrite Op = iota
	OpSync
	OpTruncate
)

// fault is the function SetFault set, nil when none is.
var fault atomic.Pointer[func(path string, op Op) int]

// SetFault has every write, sync and truncate of a file ask f first, with
// the name the file was opened by: a result other than OK is what the
// operation gives, and it is not done. It is for tests that check what
// SQLite and the driver do when the disk fails. nil turns it off.
func SetFault(f func(path string, op Op) int) {
	if f == nil {
		fault.Store(nil)
		return
	}
	fault.Store(&f)
}

// faultOf is the result SetFault's function gives for op on f, OK when none
// is set.
func (f *File) faultOf(op Op) int {
	return faultOn(f.path, op)
}

// faultOn is the result SetFault's function gives for op on the file opened
// by the name path, OK when none is set.
func faultOn(path string, op Op) int {
	if p := fault.Load(); p != nil {
		return (*p)(path, op)
	}
	return OK
}
