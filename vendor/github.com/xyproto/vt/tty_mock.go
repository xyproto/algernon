package vt

import "io"

// NewTTYFromReader constructs a TTY that sources its input bytes from r
// instead of from a real terminal. It is intended for tests and for driving
// editors / REPLs from a scripted input source.
//
// All terminal-mutating methods (RawMode, NoBlock, Restore, RestoreNoFlush,
// Flush, SetTimeout, SetTimeoutNoSave, Poll) become safe no-ops on the
// returned TTY. Close will close r if it implements io.Closer, but will not
// touch any real file descriptor. The returned TTY is otherwise
// interchangeable with one produced by NewTTY() and can be passed to any
// API that takes a *TTY.
//
// Typical usage:
//
//	tty := vt.NewTTYFromReader(strings.NewReader("hi\x11")) // "hi" + Ctrl-Q
//	for {
//	    k := tty.ReadKey()
//	    if k == "" { break }
//	    // ...
//	}
func NewTTYFromReader(r io.Reader) *TTY {
	return &TTY{reader: r, timeout: defaultTimeout}
}
