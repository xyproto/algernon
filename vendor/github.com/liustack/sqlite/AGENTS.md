# liustack/sqlite (project notes for AI agents)

## Goal

`github.com/liustack/sqlite` is a SQLite driver for Go's `database/sql` that needs no cgo. SQLite's C (the amalgamation in `engine/`) is compiled by LLVM to Go assembly that is checked in, so building a program with it needs no C compiler. The driver will also merge the autocommit writes of many goroutines into one transaction (group commit), so that concurrent writers do not each pay for a commit. See [docs/architecture.md](docs/architecture.md).

## Priorities

Stability and safety first, speed second, memory third. A change that is faster or smaller but could lose a write, return a wrong result or weaken a commit is not taken.

## Hard constraints

- Pure Go, `CGO_ENABLED=0` for the user of the module: no cgo, no C compiler, no shared library. Targets are amd64 and arm64.
- SQLite's C is never edited. `engine/sqlite3.c` and `engine/sqlite3.h` are the amalgamation as released (`engine/vendor.sh --verify`). Compile options go in `engine/sqlite.c`, each with its reason. Anything SQLite needs from outside is given by `engine/libc` or by the Go host.
- The generated Go assembly is never edited by hand. It is always what `make gen` makes from `engine/` now, and a change to `engine/`, the generator or `scripts/gen-engine.sh` carries the files made again in the same commit (once the generated package is checked in).
- The driver registers itself as `sqlite`. modernc.org/sqlite and glebarez/go-sqlite use the same name, so a program that imports two of them panics. This is accepted. `sql.OpenDB(sqlite.NewConnector(dsn))` is the way around it.
- Group commit only merges autocommit DML on the main database, and only between connections of one connector that have not changed themselves since they opened (a PRAGMA that sets something, ATTACH, temp objects, read-only). Go functions, aggregates and collations are fixed per connector, and a connector with options of its own has a key of its own, so merged connections always have the same functions. A merged statement must give what it would give run alone on its own connection, except `changes()` and `total_changes()` (documented in docs/architecture.md). Each statement runs in its own SAVEPOINT so that one failure does not fail the others, each gets its own LastInsertId and RowsAffected, and a call returns only after the commit (durability is unchanged). A statement that ran never runs again: when a failure ends the whole transaction, the statements that ran before it get SQLITE_ABORT_ROLLBACK, and only those that never reached SQLite go back to the queue. The full rules are in docs/group-commit.md. User transactions, PRAGMA, ATTACH and temp tables are never merged. One database file has one writer inside the driver.

## Current state of the repository

The driver runs on linux, darwin and windows, amd64 and arm64. Other targets stop the build (`unsupported.go`). Group commit merges concurrent autocommit writes.

