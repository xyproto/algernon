# Changelog

What changed in each release, for the people who use the driver. Upgrade with `go get github.com/liustack/sqlite@latest`.

## v0.1.8 (2026-11-21)

- **windows/arm64 is supported ([#2](https://github.com/liustack/sqlite/issues/2)).** Before, the package built on windows/arm64 but registered no driver, so a program failed at run time with `sql: unknown driver "sqlite"`. Every push is now tested on a Windows on Arm machine. Thanks to @ourongxing.
- **Other targets stop the build.** On freebsd, 32-bit or riscv64 the package built with no driver, so a program failed only when it opened a database. The build now stops with one error that lists the supported targets.
- **`PRAGMA optimize` takes the write lock.** It may run ANALYZE, which the driver ran without its write lock, so it failed with `SQLITE_BUSY` while another connection of the process wrote. `PRAGMA optimize` with a mask, as SQLite advises for a long-lived connection, no longer stops the connection from joining batches. `_pragma=optimize` in the DSN is refused with an error that says to run it with `Exec`.
- **ATTACH of a replicated file is refused.** Writes through the attached name went around the replicator and its checkpoints, so it could only stage a full copy of the file. Two databases can each have their own replicator.
- **The replicator stages beside a long write transaction** instead of waiting for its lock.
- **A reader finds a mapped region of the WAL index without a mutex**, so readers that fill every core no longer wait on each other there.

## v0.1.7 (2026-11-19)

- **A writer beside readers that keep every core busy is not starved**, and a pool of connections reads as fast as in v0.1.5 again. Both broke in v0.1.6, which is retracted.
- **The replicator checkpoints every 4000 frames beside a reader** that holds its snapshot open, instead of letting the WAL grow.

## v0.1.6 (2026-11-18), retracted

- Readers of one process no longer wait for each other on the WAL index, and each connection has its own arena of the C heap. Beside readers that keep every core busy, a writer wrote 1 900 rows a second, and a pool of connections read about 8% slower than in v0.1.5. Use v0.1.7.

## v0.1.5 (2026-11-18)

- **Group commit never runs a statement twice.** When a failure ends a batch, the statements that ran get `SQLITE_ABORT_ROLLBACK` and only those that never reached SQLite are queued again. A Go function never runs twice for one statement.
- **A failed sync stops the WAL or rollback journal** instead of checkpointing frames the disk may have lost. Calls fail until the file can open again safely. See [architecture.md](docs/architecture.md).
- **A connection is dropped after out of memory, an I/O error or a full disk**, and a call can leave a batch when its context ends.
- **Writers take the lock in the order they came.** A write from a read transaction fails at once when the lock is held, instead of waiting.
- `_wal_limit` starts the WAL over once it passes the limit. Automatic checkpoints run each `wal_autocheckpoint` frames.
- `sqlite.Error` is exported with its primary and extended result codes.
- Read queries keep their prepared statements on their connection. The engine is built with a trained profile and compares strings eight bytes at a time.
- Power losses are tested on Windows and on a machine that replicates.

## v0.1.4 (2026-11-07)

- The replicator stages and checkpoints as the writers commit, and checkpoints every 4000 frames by default.
- `Close` uploads what it staged while uploads were stopped.
- Fixes from a full review of the driver, the extensions and the replicator.

## v0.1.3 (2026-11-05)

- **Replication to S3** (AWS S3, R2, MinIO): `replica.New` and `replica.Restore`, compaction, snapshots and retention, and a staging limit while the store is down. See [replication.md](docs/replication.md).
- Go aggregate and window functions, read-only and writable virtual tables in Go, and `ext/series`, `ext/csv`, `ext/unicode`, `ext/vec`.
- Large values are read straight from their overflow pages. Memory SQLite frees goes back to the system.

## v0.1.2 (2026-11-05)

- Go 1.26.6 to 1.27 are supported, and the build stops on a newer Go (`-tags sqlite_anygo` builds anyway).
- The `modeof` and `readonly_shm` URI parameters.

## v0.1.1 (2026-11-05)

- windows/amd64: files, locks and WAL as SQLite's os_win keeps them.

## v0.1.0 (2026-11-04)

- First release: SQLite 3.54.1 for `database/sql` with no cgo, on linux and darwin, amd64 and arm64, with group commit.
