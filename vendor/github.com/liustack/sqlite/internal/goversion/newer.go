// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build go1.28 && !sqlite_anygo

package goversion

// The engine is checked with Go 1.26 and 1.27. A newer Go stops here, with
// this name in the error. Moving the line above to the next version is the
// last step of "Moving to a new Go" in docs/upstream.md.
var _ = liustack_sqlite_is_checked_up_to_go1_27_see_docs_upstream_md_or_build_with_tags_sqlite_anygo
