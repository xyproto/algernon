// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

// Package goversion stops a build with a Go newer than the one the engine was
// checked with. The Go assembly of internal/csqlite leans on how Go starts a
// function, grows its stack and calls the host, which the Go 1 promise does
// not cover, so a newer Go builds it without checking it (see "Moving to a
// new Go" in docs/upstream.md). internal/csqlite imports it: the go command
// builds it first, and its error is the only one the build reports. Build
// with -tags sqlite_anygo to build anyway.
package goversion
