// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build windows && (amd64 || arm64)

package vfs

// JournalStopped says a failed sync stopped the rollback journal of the
// database name. A failed sync stops no journal on windows: see
// journal_unix.go.
func JournalStopped(name string) bool { return false }
