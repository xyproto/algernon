// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64))

package vfs

// ShmBarrier is a full memory barrier, as sqlite3MemoryBarrier
// (__sync_synchronize): MFENCE on amd64, DMB ISH on arm64. No load or store
// moves across it in either direction. A Go atomic is not enough on arm64:
// without LSE it is a LDAXR/STLXR loop, which lets an earlier store pass a
// later load.
func ShmBarrier()
