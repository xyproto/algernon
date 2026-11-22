// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64))

package csqlite

import (
	"crypto/rand"
	"time"

	"github.com/liustack/sqlite/internal/vfs"
)

// The VFS callbacks of engine/os.c. Each turns the words of the C into Go
// values and calls internal/vfs. An int of the C arrives in a word whose
// upper bits are not set: int32 cuts it.

var files table[vfs.File]

func i32(w uintptr) int { return int(int32(w)) }

// testFiles, set by a test only, changes what the VFS does with the files of
// SQLite. TestFuzzData keeps them in one directory (path maps each name) and
// smaller than maxSize bytes, as fuzzcheck's VFS does: a write or a truncate
// that would reach it is SQLITE_FULL.
var testFiles struct {
	path    func(string) string
	maxSize int64
}

// fileName is the file name at a, as the VFS takes it.
func fileName(a uintptr) string {
	name := cstring(a)
	if testFiles.path != nil && name != "" {
		name = testFiles.path(name)
	}
	return name
}

// tooLarge says a file of size bytes is past testFiles.maxSize.
func tooLarge(size int64) bool { return testFiles.maxSize > 0 && size >= testFiles.maxSize }

func rc(r int) uintptr { return uintptr(int32(r)) }

func putInt(p uintptr, v int) { *(*int32)(at(p)) = int32(v) }

// host_open opens a file. a is engine/host.h's struct host_open_args.
func host_open(sp, name, flags, a uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteIOErr)
	f, out, r := vfs.OpenWith(fileName(name), i32(flags), vfs.OpenOptions{
		ModeOf:      cstring(*(*uintptr)(at(a))),
		ReadOnlyShm: *(*int32)(at(a + 16)) != 0,
	})
	if r != sqliteOK {
		return rc(r)
	}
	*(*int64)(at(a + 8)) = int64(files.add(f))
	putInt(a+20, out)
	putInt(a+24, f.DeviceCharacteristics())
	return sqliteOK
}

func host_delete(sp, name, syncDir uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteIOErr)
	return rc(vfs.Delete(fileName(name), i32(syncDir) != 0))
}

func host_access(sp, name, flags, res uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteIOErr)
	ok, r := vfs.Access(fileName(name), i32(flags))
	v := 0
	if ok {
		v = 1
	}
	putInt(res, v)
	return rc(r)
}

func host_full_pathname(sp, name, n, out uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteIOErr)
	p, r := vfs.FullPathname(fileName(name))
	if r != vfs.OK && r != vfs.OKSymlink {
		return rc(r)
	}
	if len(p)+1 > i32(n) {
		return sqliteCantOpen
	}
	b := bytesAt(out, len(p)+1)
	copy(b, p)
	b[len(p)] = 0
	return rc(r)
}

func host_randomness(sp, out, n uintptr) {
	defer failQuiet(sp)
	rand.Read(bytesAt(out, i32(n)))
}

func host_sleep(sp, us uintptr) {
	defer failQuiet(sp)
	time.Sleep(time.Duration(i32(us)) * time.Microsecond)
}

// testNowMs, set by a test only, is the time host_now_ms gives in place of
// the clock's: TestDiffC runs the engine at the time its C side runs at.
var testNowMs int64

func host_now_ms(sp uintptr) uintptr {
	if testNowMs != 0 {
		return uintptr(testNowMs)
	}
	return uintptr(time.Now().UnixMilli())
}

// host_localtime writes the fields of struct tm for Unix time t in
// time.Local (engine/host.h).
func host_localtime(sp, t, out uintptr) {
	defer failQuiet(sp)
	lt := time.Unix(int64(t), 0).In(time.Local)
	isDST := 0
	if lt.IsDST() {
		isDST = 1
	}
	for i, v := range [9]int{lt.Second(), lt.Minute(), lt.Hour(), lt.Day(), int(lt.Month()) - 1,
		lt.Year() - 1900, int(lt.Weekday()), lt.YearDay() - 1, isDST} {
		putInt(out+uintptr(4*i), v)
	}
}

func host_close(sp, f uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteIOErr)
	r := files.get(f).Close()
	files.remove(f)
	return rc(r)
}

func host_read(sp, f, buf, n, off uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteIOErr)
	return rc(files.get(f).ReadAt(bytesAt(buf, i32(n)), int64(off)))
}

func host_write(sp, f, buf, n, off uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteIOErr)
	if tooLarge(int64(off) + int64(i32(n))) {
		return sqliteFull
	}
	return rc(files.get(f).WriteAt(bytesAt(buf, i32(n)), int64(off)))
}

func host_truncate(sp, f, size uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteIOErr)
	if tooLarge(int64(size)) {
		return sqliteFull
	}
	return rc(files.get(f).Truncate(int64(size)))
}

func host_sync(sp, f, flags uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteIOErr)
	return rc(files.get(f).Sync(i32(flags)))
}

func host_file_size(sp, f, size uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteIOErr)
	s, r := files.get(f).Size()
	*(*int64)(at(size)) = s
	return rc(r)
}

func host_lock(sp, f, level uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteIOErr)
	return rc(files.get(f).Lock(i32(level)))
}

func host_unlock(sp, f, level uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteIOErr)
	return rc(files.get(f).Unlock(i32(level)))
}

func host_check_reserved_lock(sp, f, res uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteIOErr)
	ok, r := files.get(f).CheckReservedLock()
	v := 0
	if ok {
		v = 1
	}
	putInt(res, v)
	return rc(r)
}

func host_lock_state(sp, f uintptr) (ret uintptr) {
	defer failWith(sp, &ret, 0)
	return rc(files.get(f).LockState())
}

func host_size_hint(sp, f, size uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteIOErr)
	if tooLarge(int64(size)) {
		return sqliteFull
	}
	return rc(files.get(f).SizeHint(int64(size)))
}

func host_chunk_size(sp, f, n uintptr) {
	defer failQuiet(sp)
	files.get(f).SetChunkSize(i32(n))
}

func host_has_moved(sp, f uintptr) (ret uintptr) {
	defer failWith(sp, &ret, 0)
	if files.get(f).HasMoved() {
		return 1
	}
	return 0
}

func host_shm_map(sp, f, region, size, extend, p uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteIOErr)
	a, r := files.get(f).ShmMap(i32(region), i32(size), i32(extend) != 0)
	*(*uintptr)(at(p)) = a
	return rc(r)
}

func host_shm_lock(sp, f, offset, n, flags uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteIOErr)
	return rc(files.get(f).ShmLock(i32(offset), i32(n), i32(flags)))
}

func host_shm_barrier(sp uintptr) {
	defer failQuiet(sp)
	vfs.ShmBarrier()
}

func host_shm_unmap(sp, f, del uintptr) (ret uintptr) {
	defer failWith(sp, &ret, sqliteIOErr)
	return rc(files.get(f).ShmUnmap(i32(del) != 0))
}

const sqliteCantOpen = 14
