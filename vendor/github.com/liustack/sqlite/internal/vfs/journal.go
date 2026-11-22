// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64))

package vfs

import "sync/atomic"

// stoppedJournals counts the Files of rollback journals that a failed sync
// stopped and that are still open.
var stoppedJournals atomic.Int64

// JournalsStopped says a failed sync stopped a rollback journal that a File
// of the process still has open: see journal_unix.go.
func JournalsStopped() bool { return stoppedJournals.Load() > 0 }
