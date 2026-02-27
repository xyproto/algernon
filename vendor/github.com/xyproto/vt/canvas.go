package vt

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// ColorRune holds a single terminal cell
type ColorRune struct {
	fg    AttributeColor
	bg    AttributeColor
	r     rune
	drawn bool
	cw    uint8 // 0=normal, 1=continuation (skip), 2=wide (2-col)
}

// Char is an alias for ColorRune, for API stability
type Char ColorRune

// Canvas represents a 2D grid of colored characters
type Canvas struct {
	mut               *sync.RWMutex
	chars             []ColorRune
	oldchars          []ColorRune
	w                 uint
	h                 uint
	cursorVisible     bool // desired state
	termCursorVisible bool // last state sent to terminal
	lineWrap          bool
	runewise          bool
}

// canvasCopy is a Canvas without the mutex
type canvasCopy struct {
	chars             []ColorRune
	oldchars          []ColorRune
	w                 uint
	h                 uint
	cursorVisible     bool
	termCursorVisible bool
	lineWrap          bool
	runewise          bool
}

// NewCanvas creates a canvas sized to the current terminal
func NewCanvas() *Canvas {
	c := &Canvas{}
	c.w, c.h = MustTermSize()
	c.chars = make([]ColorRune, c.w*c.h)
	for i := 0; i < len(c.chars); i++ {
		c.chars[i].fg = Default
		c.chars[i].bg = DefaultBackground
	}
	c.oldchars = make([]ColorRune, 0)
	c.mut = &sync.RWMutex{}
	c.cursorVisible = false
	c.termCursorVisible = true // assume visible so flushCursor emits the hide escape
	c.lineWrap = false
	c.runewise = multiplexed // use explicit cursor positioning under multiplexers to avoid line-wrap issues
	c.flushCursor()
	c.SetLineWrap(c.lineWrap)
	return c
}

// Copy creates a new Canvas struct that is a copy of this one.
// The mutex is initialized as a new mutex.
func (c *Canvas) Copy() Canvas {
	c.mut.RLock()
	defer c.mut.RUnlock()

	cc := canvasCopy{
		chars:             make([]ColorRune, len(c.chars)),
		oldchars:          make([]ColorRune, len(c.oldchars)),
		w:                 c.w,
		h:                 c.h,
		cursorVisible:     c.cursorVisible,
		termCursorVisible: c.termCursorVisible,
		lineWrap:          c.lineWrap,
		runewise:          c.runewise,
	}
	copy(cc.chars, c.chars)
	copy(cc.oldchars, c.oldchars)

	return Canvas{
		chars:             cc.chars,
		oldchars:          cc.oldchars,
		w:                 cc.w,
		h:                 cc.h,
		cursorVisible:     cc.cursorVisible,
		termCursorVisible: cc.termCursorVisible,
		lineWrap:          cc.lineWrap,
		runewise:          cc.runewise,
		mut:               &sync.RWMutex{},
	}
}

// FillBackground changes the background color for each character
func (c *Canvas) FillBackground(bg AttributeColor) {
	converted := bg.Background()
	c.mut.Lock()
	for i := range c.chars {
		c.chars[i].bg = converted
		c.chars[i].drawn = false
	}
	c.mut.Unlock()
}

// Fill changes the foreground color for each character
func (c *Canvas) Fill(fg AttributeColor) {
	c.mut.Lock()
	for i := range c.chars {
		c.chars[i].fg = fg
	}
	c.mut.Unlock()
}

// String returns only the characters, as a long string with a newline after each row
func (c *Canvas) String() string {
	var sb strings.Builder
	c.mut.RLock()
	for y := uint(0); y < c.h; y++ {
		for x := uint(0); x < c.w; x++ {
			cr := &((*c).chars[y*c.w+x])
			if cr.r == rune(0) {
				sb.WriteRune(' ')
			} else {
				sb.WriteRune(cr.r)
			}
		}
		sb.WriteRune('\n')
	}
	c.mut.RUnlock()
	return sb.String()
}

