package vt

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"unicode"
)

// NewCanvasWithSize constructs a Canvas of the given width and height
// without querying the terminal for its size. It is the test-friendly
// counterpart to NewCanvas and is intended for unit tests, scripted
// scenarios, and anywhere a deterministic grid is needed.
//
// If w or h is zero, a 1x1 canvas is returned so callers never need to
// guard against a zero-area grid.
func NewCanvasWithSize(w, h uint) *Canvas {
	if w == 0 {
		w = 1
	}
	if h == 0 {
		h = 1
	}
	c := &Canvas{
		w:                 w,
		h:                 h,
		mut:               &sync.RWMutex{},
		cursorVisible:     false,
		termCursorVisible: true,
		lineWrap:          false,
		runewise:          false,
	}
	c.chars = make([]ColorRune, c.w*c.h)
	for i := range c.chars {
		c.chars[i].fg = Default
		c.chars[i].bg = DefaultBackground
	}
	c.oldchars = make([]ColorRune, 0)
	return c
}

// snapshotVersion is the format version tag emitted by Canvas.Snapshot.
// The format is intentionally simple and line-oriented so golden-file
// diffs read naturally. Incompatible changes should bump this number.
const snapshotVersion = 1

// Snapshot writes a deterministic, diff-friendly textual representation
// of the canvas to w. The format is:
//
//	vt-snapshot <version> w=<W> h=<H>
//	<row 0, exactly W runes>
//	<row 1, exactly W runes>
//	...
//	<row H-1, exactly W runes>
//
// Null runes render as spaces; non-printable runes render as '?'. No ANSI
// escape sequences, colors, or cursor state are included — callers that
// care about those can inspect the Canvas directly. Width is measured in
// runes, not display cells, so wide-character continuation cells render as
// the single rune that was stored there (typically a space) to keep the
// output rectangular.
func (c *Canvas) Snapshot(w io.Writer) error {
	c.mut.RLock()
	defer c.mut.RUnlock()

	var sb strings.Builder
	fmt.Fprintf(&sb, "vt-snapshot %d w=%d h=%d\n", snapshotVersion, c.w, c.h)
	for y := uint(0); y < c.h; y++ {
		for x := uint(0); x < c.w; x++ {
			r := c.chars[y*c.w+x].r
			switch {
			case r == 0:
				sb.WriteRune(' ')
			case unicode.IsPrint(r):
				sb.WriteRune(r)
			default:
				sb.WriteRune('?')
			}
		}
		sb.WriteByte('\n')
	}
	_, err := io.WriteString(w, sb.String())
	return err
}
