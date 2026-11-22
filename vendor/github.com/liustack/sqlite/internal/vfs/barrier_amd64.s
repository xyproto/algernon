// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build linux || darwin || windows

#include "textflag.h"

// func ShmBarrier()
TEXT ·ShmBarrier(SB), NOSPLIT|NOFRAME, $0-0
	MFENCE
	RET