// PlotAll tries to plot each individual rune.
// It's very inefficient and meant to be used as a robust fallback.
func (c *Canvas) PlotAll() {
	w := c.w
	h := c.h
	c.mut.Lock()
	for y := range h {
		for x := int(w - 1); x >= 0; x-- {
			cr := &((*c).chars[y*w+uint(x)])
			if cr.cw == 1 {
				continue // continuation of a wide character
			}
			r := cr.r
			if cr.r == rune(0) {
				r = ' '
			}
			SetXY(uint(x), y)
			fmt.Print(cr.fg.Combine(cr.bg).String() + string(r) + NoColor)
		}
	}
	c.mut.Unlock()
}

// Return the size of the current canvas
func (c *Canvas) Size() (uint, uint) {
	return c.w, c.h
}

// Width returns the canvas width
func (c *Canvas) Width() uint {
	return c.w
}

// Height returns the canvas height
func (c *Canvas) Height() uint {
	return c.h
}

// Clear canvas
func (c *Canvas) Clear() {
	c.mut.Lock()
	defer c.mut.Unlock()
	for i := range c.chars {
		c.chars[i].r = rune(0)
		c.chars[i].drawn = false
	}
}

// SetLineWrap enables or disables line wrapping
func (c *Canvas) SetLineWrap(enable bool) {
	c.mut.Lock()
	defer c.mut.Unlock()
	c.lineWrap = enable
	SetLineWrap(enable)
}

// SetShowCursor sets the cursor visibility
func (c *Canvas) SetShowCursor(enable bool) {
	c.mut.Lock()
	c.cursorVisible = enable
	c.mut.Unlock()
	c.flushCursor()
}

// HideCursor hides the cursor. Redundant calls emit no escape.
func (c *Canvas) HideCursor() {
	c.mut.Lock()
	c.cursorVisible = false
	c.mut.Unlock()
	c.flushCursor()
}

// ShowCursor shows the cursor. Redundant calls emit no escape.
func (c *Canvas) ShowCursor() {
	c.mut.Lock()
	c.cursorVisible = true
	c.mut.Unlock()
	c.flushCursor()
}

// flushCursor emits the cursor escape only when desired != actual state.
// Coalesces redundant calls, e.g. hide→show→hide emits only one escape.
func (c *Canvas) flushCursor() {
	c.mut.Lock()
	desired := c.cursorVisible
	if desired == c.termCursorVisible {
		c.mut.Unlock()
		return
	}
	c.termCursorVisible = desired
	c.mut.Unlock()
	ShowCursor(desired)
}

// SetRunewise enables or disables per-rune rendering
func (c *Canvas) SetRunewise(b bool) {
	c.mut.Lock()
	defer c.mut.Unlock()
	c.runewise = b
}

// W returns the canvas width
func (c *Canvas) W() uint {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.w
}

// H returns the canvas height
func (c *Canvas) H() uint {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.h
}

// DrawAndSetCursor draws the entire canvas and then places the cursor at x,y
func (c *Canvas) DrawAndSetCursor(x, y uint) {
	c.Draw()
	SetXY(x, y)
}

