// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

package sqlite

// The result codes of SQLite (https://sqlite.org/rescode.html), for
// Error.Code and Error.ExtendedCode. Each is SQLITE_ and the name in
// sqlite3.h. TestErrorCodesMatchHeader checks them against engine/sqlite3.h.

// Primary result codes.
const (
	ERROR      = 1
	INTERNAL   = 2
	PERM       = 3
	ABORT      = 4
	BUSY       = 5
	LOCKED     = 6
	NOMEM      = 7
	READONLY   = 8
	INTERRUPT  = 9
	IOERR      = 10
	CORRUPT    = 11
	NOTFOUND   = 12
	FULL       = 13
	CANTOPEN   = 14
	PROTOCOL   = 15
	EMPTY      = 16
	SCHEMA     = 17
	TOOBIG     = 18
	CONSTRAINT = 19
	MISMATCH   = 20
	MISUSE     = 21
	NOLFS      = 22
	AUTH       = 23
	FORMAT     = 24
	RANGE      = 25
	NOTADB     = 26
	NOTICE     = 27
	WARNING    = 28
)

// Extended result codes.
const (
	ERROR_MISSING_COLLSEQ   = ERROR | 1<<8
	ERROR_RETRY             = ERROR | 2<<8
	ERROR_SNAPSHOT          = ERROR | 3<<8
	ERROR_RESERVESIZE       = ERROR | 4<<8
	ERROR_KEY               = ERROR | 5<<8
	ERROR_UNABLE            = ERROR | 6<<8
	IOERR_READ              = IOERR | 1<<8
	IOERR_SHORT_READ        = IOERR | 2<<8
	IOERR_WRITE             = IOERR | 3<<8
	IOERR_FSYNC             = IOERR | 4<<8
	IOERR_DIR_FSYNC         = IOERR | 5<<8
	IOERR_TRUNCATE          = IOERR | 6<<8
	IOERR_FSTAT             = IOERR | 7<<8
	IOERR_UNLOCK            = IOERR | 8<<8
	IOERR_RDLOCK            = IOERR | 9<<8
	IOERR_DELETE            = IOERR | 10<<8
	IOERR_BLOCKED           = IOERR | 11<<8
	IOERR_NOMEM             = IOERR | 12<<8
	IOERR_ACCESS            = IOERR | 13<<8
	IOERR_CHECKRESERVEDLOCK = IOERR | 14<<8
	IOERR_LOCK              = IOERR | 15<<8
	IOERR_CLOSE             = IOERR | 16<<8
	IOERR_DIR_CLOSE         = IOERR | 17<<8
	IOERR_SHMOPEN           = IOERR | 18<<8
	IOERR_SHMSIZE           = IOERR | 19<<8
	IOERR_SHMLOCK           = IOERR | 20<<8
	IOERR_SHMMAP            = IOERR | 21<<8
	IOERR_SEEK              = IOERR | 22<<8
	IOERR_DELETE_NOENT      = IOERR | 23<<8
	IOERR_MMAP              = IOERR | 24<<8
	IOERR_GETTEMPPATH       = IOERR | 25<<8
	IOERR_CONVPATH          = IOERR | 26<<8
	IOERR_VNODE             = IOERR | 27<<8
	IOERR_AUTH              = IOERR | 28<<8
	IOERR_BEGIN_ATOMIC      = IOERR | 29<<8
	IOERR_COMMIT_ATOMIC     = IOERR | 30<<8
	IOERR_ROLLBACK_ATOMIC   = IOERR | 31<<8
	IOERR_DATA              = IOERR | 32<<8
	IOERR_CORRUPTFS         = IOERR | 33<<8
	IOERR_IN_PAGE           = IOERR | 34<<8
	IOERR_BADKEY            = IOERR | 35<<8
	IOERR_CODEC             = IOERR | 36<<8
	LOCKED_SHAREDCACHE      = LOCKED | 1<<8
	LOCKED_VTAB             = LOCKED | 2<<8
	BUSY_RECOVERY           = BUSY | 1<<8
	BUSY_SNAPSHOT           = BUSY | 2<<8
	BUSY_TIMEOUT            = BUSY | 3<<8
	CANTOPEN_NOTEMPDIR      = CANTOPEN | 1<<8
	CANTOPEN_ISDIR          = CANTOPEN | 2<<8
	CANTOPEN_FULLPATH       = CANTOPEN | 3<<8
	CANTOPEN_CONVPATH       = CANTOPEN | 4<<8
	CANTOPEN_DIRTYWAL       = CANTOPEN | 5<<8
	CANTOPEN_SYMLINK        = CANTOPEN | 6<<8
	CORRUPT_VTAB            = CORRUPT | 1<<8
	CORRUPT_SEQUENCE        = CORRUPT | 2<<8
	CORRUPT_INDEX           = CORRUPT | 3<<8
	READONLY_RECOVERY       = READONLY | 1<<8
	READONLY_CANTLOCK       = READONLY | 2<<8
	READONLY_ROLLBACK       = READONLY | 3<<8
	READONLY_DBMOVED        = READONLY | 4<<8
	READONLY_CANTINIT       = READONLY | 5<<8
	READONLY_DIRECTORY      = READONLY | 6<<8
	ABORT_ROLLBACK          = ABORT | 2<<8
	CONSTRAINT_CHECK        = CONSTRAINT | 1<<8
	CONSTRAINT_COMMITHOOK   = CONSTRAINT | 2<<8
	CONSTRAINT_FOREIGNKEY   = CONSTRAINT | 3<<8
	CONSTRAINT_FUNCTION     = CONSTRAINT | 4<<8
	CONSTRAINT_NOTNULL      = CONSTRAINT | 5<<8
	CONSTRAINT_PRIMARYKEY   = CONSTRAINT | 6<<8
	CONSTRAINT_TRIGGER      = CONSTRAINT | 7<<8
	CONSTRAINT_UNIQUE       = CONSTRAINT | 8<<8
	CONSTRAINT_VTAB         = CONSTRAINT | 9<<8
	CONSTRAINT_ROWID        = CONSTRAINT | 10<<8
	CONSTRAINT_PINNED       = CONSTRAINT | 11<<8
	CONSTRAINT_DATATYPE     = CONSTRAINT | 12<<8
	NOTICE_RECOVER_WAL      = NOTICE | 1<<8
	NOTICE_RECOVER_ROLLBACK = NOTICE | 2<<8
	NOTICE_RBU              = NOTICE | 3<<8
	WARNING_AUTOINDEX       = WARNING | 1<<8
	AUTH_USER               = AUTH | 1<<8
)
