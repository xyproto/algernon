// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build (linux || darwin) && (amd64 || arm64)

package csqlite

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

// mapStack maps a shadow stack: stackSize bytes aligned to stackSize, and an
// inaccessible guard page after them. It maps twice the size, keeps the
// aligned region and the guard page, and gives the rest back. The pages are
// not reserved: the kernel gives them as the C touches them.
func mapStack() (uintptr, error) {
	n := uintptr(2*stackSize + pageSize)
	p, err := unix.MmapPtr(-1, 0, nil, n, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_ANON|unix.MAP_PRIVATE|unix.MAP_NORESERVE)
	if err != nil {
		return 0, err
	}
	start := uintptr(p)
	base, end := stackLayout(start, pageSize)
	if base > start {
		munmap(start, base-start)
	}
	if start+n > end {
		munmap(end, start+n-end)
	}
	if err := unix.Mprotect(bytesAt(base+stackSize, int(pageSize)), unix.PROT_NONE); err != nil {
		munmap(base, stackSize+pageSize)
		return 0, err
	}
	return base, nil
}

// unmapStack gives back the shadow stack at base and its guard page.
func unmapStack(base uintptr) { munmap(base, stackSize+pageSize) }

// mmap maps n bytes of zeroed memory.
func mmap(n uintptr) (unsafe.Pointer, error) {
	return unix.MmapPtr(-1, 0, nil, n, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_ANON|unix.MAP_PRIVATE)
}

// munmap gives back n bytes at p. It fails only for an address the host
// never mapped, which is a bug of the host.
func munmap(p, n uintptr) {
	if err := unix.MunmapPtr(at(p), n); err != nil {
		panic("csqlite: munmap: " + err.Error())
	}
}
