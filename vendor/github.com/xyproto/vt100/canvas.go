package vt100

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type Char struct {
	fg    AttributeColor // Foreground color
	bg    AttributeColor // Background color
	s     rune           // The character to draw
	drawn bool           // Has been drawn to screen yet?
	// Not having a background color, and storing the foreground color as a string is a design choice
}

type Canvas struct {
	w             uint
	h             uint
	chars         []Char
	oldchars      []Char
	mut           *sync.RWMutex
	cursorVisible bool
	lineWrap      bool
}

func NewCanvas() *Canvas {
	var err error
	c := &Canvas{}
	c.w, c.h, err = TermSize()
	if err != nil {
		// Use 80x25 if the size can't be detected
		c.w = 80
		c.h = 25
	}
	c.chars = make([]Char, c.w*c.h)
	c.oldchars = make([]Char, 0, 0)
	c.mut = &sync.RWMutex{}
	c.cursorVisible = false
	ShowCursor(false)
	c.lineWrap = false
	SetLineWrap(false)
	return c
}

// Copy creates a new Canvas struct that is a copy of this one.
// The mutex is kept as a pointer to the original.
func (c *Canvas) Copy() Canvas {
	var c2 Canvas
	c2.w = c.w
	c2.h = c.h
	chars2 := make([]Char, len(c.chars), len(c.chars))
	for i, ch := range c.chars {
		var ch2 Char
		ch2.fg = ch.fg
		ch2.bg = ch.bg
		ch2.s = ch.s
		ch2.drawn = ch.drawn
		chars2[i] = ch
	}
	c2.chars = chars2
	oldchars2 := make([]Char, len(c.chars), len(c.chars))
	for i, ch := range c.oldchars {
		var ch2 Char
		ch2.fg = ch.fg
		ch2.bg = ch.bg
		ch2.s = ch.s
		ch2.drawn = ch.drawn
		oldchars2[i] = ch
	}
	c2.oldchars = oldchars2
	c2.mut = c.mut
	c2.cursorVisible = c.cursorVisible
	c2.lineWrap = c.lineWrap
	return c2
}

// Change the background color for each character
func (c *Canvas) FillBackground(bg AttributeColor) {
	c.mut.Lock()
	converted := bg.Background()
	for i := range c.chars {
		c.chars[i].bg = converted
		c.chars[i].drawn = false
	}
	c.mut.Unlock()
}

// Change the foreground color for each character
func (c *Canvas) Fill(fg AttributeColor) {
	c.mut.Lock()
	for i := range c.chars {
		c.chars[i].fg = fg
	}
	c.mut.Unlock()
}

// Bytes returns only the characters, as a long string with a newline after each row
func (c *Canvas) String() string {
	var sb strings.Builder
	for y := uint(0); y < c.h; y++ {
		c.mut.RLock()
		for x := uint(0); x < c.w; x++ {
			ch := &((*c).chars[y*c.w+x])
			if ch.s == rune(0) {
				sb.WriteRune(' ')
			} else {
				sb.WriteRune(ch.s)
			}
		}
		c.mut.RUnlock()
		sb.WriteRune('\n')
	}
	return sb.String()
}

// Return the size of the current canvas
func (c *Canvas) Size() (uint, uint) {
	return c.w, c.h
}

func (c *Canvas) Width() uint {
	return c.w
}

func (c *Canvas) Height() uint {
	return c.h
}

func umin(a, b uint) uint {
	if a < b {
		return a
	}
	return b
}

// Move cursor to the given position (from 0 and up, the terminal code is from 1 and up)
func SetXY(x, y uint) {
	Set("Cursor Home", map[string]string{"{ROW}": strconv.Itoa(int(y + 1)), "{COLUMN}": strconv.Itoa(int(x + 1))})
}

// Move the cursor down
func Down(n uint) {
	Set("Cursor Down", map[string]string{"{COUNT}": strconv.Itoa(int(n))})
}

// Move the cursor up
func Up(n uint) {
	Set("Cursor Up", map[string]string{"{COUNT}": strconv.Itoa(int(n))})
}

