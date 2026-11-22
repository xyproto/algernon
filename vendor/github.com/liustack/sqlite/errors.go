// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64))

package sqlite

import "github.com/liustack/sqlite/internal/csqlite"

// Error is an error SQLite gave, with its result code. Read it with
// errors.As:
//
//	var e *sqlite.Error
//	if errors.As(err, &e) && e.Code() == sqlite.CONSTRAINT {
//		// a constraint failed
//	}
//
// Code is the primary result code (CONSTRAINT), ExtendedCode the extended one
// (CONSTRAINT_UNIQUE), as listed in codes.go and at
// https://sqlite.org/rescode.html. Unwrap gives the error under it when there
// is one: the panic of a Go collation, or for ABORT_ROLLBACK under group
// commit, the error that rolled the batch back.
//
// A statement stopped by its context gives the context's error instead.
type Error = csqlite.Error