// draw is the shared implementation for Draw and HideCursorAndDraw.
// When permanentlyHideCursor is true, the cursor stays hidden after drawing.
func (c *Canvas) draw(permanentlyHideCursor bool) {
	var (
		lastfg        = Default
		lastbg        = Default
		cr            ColorRune
		oldcr         ColorRune
		sb            strings.Builder
		cursorVisible bool
		lineWrap      bool
		runewise      bool
	)

	cr.fg = Default
	cr.bg = Default
	oldcr.fg = Default
	oldcr.bg = Default

	// NOTE: If too many runes are written to the screen, the contents will scroll up
	// and it will appear as though the first line(s) are lost.

	c.mut.RLock()

	if len((*c).chars) == 0 {
		c.mut.RUnlock()
		return
	}

	firstRun := len(c.oldchars) == 0
	skipAll := !firstRun // assume nothing changed; disproved below
	cursorVisible = c.cursorVisible
	lineWrap = c.lineWrap
	runewise = c.runewise

	size := c.w*c.h - 1
	sb.Grow(int(size))

	if !firstRun {
		for index := range size {
			cr = (*c).chars[index]
			if cr.cw == 1 {
				continue // continuation of a wide character
			}
			oldcr = (*c).oldchars[index]
			// Any difference from the last rendered frame means we cannot skip
			if !cr.fg.Equal(oldcr.fg) || !cr.bg.Equal(oldcr.bg) || cr.r != oldcr.r {
				skipAll = false
			}
			// Only emit a color code when it differs from the previous cell
			if (index == 0) || !lastfg.Equal(cr.fg) || !lastbg.Equal(cr.bg) {
				sb.WriteString(cr.fg.Combine(cr.bg).String())
			}
			if cr.r != 0 {
				sb.WriteRune(cr.r)
			} else {
				sb.WriteByte(' ')
			}
			lastfg = cr.fg
			lastbg = cr.bg
		}
	} else {
		for index := range size {
			cr = (*c).chars[index]
			if cr.cw == 1 {
				continue
			}
			if (index == 0) || !lastfg.Equal(cr.fg) || !lastbg.Equal(cr.bg) {
				sb.WriteString(cr.fg.Combine(cr.bg).String())
			}
			if cr.r != 0 {
				sb.WriteRune(cr.r)
			} else {
				sb.WriteByte(' ')
			}
			lastfg = cr.fg
			lastbg = cr.bg
		}
	}

	c.mut.RUnlock()

	if skipAll {
		return
	}

	// Hide cursor before drawing, then flush once
	reEnableCursor := false
	c.mut.Lock()
	if permanentlyHideCursor {
		c.cursorVisible = false
	} else if cursorVisible {
		c.cursorVisible = false
		reEnableCursor = true
	}
	c.mut.Unlock()
	c.flushCursor()

	// Temporarily enable line wrap if needed so the full screen write works
	reDisableLineWrap := false
	if !lineWrap {
		c.SetLineWrap(true)
		reDisableLineWrap = true
	}

	if runewise {
		Clear()
		c.PlotAll()
	} else {
		c.mut.Lock()
		SetXY(0, 0)
		if !writeAllToStdout([]byte(sb.String())) {
			c.mut.Unlock()
			return
		}
		c.mut.Unlock()
	}

	// Restore cursor if it was only temporarily hidden
	if reEnableCursor {
		c.mut.Lock()
		c.cursorVisible = true
		c.mut.Unlock()
		c.flushCursor()
	}

	if reDisableLineWrap {
		c.SetLineWrap(false)
	}

	// Save the current state to oldchars
	c.mut.Lock()
	if lc := len(c.chars); len(c.oldchars) != lc {
		c.oldchars = make([]ColorRune, lc)
	}
	copy(c.oldchars, c.chars)
	c.mut.Unlock()
}

// Draw the entire canvas
func (c *Canvas) Draw() {
	c.draw(false)
}

// HideCursorAndDraw hides the cursor and draws the entire canvas
func (c *Canvas) HideCursorAndDraw() {
	c.draw(true)
}

// Redraw marks all cells dirty and re-renders
func (c *Canvas) Redraw() {
	c.mut.Lock()
	for i := range c.chars {
		c.chars[i].drawn = false
	}
	c.mut.Unlock()
	c.draw(false)
}

// HideCursorAndRedraw marks all cells dirty, hides the cursor, and re-renders
func (c *Canvas) HideCursorAndRedraw() {
	c.mut.Lock()
	for i := range c.chars {
		c.chars[i].drawn = false
	}
	c.mut.Unlock()
	c.draw(true)
}

// RedrawFull forces a full-frame redraw by discarding the previous frame
func (c *Canvas) RedrawFull() {
	c.mut.Lock()
	for i := range c.chars {
		c.chars[i].drawn = false
	}
	c.oldchars = nil
	c.mut.Unlock()
	c.draw(false)
}

