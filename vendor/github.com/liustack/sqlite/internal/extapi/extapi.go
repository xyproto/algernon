// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

// Package extapi gives the packages of ext/ what the driver does not offer
// its users. Package sqlite fills it in as it is initialized.
package extapi

import "database/sql/driver"

// BorrowedFunction is sqlite.WithFunction for a function that reads its
// BLOB arguments in place, without a copy: they are SQLite's memory and
// hold only until the function returns, so it must not keep them: after it,
// the memory may hold another connection's data, or be unmapped, and a read
// of it then kills the process. It returns a sqlite.Option. flags are
// sqlite.FuncFlag values.
var BorrowedFunction func(name string, nArg int, fn func(args []driver.Value) (driver.Value, error), flags ...int) any
