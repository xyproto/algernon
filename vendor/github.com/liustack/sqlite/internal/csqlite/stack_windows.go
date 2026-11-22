// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build windows && (amd64 || arm64)

package csqlite

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

// mapStack maps a shadow stack: stackSize bytes aligned to stackSize, and an
// inaccessible guard page after them. Windows frees a reservation only
// whole, so it reserves twice the size to find an aligned address, frees
// it, and takes the aligned part at that address, again if another thread
// took the address in between. The pages are committed: a page the C
// touched while only reserved would stop the process. Commit charges the
// system's commit limit, not memory: Windows gives a page as it is touched.
func mapStack() (uintptr, error) {
	for range 16 {
		p, err := windows.VirtualAlloc(0, 2*stackSize+uintptr(pageSize), windows.MEM_RESERVE, windows.PAGE_NOACCESS)
		if err != nil {
			return 0, err
		}
		base, _ := stackLayout(p, pageSize)
		if err := windows.VirtualFree(p, 0, windows.MEM_RELEASE); err != nil {
			return 0, err
		}
		got, err := windows.VirtualAlloc(base, stackSize+pageSize, windows.MEM_RESERVE|windows.MEM_COMMIT, windows.PAGE_READWRITE)
		if err == windows.ERROR_INVALID_ADDRESS {
			continue // another thread took the address
		}
		if err != nil {
			return 0, err // out of commit, most likely
		}
		var old uint32
		if err := windows.VirtualProtect(got+stackSize, pageSize, windows.PAGE_NOACCESS, &old); err != nil {
			windows.VirtualFree(got, 0, windows.MEM_RELEASE)
			return 0, err
		}
		return got, nil
	}
	return 0, errors.New("no aligned address could be kept")
}

// unmapStack gives back the shadow stack at base and its guard page.
func unmapStack(base uintptr) { munmap(base, stackSize+pageSize) }

// mmap maps n bytes of zeroed memory.
func mmap(n uintptr) (unsafe.Pointer, error) {
	p, err := windows.VirtualAlloc(0, n, windows.MEM_RESERVE|windows.MEM_COMMIT, windows.PAGE_READWRITE)
	return at(p), err
}

// munmap gives back the n bytes at p, all of what one mmap or mapStack gave.
// It fails only for an address the host never mapped, which is a bug of the
// host.
func munmap(p, n uintptr) {
	if err := windows.VirtualFree(p, 0, windows.MEM_RELEASE); err != nil {
		panic("csqlite: VirtualFree: " + err.Error())
	}
}