- `engine/`: the C tree. `sqlite3.c` and `sqlite3.h` (SQLite 3.54.1), `sqlite.c` (the compile options), `os.c` and `host.h` (the VFS and mutexes that call the Go host), `flags.sh` (the clang flags, which the generator reads), `vendor.sh`. It has no Go file.
- `engine/libc/`: the small libc SQLite is compiled against: headers, and musl's string and math functions (`vendor.sh` copies and checks them, and patches musl's `libm.h`). No stdio. The math functions are the ones SQLite calls (its SQL math functions, and `log` in FTS5), built without fused multiply-add so every target gives the same bits.
- `engine/pgo/`: the profile `make gen` compiles the C with (`profile.txt`), and what trains it: `train.sh`, the SQL it runs (`train.sql`), the C program that runs it as the driver would (`train.c`) and the host callbacks in C (`host.c`). The benchmarks of `bench/` never go into `train.sql`. See `engine/pgo/README.md`.
- `engine/native.sh`: compiles the engine and its libc for the machine as the generator compiles them, for `engine/pgo/train.sh` and `engine/diff/diff.sh`.
- `engine/diff/`: the C side of `TestDiffC`, which holds the engine in Go assembly to the same C compiled by clang, statement by statement, on the SQL of SQLite's fuzz cases (`diff.c`, built by `diff.sh`). See [docs/testing.md](docs/testing.md#the-same-c-compiled-by-clang).
- `internal/goasm/gen/`: the generator, from github.com/liustack/slowjs. It compiles C to IR, moves address-taken locals to a shadow stack (`pass/`, an LLVM plugin), runs llc and rewrites each function as Go assembly. Its README and parts of its code still describe the QuickJS tree it was written for.
- `internal/csqlite/`: the generated engine (`*_amd64.s`, `*_arm64.s`, `engine_*_<arch>.go`, never edited by hand) and the Go host beside it (`stack.go`, `malloc.go`, `mutex.go`, `hostvfs.go`) with the C API the driver calls (`api.go`).
- `internal/vfs/`: files, locks and shared memory for SQLite, pure Go: as os_unix on linux and darwin, as os_win on windows.
- `internal/powerloss/`: builds the files a power loss could leave, for the tests that play one (`powerloss_test.go`).
- `internal/goversion/`: stops the build on a Go newer than the one the engine was checked with (`-tags sqlite_anygo` builds anyway). Supported: Go 1.26.6 to 1.27, see docs/upstream.md.
- `internal/extapi/`: what the packages of `ext/` get from the driver and its users do not, such as a function that reads its BLOB arguments without a copy (`ext/vec`).
- The root package: the `database/sql` driver.
- `replica`: the replicator. `New(ctx, path, Config)` runs before the program opens the database. The uploader sends the staged files to `<path>/0000/` in the bucket, as Litestream's S3 backend lays them out, each written once (If-None-Match). At start it checks that the replica ends where the staged files do. `Restore(ctx, path, Config, RestoreOptions)` rebuilds a database from the replica, the latest state or the one at a TXID or a time, checking every file, and leaves the last file staged so that `New` goes on from it. In the background the replicator merges level 0 into level 1 each minute and level 1 into level 2 each hour, uploads a snapshot (level 9, read from the database) every `Retention` or day, and deletes what `Retention` lets go. While the store is down, staging goes on up to `StagingLimit` and then stops while checkpoints go on, and a full copy follows once what is staged is uploaded. How to use it and what it guarantees: [docs/replication.md](docs/replication.md).
- `replica/internal/s3`: the S3 client of the replicator (SigV4, PUT, multipart upload, ranged GET, ListObjectsV2, DELETE, conditional PUT), no dependencies. See [docs/testing.md](docs/testing.md#the-s3-client-and-the-replica) for its tests.
- `replica/internal/capture`: stages each change of a database as an LTX file (github.com/superfly/ltx, the format Litestream reads) in `<db>-replica/ltx/0/`, and runs its checkpoints. `internal/replhook` gives it what it needs of the driver: a file attached for replication (no automatic checkpoints, the PRAGMAs that checkpoint refused, a call after each write with the length of the WAL) and the file's write lock. See [docs/testing.md](docs/testing.md#capture).
- `bench/`: benchmarks against other drivers, a module of its own (see `bench/README.md`).
- `scripts/`: `gen-engine.sh` (`make gen`, `make pass`), `check-gen.sh` (`make check-gen`), `check-files.sh` (no large or binary file in the tree, part of `make check`), `go-versions.sh` (the tests with each supported Go), `docs-list.sh`, `llvm-bin.sh`, `llvm-get.sh` (downloads LLVM 22.1.8 on linux, for the weekly workflow), `fuzzdata.go` (downloads SQLite's fuzz cases for `TestFuzzData`, `go run scripts/fuzzdata.go`), `slt.sh` (clones SQL Logic Test for `TestSQLLogic`, needs fossil).

## Docs (`docs/`)

1. Every doc starts with front matter: `summary` (one sentence saying what question the doc answers) and `read_when` (when to read it). See the comment at the top of [scripts/docs-list.sh](scripts/docs-list.sh).
2. Before starting work, run `scripts/docs-list.sh` and read the docs whose Read when applies. Before committing, run `scripts/docs-list.sh --check`.
3. What each doc is for:
   - [docs/architecture.md](docs/architecture.md): the layers, the route from C to Go assembly, the decisions and the routes left out
   - [docs/group-commit.md](docs/group-commit.md): the rules of group commit: what merges, how a batch runs, who gets what error when it fails
   - [docs/query-cache.md](docs/query-cache.md): the rules of the statements the driver keeps: which, until when, and why writes are not kept
   - [docs/testing.md](docs/testing.md): the checks before a commit and the tests of the generator
   - [docs/commit.md](docs/commit.md): commit rules
   - [docs/upstream.md](docs/upstream.md): where the code comes from and how to move to a new SQLite
   - [docs/benchmarks.md](docs/benchmarks.md): how fast the driver is against the other drivers, on linux/amd64 and darwin/arm64
   - [docs/replication.md](docs/replication.md): how to replicate a database to S3 and restore it, what it changes for the program, and its limits
   - [docs/busy-cores.md](docs/busy-cores.md): how long a call waits when goroutines outnumber cores, what the driver does about it, what a program can do, and what a change to a lock or a wake must keep
4. Docs are written for readers: conclusions and how to do things, not reasoning or work logs. The README is a landing page and only links to the docs.
5. Docs are written in English. README.md, docs/architecture.md, docs/benchmarks.md, docs/busy-cores.md, docs/group-commit.md, docs/query-cache.md, docs/replication.md and docs/upstream.md also have a `*.zh-CN.md` Chinese version, linked from the top of each side: when you change one side, change the other to match. `scripts/docs-list.sh` skips `*.zh-CN.md` files.
6. English: keep it short and direct, use everyday words, and do not invent terms. Both languages: do not use semicolons, and do not use dashes as punctuation.

## License headers

- A file written for this repository starts with this header, in the comment syntax of its language:

  ```go
  // Copyright (c) 2026 The liustack/sqlite Authors.
  // SPDX-License-Identifier: MIT
  // Use of this source code is governed by the MIT license in LICENSE at the
  // root of this repository.
  ```

- Code from a third party keeps its own header and license file, and gets a row in [NOTICE.md](NOTICE.md): SQLite (public domain), musl (MIT, `engine/libc/COPYRIGHT`).

## Draft folder

Plans, experiment notes and raw measurements go in the repository draft folder `.issues/<YYYY-MM-DD-topic>/`, the files of one task in one folder. `.issues/` is not checked in (add it to `.git/info/exclude` in a fresh clone), and checked-in code, docs and scripts never refer to it. Settled conclusions are written into `docs/`.

## Verification and commits

- `make check` before every commit (see [docs/testing.md](docs/testing.md)), and `make test-gen` after a change to the generator. Commit rules are in [docs/commit.md](docs/commit.md).
- For new behavior, write a failing test first. For a bug, find the root cause before changing code, and do not hide it with fallback values.
- Tools are Go programs or short shell scripts. Do not add Python.
- Once a branch is tested (`make check`) and reviewed, merge it into `main` and push. Ask the user before creating or pushing a tag.
