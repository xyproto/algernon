// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

// Package gostackparse parses goroutines stack traces as produced by panic()
// or debug.Stack() at ~300 MiB/s.
package gostackparse

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type parserState int

const (
	stateHeader = iota
	stateStackFunc
	stateStackFile
	stateCreatedBy
	stateCreatedByFunc
	stateCreatedByFile
	stateOriginatingFrom
)

// Parse parses a goroutines stack trace dump as produced by runtime.Stack().
// The parser is forgiving and will continue parsing even when encountering
// unexpected data. When this happens it will try to discard the entire
// goroutine that encountered the problem and continue with the next one. It
// will also return an error for every goroutine that couldn't be parsed. If
// all goroutines were parsed successfully, the []error slice is empty.
func Parse(r io.Reader) ([]*Goroutine, []error) {
	var (
		sc      = bufio.NewScanner(r)
		state   parserState
		lineNum int
		line    []byte

		goroutines []*Goroutine
		errs       []error
		g          *Goroutine
		f          *Frame

		abortGoroutine = func(msg string) {
			err := fmt.Errorf(
				"%s on line %d: %q",
				msg,
				lineNum,
				line,
			)
			errs = append(errs, err)
			goroutines = goroutines[0 : len(goroutines)-1]
			state = stateHeader
		}
	)

	// This for loop implements a simple line based state machine for parsing
	// the goroutines dump. The state != stateHeader makes sure we handle an
	// unexpected EOF the same way we'd handle an empty line.
	for sc.Scan() || state != stateHeader {
		line = sc.Bytes()
		lineNum++

	statemachine:
		switch state {
		case stateHeader, stateOriginatingFrom:
			// Ignore lines that don't look like goroutine headers. This helps with
			// leading/trailing whitespace but also in case we encountered an error
			// during the previous goroutine and need to seek to the beginning of the
			// next one.
			if state == stateHeader {
				if !bytes.HasPrefix(line, goroutinePrefix) {
					continue
				}
				g = parseGoroutineHeader(line[len(goroutinePrefix):])
				goroutines = append(goroutines, g)
			}
			if state == stateOriginatingFrom {
				ancestorIDStr := line[len(originatingFromPrefix) : len(line)-2]
				ancestorID, err := strconv.Atoi(string(ancestorIDStr))
				if err != nil {
					abortGoroutine(err.Error())
					continue
				}
				ancestorG := &Goroutine{ID: ancestorID}
				g.Ancestor = ancestorG
				g = ancestorG
			}

			state = stateStackFunc
			if g == nil {
				abortGoroutine("invalid goroutine header")
			}
		case stateStackFunc, stateCreatedByFunc:
			f = parseFunc(line, state)
			if f == nil {
				if bytes.Equal(line, framesElided) {
					g.FramesElided = true
					state = stateCreatedBy
					continue
				}
				if bytes.HasPrefix(line, originatingFromPrefix) {
					state = stateOriginatingFrom
					goto statemachine
				}
				abortGoroutine("invalid function call")
				continue
			}
			if state == stateStackFunc {
				g.Stack = append(g.Stack, f)
				state = stateStackFile
			} else {
				g.CreatedBy = f
				state = stateCreatedByFile
			}
		case stateStackFile, stateCreatedByFile:
			if !parseFile(line, f) {
				abortGoroutine("invalid file:line ref")
				continue
			}
			state = stateCreatedBy
		case stateCreatedBy:
			if bytes.HasPrefix(line, createdByPrefix) {
				line = line[len(createdByPrefix):]
				state = stateCreatedByFunc
				goto statemachine
			} else if len(line) == 0 {
				state = stateHeader
			} else {
				state = stateStackFunc
				goto statemachine
			}
		}
	}

	if err := sc.Err(); err != nil {
		errs = append(errs, err)
	}
	return goroutines, errs
}

var (
	goroutinePrefix       = []byte("goroutine ")
	createdByPrefix       = []byte("created by ")
	originatingFromPrefix = []byte("[originating from goroutine ")
	framesElided          = []byte("...additional frames elided...")
)

var goroutineHeader = regexp.MustCompile(
	`^(\d+) \[([^,]+)(?:, (\d+) minutes)?(, locked to thread)?\]:$`,
)

