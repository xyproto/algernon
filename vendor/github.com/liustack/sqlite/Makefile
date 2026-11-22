# Copyright (c) 2026 The liustack/sqlite Authors.
# SPDX-License-Identifier: MIT
# Use of this source code is governed by the MIT license in LICENSE at the
# root of this repository.

.PHONY: bench bench-docs check check-gen gen pass test-gen vendor-check

# The cases of SQLite's fuzzcheck, once go run scripts/fuzzdata.go put them in
# .build/fuzzdata (docs/testing.md).
FUZZDATA ?= $(if $(wildcard .build/fuzzdata/fuzzdata1.db),$(CURDIR)/.build/fuzzdata)
# SQL Logic Test, once scripts/slt.sh put it in .build/slt (docs/testing.md).
SLT ?= $(if $(wildcard .build/slt/src/test/select1.test),$(CURDIR)/.build/slt/src/test)

# The checks every commit passes (docs/testing.md).
check: vendor-check
	scripts/check-files.sh
	go build ./...
	test -z "$$(git ls-files --cached --others --exclude-standard '*.go' | xargs gofmt -l -s)"
	go vet -asmdecl=false ./...
	scripts/docs-list.sh --check >/dev/null
	FUZZDATA=$(FUZZDATA) SLT=$(SLT) go test -count=1 ./...
	FUZZDATA= SLT= go test -race -count=1 . ./internal/vfs ./internal/csqlite ./replica/...
	cd bench && go vet -tags liustack ./... && go vet -tags turso ./... && go run ./cmd/benchrun docs -check

# The C of engine/ as Go assembly, on a machine with LLVM 22.1.8: make gen,
# make gen ARCH=amd64. ARCH defaults to the machine's. It compiles with the
# profile engine/pgo/profile.txt, which engine/pgo/train.sh trains.
gen:
	scripts/gen-engine.sh $(ARCH)

# The checked-in engine is what engine/ generates, and passes the post-link
# check on both targets (docs/testing.md).
check-gen:
	scripts/check-gen.sh

# The LLVM pass alone, into .build/shadow.so.
pass:
	scripts/gen-engine.sh pass

# The tests of the generator that run LLVM (docs/testing.md).
test-gen: pass
	GEN_LLVM=$$(scripts/llvm-bin.sh) GEN_SHADOW_PASS=$(CURDIR)/.build/shadow.so \
		go test -count=1 -timeout 30m ./internal/goasm/...

# The driver benchmarks (bench/README.md), for example
# make bench ARGS="-count 2 -drivers liustack,mattn".
bench:
	cd bench && go run ./cmd/benchrun run $(ARGS)

# The tables of docs/benchmarks*.md and the READMEs, from bench/published.
bench-docs:
	cd bench && go run ./cmd/benchrun docs

# The vendored C is what its pinned release has.
vendor-check:
	engine/vendor.sh --verify
	engine/libc/vendor.sh --verify