// HideCursorAndRedrawFull hides the cursor and forces a full-frame redraw
func (c *Canvas) HideCursorAndRedrawFull() {
	c.mut.Lock()
	for i := range c.chars {
		c.chars[i].drawn = false
	}
	c.oldchars = nil
	c.mut.Unlock()
	c.draw(true)
}

// At returns the rune at the given coordinates, or an error if out of bounds
func (c *Canvas) At(x, y uint) (rune, error) {
	c.mut.RLock()
	defer c.mut.RUnlock()
	chars := (*c).chars
	index := y*c.w + x
	if index >= uint(len(chars)) {
		return rune(0), errors.New("out of bounds")
	}
	return chars[index].r, nil
}

// Plot sets the rune at (x, y) and marks the cell as undrawn
func (c *Canvas) Plot(x, y uint, r rune) {
	if x >= c.w || y >= c.h {
		return
	}
	index := y*c.w + x
	c.mut.Lock()
	chars := (*c).chars
	chars[index].r = r
	chars[index].drawn = false
	c.mut.Unlock()
}

// PlotColor sets the rune and foreground color at (x, y)
func (c *Canvas) PlotColor(x, y uint, fg AttributeColor, r rune) {
	if x >= c.w || y >= c.h {
		return
	}
	index := y*c.w + x
	c.mut.Lock()
	chars := (*c).chars
	chars[index].r = r
	chars[index].fg = fg
	chars[index].drawn = false
	c.mut.Unlock()
}

// Write is an alias for WriteString, for backwards compatibility
func (c *Canvas) Write(x, y uint, fg, bg AttributeColor, s string) {
	c.WriteString(x, y, fg, bg, s)
}

// WriteString will write a string to the canvas
func (c *Canvas) WriteString(x, y uint, fg, bg AttributeColor, s string) {
	if x >= c.w || y >= c.h {
		return
	}
	c.mut.RLock()
	chars := (*c).chars
	counter := uint(0)
	startpos := y*c.w + x
	lchars := uint(len(chars))
	c.mut.RUnlock()
	bgb := bg.Background()
	for _, r := range s {
		i := startpos + counter
		if i >= lchars {
			break
		}
		c.mut.Lock()
		chars[i].r = r
		chars[i].fg = fg
		chars[i].bg = bgb
		chars[i].drawn = false
		c.mut.Unlock()
		counter++
	}
}

// WriteRune will write a colored rune to the canvas
func (c *Canvas) WriteRune(x, y uint, fg, bg AttributeColor, r rune) {
	if x >= c.w || y >= c.h {
		return
	}
	index := y*c.w + x
	c.mut.Lock()
	defer c.mut.Unlock()
	chars := (*c).chars
	chars[index].r = r
	chars[index].fg = fg
	chars[index].bg = bg.Background()
	chars[index].drawn = false
}

// WriteRuneB will write a colored rune to the canvas.
// The x and y must be within range (x < c.w and y < c.h).
func (c *Canvas) WriteRuneB(x, y uint, fg, bgb AttributeColor, r rune) {
	index := y*c.w + x
	c.mut.Lock()
	defer c.mut.Unlock()
	(*c).chars[index] = ColorRune{fg, bgb, r, false, 0}
}

// WriteRuneBNoLock will write a colored rune to the canvas.
// The x and y must be within range (x < c.w and y < c.h).
// The canvas mutex is not locked.
func (c *Canvas) WriteRuneBNoLock(x, y uint, fg, bgb AttributeColor, r rune) {
	(*c).chars[y*c.w+x] = ColorRune{fg, bgb, r, false, 0}
}

// WriteWideRuneB writes a double-width (CJK) rune to the canvas.
// The next cell (x+1) is marked as a continuation cell and skipped during drawing.
// The x and y must be within range (x+1 < c.w and y < c.h).
func (c *Canvas) WriteWideRuneB(x, y uint, fg, bgb AttributeColor, r rune) {
	base := y*c.w + x
	c.mut.Lock()
	defer c.mut.Unlock()
	(*c).chars[base] = ColorRune{fg, bgb, r, false, 2}
	(*c).chars[base+1] = ColorRune{fg, bgb, 0, false, 1}
}

