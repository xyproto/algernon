# Contributing to sqlite

First, the policy: **liustack/sqlite does not accept pull requests.** A database driver has to keep every write it is given, and here a change can reach from SQLite's C, through the generator that turns it into Go assembly, to group commit and the replicator. Getting that right takes a lot of context, so one maintainer writes, reviews and owns every line.

Two ways to help that make a real difference:

- **[Open an issue](https://github.com/liustack/sqlite/issues/new/choose).** Bugs, wrong results, slow queries, confusing errors, unclear docs, ideas. A clear issue with a program that reproduces it helps more than a patch, and issues shape what gets built next. The templates tell you what to include.
- **Fork it.** Under the MIT license your copy is yours to change, rename and publish. No permission needed. SQLite stays in the public domain and musl's code keeps its MIT license ([NOTICE.md](NOTICE.md)).

Everything below is for people working on a fork.

## Scope

The driver is SQLite for Go's `database/sql` that builds with `CGO_ENABLED=0`: SQLite's C is compiled ahead of time to Go assembly that is checked in. These stay out:

- cgo, a C compiler at build time, or a shared library.
- Changes to SQLite's C. `engine/sqlite3.c` and `engine/sqlite3.h` are the amalgamation as released. A bug in SQLite goes to SQLite, and the driver moves to the release that fixes it ([docs/upstream.md](docs/upstream.md)).
- Hand edits to the generated assembly. It is always what `make gen` makes from `engine/`.
- A change that is faster but could lose a write, return a wrong result or weaken a commit.

[AGENTS.md](AGENTS.md) has the full list of constraints.

## Setup

You need Go 1.26.6 or later. A Go newer than 1.27 stops the build unless you pass `-tags sqlite_anygo` ([docs/upstream.md](docs/upstream.md) says why).

```sh
make check   # build, gofmt, vet, docs check, tests, race tests
```

Making the assembly again (`make gen`) and the tests of the generator need LLVM 22.1.8. The tests of the replicator need a MinIO binary in `MINIO`. [docs/testing.md](docs/testing.md) says how to get both.

## Tests

- New behavior comes with a test that fails first. A bug fix comes with a test that reproduces it, in the same commit.
- `make check` passes before every commit. It runs the driver, the VFS and the replicator under `go test -race` too.
- The tests that need LLVM, MinIO, the sqlite3 shell or a long run are off by default. [docs/testing.md](docs/testing.md) says how to turn them on and when they are needed.

## Commits

- `type(scope): imperative summary`, at most 72 characters, no period at the end.
- One commit does one thing, and the tree builds and passes its tests after each. Keep a refactor or a reformat apart from a change of behavior. A change to `engine/` or the generator carries the assembly made again.
- Docs are written in English. The README and the architecture, benchmarks and upstream docs have a `*.zh-CN.md` twin that changes with them.

The full rules are in [docs/commit.md](docs/commit.md).
