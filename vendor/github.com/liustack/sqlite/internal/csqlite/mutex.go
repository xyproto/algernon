// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64))

package csqlite

import (
	"sync"
	"sync/atomic"
)

// A table gives small numbers (handles) to Go values the C names: a handle is
// never 0, so the C can test it for NULL. Lookups take no lock.
type table[T any] struct {
	mu    sync.Mutex
	free  []uint32
	n     uint32
	pages [1024]atomic.Pointer[[1024]*T]
}

func (t *table[T]) add(v *T) uintptr {
	t.mu.Lock()
	defer t.mu.Unlock()
	var h uint32
	if k := len(t.free); k > 0 {
		h = t.free[k-1]
		t.free = t.free[:k-1]
	} else {
		t.n++
		h = t.n
		if h>>10 >= uint32(len(t.pages)) {
			panic("csqlite: too many handles")
		}
		if t.pages[h>>10].Load() == nil {
			t.pages[h>>10].Store(new([1024]*T))
		}
	}
	t.pages[h>>10].Load()[h&1023] = v
	return uintptr(h)
}

func (t *table[T]) get(h uintptr) *T {
	h &= 0xffffffff
	p := t.pages[h>>10].Load()
	if p == nil || p[h&1023] == nil {
		panic("csqlite: a handle the host did not give")
	}
	return p[h&1023]
}

func (t *table[T]) remove(h uintptr) {
	h &= 0xffffffff
	t.get(h)
	t.mu.Lock()
	t.pages[h>>10].Load()[h&1023] = nil
	t.free = append(t.free, uint32(h))
	t.mu.Unlock()
}

// SQLite's mutex types (sqlite3.h).
const (
	mutexFast      = 0
	mutexRecursive = 1
	mutexStaticMax = 13 // SQLITE_MUTEX_STATIC_VFS3, the last static one
)

// A mutex of SQLite. A recursive one knows its owner by the shadow-stack
// region of the connection that holds it: one goroutine at a time runs C on
// a region.
type mutex struct {
	mu        sync.Mutex
	recursive bool
	owner     atomic.Uintptr
	depth     int
}

var (
	mutexes      table[mutex]
	staticMutex  [mutexStaticMax + 1]uintptr
	staticMutexO sync.Once
)

func host_mutex_alloc(sp, typ uintptr) (ret uintptr) {
	// A NULL mutex is what SQLite reads as out of memory.
	defer failWith(sp, &ret, 0)
	typ = uintptr(int32(typ))
	switch {
	case typ == mutexFast:
		return mutexes.add(&mutex{})
	case typ == mutexRecursive:
		return mutexes.add(&mutex{recursive: true})
	case typ <= mutexStaticMax:
		staticMutexO.Do(func() {
			for i := 2; i <= mutexStaticMax; i++ {
				staticMutex[i] = mutexes.add(&mutex{})
			}
		})
		return staticMutex[typ]
	}
	return 0
}

func host_mutex_free(sp, m uintptr) {
	defer failQuiet(sp)
	for _, s := range staticMutex {
		if m&0xffffffff == s {
			return
		}
	}
	mutexes.remove(m)
}

func host_mutex_enter(sp, m uintptr) {
	defer failAndGoOn(sp)
	x := mutexes.get(m)
	if !x.recursive {
		x.mu.Lock()
		return
	}
	me := stackBase(sp)
	if x.owner.Load() == me {
		x.depth++
		return
	}
	x.mu.Lock()
	x.owner.Store(me)
	x.depth = 1
}

func host_mutex_try(sp, m uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteBusy)
	x := mutexes.get(m)
	if !x.recursive {
		if x.mu.TryLock() {
			return sqliteOK
		}
		return sqliteBusy
	}
	me := stackBase(sp)
	if x.owner.Load() == me {
		x.depth++
		return sqliteOK
	}
	if !x.mu.TryLock() {
		return sqliteBusy
	}
	x.owner.Store(me)
	x.depth = 1
	return sqliteOK
}

func host_mutex_leave(sp, m uintptr) {
	defer failAndGoOn(sp)
	x := mutexes.get(m)
	if x.recursive {
		x.depth--
		if x.depth > 0 {
			return
		}
		x.owner.Store(0)
	}
	x.mu.Unlock()
}

const (
	sqliteOK   = 0
	sqliteBusy = 5
)