// WriteWideRuneBNoLock writes a double-width (CJK) rune to the canvas without locking.
// The next cell (x+1) is marked as a continuation cell and skipped during drawing.
// The x and y must be within range (x+1 < c.w and y < c.h).
func (c *Canvas) WriteWideRuneBNoLock(x, y uint, fg, bgb AttributeColor, r rune) {
	base := y*c.w + x
	(*c).chars[base] = ColorRune{fg, bgb, r, false, 2}
	(*c).chars[base+1] = ColorRune{fg, bgb, 0, false, 1}
}

// WriteBackground sets the background color at (x, y)
func (c *Canvas) WriteBackground(x, y uint, bg AttributeColor) {
	index := y*c.w + x
	c.mut.Lock()
	defer c.mut.Unlock()
	(*c).chars[index].bg = bg
	(*c).chars[index].drawn = false
}

// WriteBackgroundAddRuneIfEmpty sets the background color at (x, y) and writes r if the cell is empty
func (c *Canvas) WriteBackgroundAddRuneIfEmpty(x, y uint, bg AttributeColor, r rune) {
	index := y*c.w + x
	c.mut.Lock()
	defer c.mut.Unlock()
	(*c).chars[index].bg = bg
	if (*c).chars[index].r == 0 {
		(*c).chars[index].r = r
	}
	(*c).chars[index].drawn = false
}

// WriteBackgroundNoLock sets the background color at (x, y) without locking
func (c *Canvas) WriteBackgroundNoLock(x, y uint, bg AttributeColor) {
	index := y*c.w + x
	(*c).chars[index].bg = bg
	(*c).chars[index].drawn = false
}

// Lock the canvas mutex
func (c *Canvas) Lock() {
	c.mut.Lock()
}

// Unlock the canvas mutex
func (c *Canvas) Unlock() {
	c.mut.Unlock()
}

// WriteRunesB fills count cells starting at (x, y) with the given colored rune
func (c *Canvas) WriteRunesB(x, y uint, fg, bgb AttributeColor, r rune, count uint) {
	startIndex := y*c.w + x
	afterLastIndex := startIndex + count
	c.mut.Lock()
	chars := (*c).chars
	for i := startIndex; i < afterLastIndex; i++ {
		chars[i] = ColorRune{fg, bgb, r, false, 0}
	}
	c.mut.Unlock()
}

// Resize adjusts the canvas to the current terminal size, discarding old content
func (c *Canvas) Resize() {
	w, h := MustTermSize()
	c.mut.Lock()
	defer c.mut.Unlock()
	if (w != c.w) || (h != c.h) {
		c.w = w
		c.h = h
		c.chars = make([]ColorRune, w*h)
		c.oldchars = nil
	}
}

// Resized checks if the terminal was resized and returns a new Canvas if so.
// Returns nil if the size has not changed.
func (c *Canvas) Resized() *Canvas {
	w, h := MustTermSize()
	if (w != c.w) || (h != c.h) {
		// The terminal was resized!
		oldc := c

		nc := &Canvas{}
		nc.w = w
		nc.h = h
		nc.chars = make([]ColorRune, w*h)
		nc.mut = &sync.RWMutex{}

		nc.mut.Lock()
		c.mut.Lock()
		defer c.mut.Unlock()
		defer nc.mut.Unlock()
	OUT:
		// Copy over old characters, marking them as not yet drawn
		for y := uint(0); y < umin(oldc.h, h); y++ {
			for x := uint(0); x < umin(oldc.w, w); x++ {
				oldIndex := y*oldc.w + x
				index := y*nc.w + x
				if oldIndex > index {
					break OUT
				}
				cr := oldc.chars[oldIndex]
				cr.drawn = false
				nc.chars[index] = cr
			}
		}
		return nc
	}
	return nil
}