// Move the cursor to the right
func Right(n uint) {
	Set("Cursor Forward", map[string]string{"{COUNT}": strconv.Itoa(int(n))})
}

// Move the cursor to the left
func Left(n uint) {
	Set("Cursor Backward", map[string]string{"{COUNT}": strconv.Itoa(int(n))})
}

func Home() {
	Set("Cursor Home", map[string]string{"{ROW};{COLUMN}": ""})
}

func Reset() {
	Do("Reset Device")
}

// Clear screen
func Clear() {
	Do("Erase Screen")
}

// Clear canvas
func (c *Canvas) Clear() {
	c.mut.Lock()
	for _, ch := range c.chars {
		ch.s = rune(0)
		ch.drawn = false
	}
	c.mut.Unlock()
}

func SetLineWrap(enable bool) {
	if enable {
		Do("Enable Line Wrap")
	} else {
		Do("Disable Line Wrap")
	}
}

func ShowCursor(enable bool) {
	// Thanks https://rosettacode.org/wiki/Terminal_control/Hiding_the_cursor#Escape_code
	if enable {
		fmt.Print("\033[?25h")
	} else {
		fmt.Print("\033[?25l")
	}
}

func (c *Canvas) W() uint {
	return c.w
}

func (c *Canvas) H() uint {
	return c.h
}

func (c *Canvas) ShowCursor() {
	if !c.cursorVisible {
		c.cursorVisible = true
	}
	ShowCursor(true)
}

func (c *Canvas) HideCursor() {
	if c.cursorVisible {
		c.cursorVisible = false
	}
	ShowCursor(false)
}

// Draw the entire canvas
func (c *Canvas) Draw() {
	c.mut.Lock()
	defer c.mut.Unlock()
	var (
		lastfg, lastbg AttributeColor
		ch             *Char
		oldch          *Char
		all            strings.Builder
	)
	firstRun := 0 == len(c.oldchars)
	skipAll := !firstRun // true by default, except for the first run

	for y := uint(0); y < c.h; y++ {
		for x := uint(0); x < c.w; x++ {
			index := y*c.w + x
			ch = &((*c).chars[index])
			if !firstRun {
				oldch = &((*c).oldchars[index])
				if ch.fg.Equal(lastfg) && ch.bg.Equal(lastbg) && ch.fg.Equal(oldch.fg) && ch.bg.Equal(oldch.bg) && ch.s == oldch.s {
					// One is not skippable, can not skip all
					skipAll = false
				}
			}
			// Write this character
			if ch.s == rune(0) || len(string(ch.s)) == 0 {
				// Only output a color code if it's different from the last character, or it's the first one
				if (x == 0 && y == 0) || !lastfg.Equal(ch.fg) || !lastbg.Equal(ch.bg) {
					all.WriteString(ch.fg.Combine(ch.bg).String())
				}
				// Write a blank
				all.WriteRune(' ')
			} else {
				// Only output a color code if it's different from the last character, or it's the first one
				if (x == 0 && y == 0) || !lastfg.Equal(ch.fg) || !lastbg.Equal(ch.bg) {
					all.WriteString(ch.fg.Combine(ch.bg).String())
				}
				// Write the character
				all.WriteRune(ch.s)
			}
			lastfg = ch.fg
			lastbg = ch.bg
		}
	}

	// Output the combined string, also disable the color codes
	if !skipAll {

		// Hide the cursor, temporarily, if it's visible
		if c.cursorVisible {
			ShowCursor(false)
		}
		// Enable line wrap, temporarily, if it's diabled
		if !c.lineWrap {
			SetLineWrap(true)
		}

		all.WriteString(NoColor())
		SetXY(0, 0)
		fmt.Print(all.String())

		// Restore the cursor, if it was temporarily hidden
		if c.cursorVisible {
			ShowCursor(true)
		}
		// Restore the line wrap, if it was temporarily enabled
		if !c.lineWrap {
			SetLineWrap(false)
		}

		// Save the current state to oldchars
		c.oldchars = make([]Char, len(c.chars))
		copy(c.oldchars, c.chars)
	}

}

