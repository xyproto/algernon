// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build windows && (amd64 || arm64)

package vfs

// WALStopped says a failed sync stopped the WAL of the database name. A
// failed sync stops no WAL on windows: see wal_unix.go.
func WALStopped(name string) bool { return false }
