// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build !(((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64)))

package sqlite

// The engine is Go assembly for amd64 and arm64, and the files and locks are
// written for linux, darwin and windows. Elsewhere the package would build
// with no driver, and a program would fail only when it opened a database,
// so the build stops here with a name that says why.
var _ = liustack_sqlite_runs_on_linux_darwin_windows_amd64_arm64_only