func (c *Canvas) Redraw() {
	// TODO: Consider using a single for-loop instead of 1 (range) + 2 (x,y)
	c.mut.Lock()
	for _, ch := range c.chars {
		ch.drawn = false
	}
	c.mut.Unlock()
	c.Draw()
}

// At returns the rune at the given coordinates, or an error if out of bounds
func (c *Canvas) At(x, y uint) (rune, error) {
	index := y*c.w + x
	c.mut.RLock()
	defer c.mut.RUnlock()
	chars := (*c).chars
	if index < uint(0) || index >= uint(len(chars)) {
		return rune(0), errors.New("out of bounds")
	}
	return chars[index].s, nil
}

func (c *Canvas) Plot(x, y uint, s rune) {
	if x < 0 || y < 0 {
		return
	}
	if x >= c.w || y >= c.h {
		return
	}
	index := y*c.w + x
	c.mut.Lock()
	chars := (*c).chars
	chars[index].s = s
	chars[index].drawn = false
	c.mut.Unlock()
}

func (c *Canvas) PlotColor(x, y uint, fg AttributeColor, s rune) {
	if x < 0 || y < 0 {
		return
	}
	if x >= c.w || y >= c.h {
		return
	}
	index := y*c.w + x
	c.mut.Lock()
	chars := (*c).chars
	chars[index].s = s
	chars[index].fg = fg
	chars[index].drawn = false
	c.mut.Unlock()
}

// WriteString will write a string to the canvas.
func (c *Canvas) WriteString(x, y uint, fg, bg AttributeColor, s string) {
	if x < 0 || y < 0 {
		return
	}
	if x >= c.w || y >= c.h {
		return
	}
	c.mut.Lock()
	chars := (*c).chars
	counter := uint(0)
	for _, r := range s {
		chars[y*c.w+x+counter].s = r
		chars[y*c.w+x+counter].fg = fg
		chars[y*c.w+x+counter].bg = bg.Background()
		chars[y*c.w+x+counter].drawn = false
		counter++
	}
	c.mut.Unlock()
}

func (c *Canvas) Write(x, y uint, fg, bg AttributeColor, s string) {
	c.WriteString(x, y, fg, bg, s)
}

// WriteRune will write a colored rune to the canvas
func (c *Canvas) WriteRune(x, y uint, fg, bg AttributeColor, r rune) {
	if x < 0 || y < 0 {
		return
	}
	if x >= c.w || y >= c.h {
		return
	}
	index := y*c.w + x
	c.mut.Lock()
	chars := (*c).chars
	chars[index].s = r
	chars[index].fg = fg
	chars[index].bg = bg.Background()
	chars[index].drawn = false
	c.mut.Unlock()
}

func (c *Canvas) Resize() {
	w, h, err := TermSize()
	if err != nil {
		return
	}
	c.mut.Lock()
	if (w != c.w) || (h != c.h) {
		// Resize to the new size
		c.w = w
		c.h = h
		c.chars = make([]Char, w*h)
		c.mut = &sync.RWMutex{}
	}
	c.mut.Unlock()
}

// Check if the canvas was resized, and adjust values accordingly.
// Returns a new canvas, or nil.
func (c *Canvas) Resized() *Canvas {
	w, h, err := TermSize()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	if (w != c.w) || (h != c.h) {
		// The terminal was resized!
		oldc := c

		nc := &Canvas{}
		nc.w = w
		nc.h = h
		nc.chars = make([]Char, w*h)
		nc.mut = &sync.RWMutex{}

		nc.mut.Lock()
		c.mut.Lock()
		defer c.mut.Unlock()
		defer nc.mut.Unlock()
	OUT:
		// Plot in the old characters
		for y := uint(0); y < umin(oldc.h, h); y++ {
			for x := uint(0); x < umin(oldc.w, w); x++ {
				oldIndex := y*oldc.w + x
				index := y*nc.w + x
				if oldIndex > index {
					break OUT
				}
				// Copy over old characters, and mark them as not drawn
				ch := oldc.chars[oldIndex]
				ch.drawn = false
				nc.chars[index] = ch
			}
		}
		// Return the new canvas
		return nc
	}
	return nil
}