// parseGoroutineHeader parses a goroutine header line and returns a new
// Goroutine on success or nil on error. It expects the `goroutinePrefix` to
// be trimmed from the beginning of the line.
//
// Example Input:
// 1 [chan receive, 6883 minutes]:
//
// Example Output:
// &Goroutine{ID: 1, State "chan receive", Waitduration: 6883*time.Minute}
func parseGoroutineHeader(line []byte) *Goroutine {
	// TODO(fg) would probably be faster if we didn't use a regexp for this, but
	// might be more hassle than its worth.
	m := goroutineHeader.FindSubmatch(line)
	if len(m) != 5 {
		return nil
	}
	var (
		id          = m[1]
		state       = m[2]
		waitminutes = m[3]
		locked      = m[4]

		g   = &Goroutine{State: string(state), LockedToThread: len(locked) > 0}
		err error
	)

	// regex currently sucks `abc minutes` into the state string if abc is
	// non-numeric, let's not consider this a valid goroutine.
	if strings.HasSuffix(g.State, " minutes") {
		return nil
	}
	if g.ID, err = strconv.Atoi(string(id)); err != nil {
		// should be impossible to end up here
		return nil
	}
	if len(waitminutes) > 0 {
		min, err := strconv.Atoi(string(waitminutes))
		if err != nil {
			// should be impossible to end up here
			return nil
		}
		g.Wait = time.Duration(min) * time.Minute
	}
	return g
}

// parseFunc parse a func call with potential argument addresses and returns a
// new Frame for it on success or nil on error.
//
// Example Input:
// runtime/pprof.writeGoroutineStacks(0x2b016e0, 0xc0995cafc0, 0xc00468e150, 0x0)
//
// Example Output:
// &Frame{Func: "runtime/pprof.writeGoroutineStacks"}
func parseFunc(line []byte, state parserState) *Frame {
	// createdBy func calls don't have trailing parens, just return them as-is.
	if state == stateCreatedByFunc {
		return &Frame{Func: string(line)}
	}

	// A valid func call is supposed to have at least one matched pair of parens.
	// Multiple matched pairs are allowed, but nesting is not.
	var (
		openIndex  = -1
		closeIndex = -1
	)
	for i, r := range line {
		switch r {
		case '(':
			if openIndex != -1 && closeIndex == -1 {
				return nil
			}
			openIndex = i
			closeIndex = -1
		case ')':
			if openIndex == -1 || closeIndex != -1 {
				return nil
			}
			closeIndex = i
		}
	}

	if openIndex == -1 || closeIndex == -1 || openIndex == 0 {
		return nil
	}
	return &Frame{Func: string(line[0:openIndex])}
}

// parseFile parses a file line and updates f accordingly or returns false on
// error.
//
// Example Input:
// \t/root/go1.15.6.linux.amd64/src/net/http/server.go:2969 +0x36c
//
// Example Update:
// &Frame{File: "/root/go1.15.6.linux.amd64/src/net/http/server.go", Line: 2969}
//
// Note: The +0x36c is the relative pc offset from the function entry pc. It's
// omitted if it's 0.
func parseFile(line []byte, f *Frame) (ret bool) {
	if len(line) < 2 || line[0] != '\t' {
		return false
	}
	line = line[1:]

	const (
		stateFilename = iota
		stateColon
		stateLine
	)

	var state = stateFilename
	for i, c := range line {
		switch state {
		case stateFilename:
			if c == ':' {
				state = stateColon
			}
		case stateColon:
			if isDigit(c) {
				f.File = string(line[0 : i-1])
				f.Line = int(c - '0')
				state = stateLine
				ret = true
			} else {
				state = stateFilename
			}
		case stateLine:
			if c == ' ' {
				return true
			} else if !isDigit(c) {
				return false
			}
			f.Line = f.Line*10 + int(c-'0')
		}

	}
	return
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// Goroutine represents a single goroutine and its stack after extracting it
// from the runtime.Stack() text format. See [1] for more info.
// [1] https://github.com/felixge/go-profiler-notes/blob/main/goroutine.md
type Goroutine struct {
	// ID is the goroutine id (aka `goid`).
	ID int
	// State is the `atomicstatus` of the goroutine, or if "waiting" the
	// `waitreason`.
	State string
	// Wait is the approximate duration a goroutine has been waiting or in a
	// syscall as determined by the first gc after the wait started. Aka
	// `waitsince`.
	Wait time.Duration
	// LockedToThread is true if the goroutine is locked by a thread, aka
	// `lockedm`.
	LockedToThread bool
	// Stack is the stack trace of the goroutine.
	Stack []*Frame
	// FramesElided is true if the stack trace contains a message indicating that
	// additional frames were elided. This happens when the stack depth exceeds
	// 100.
	FramesElided bool
	// CreatedBy is the frame that created this goroutine, nil for main().
	CreatedBy *Frame
	// Ancestors are the Goroutines that created this goroutine.
	// See GODEBUG=tracebackancestors=n in https://pkg.go.dev/runtime.
	Ancestor *Goroutine `json:"Ancestor,omitempty"`
}

// Frame is a single call frame on the stack.
type Frame struct {
	// Func is the name of the function, including package name, e.g. "main.main"
	// or "net/http.(*Server).Serve".
	Func string
	// File is the absolute path of source file e.g.
	// "/go/src/example.org/example/main.go".
	File string
	// Line is the line number of inside of the source file that was active when
	// the sample was taken.
	Line int
}
