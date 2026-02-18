package vt

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync"
)

const (
	cursorHome         = "\033[H"
	cursorHomeTemplate = "\033[%d;%dH"
	cursorUp           = "\033[A"
	cursorDown         = "\033[B"
	cursorForward      = "\033[C"
	cursorBackward     = "\033[D"
	saveCursor         = "\033[s"
	restoreCursor      = "\033[u"
	saveCursorAttrs    = "\033[7"
	restoreCursorAttrs = "\033[8"
	resetDevice        = "\033c"
	eraseScreen        = "\033[2J"
	eraseEndOfLine     = "\033[K"
	eraseStartOfLine   = "\033[1K"
	eraseLine          = "\033[2K"
	eraseDown          = "\033[J"
	eraseUp            = "\033[1J"
	enableLineWrap     = "\033[?7h"
	disableLineWrap    = "\033[?7l"
	showCursor         = "\033[?25h"
	hideCursor         = "\033[?25l"
	echoOff            = "\033[12h"
	attributeTemplate  = "\033[%sm"
)

type AttributeColor uint32

type ColorRune struct {
	fg    AttributeColor
	bg    AttributeColor
	r     rune // The character to draw
	drawn bool // Has been drawn to screen yet?
}

// for API stability
type Char ColorRune
type Canvas struct {
	mut           *sync.RWMutex
	chars         []ColorRune
	oldchars      []ColorRune
	w             uint
	h             uint
	cursorVisible bool
	lineWrap      bool
	runewise      bool
}

// canvasCopy is a Canvas without the mutex
type canvasCopy struct {
	chars         []ColorRune
	oldchars      []ColorRune
	w             uint
	h             uint
	cursorVisible bool
	lineWrap      bool
	runewise      bool
}

const (
	// Non-color attributes
	ResetAll   AttributeColor = 0
	Bright     AttributeColor = 1
	Dim        AttributeColor = 2
	Underscore AttributeColor = 4
	Blink      AttributeColor = 5
	Reverse    AttributeColor = 7
	Hidden     AttributeColor = 8
	None       AttributeColor = 0

	Black     AttributeColor = 30
	Red       AttributeColor = 31
	Green     AttributeColor = 32
	Yellow    AttributeColor = 33
	Blue      AttributeColor = 34
	Magenta   AttributeColor = 35
	Cyan      AttributeColor = 36
	LightGray AttributeColor = 37

	DarkGray     AttributeColor = 90
	LightRed     AttributeColor = 91
	LightGreen   AttributeColor = 92
	LightYellow  AttributeColor = 93
	LightBlue    AttributeColor = 94
	LightMagenta AttributeColor = 95
	LightCyan    AttributeColor = 96
	White        AttributeColor = 97

	BackgroundBlack     AttributeColor = 40
	BackgroundRed       AttributeColor = 41
	BackgroundGreen     AttributeColor = 42
	BackgroundYellow    AttributeColor = 43
	BackgroundBlue      AttributeColor = 44
	BackgroundMagenta   AttributeColor = 45
	BackgroundCyan      AttributeColor = 46
	BackgroundLightGray AttributeColor = 47

	Default           AttributeColor = 39
	DefaultBackground AttributeColor = 49
)

var (
	Pink              = LightMagenta
	Gray              = DarkGray
	BackgroundWhite   = BackgroundLightGray
	BackgroundGray    = BackgroundLightGray
	BackgroundDefault = DefaultBackground
)

var (
	DarkColorMap = map[string]AttributeColor{
		"black":        Black,
		"Black":        Black,
		"red":          Red,
		"Red":          Red,
		"green":        Green,
		"Green":        Green,
		"yellow":       Yellow,
		"Yellow":       Yellow,
		"blue":         Blue,
		"Blue":         Blue,
		"magenta":      Magenta,
		"Magenta":      Magenta,
		"cyan":         Cyan,
		"Cyan":         Cyan,
		"gray":         DarkGray,
		"Gray":         DarkGray,
		"white":        LightGray,
		"White":        LightGray,
		"lightwhite":   White,
		"LightWhite":   White,
		"darkred":      Red,
		"DarkRed":      Red,
		"darkgreen":    Green,
		"DarkGreen":    Green,
		"darkyellow":   Yellow,
		"DarkYellow":   Yellow,
		"darkblue":     Blue,
		"DarkBlue":     Blue,
		"darkmagenta":  Magenta,
		"DarkMagenta":  Magenta,
		"darkcyan":     Cyan,
		"DarkCyan":     Cyan,
		"darkgray":     DarkGray,
		"DarkGray":     DarkGray,
		"lightred":     LightRed,
		"LightRed":     LightRed,
		"lightgreen":   LightGreen,
		"LightGreen":   LightGreen,
		"lightyellow":  LightYellow,
		"LightYellow":  LightYellow,
		"lightblue":    LightBlue,
		"LightBlue":    LightBlue,
		"lightmagenta": LightMagenta,
		"LightMagenta": LightMagenta,
		"lightcyan":    LightCyan,
		"LightCyan":    LightCyan,
		"lightgray":    LightGray,
		"LightGray":    LightGray,
	}
	LightColorMap = map[string]AttributeColor{
		"black":        Black,
		"Black":        Black,
		"red":          LightRed,
		"Red":          LightRed,
		"green":        LightGreen,
		"Green":        LightGreen,
		"yellow":       LightYellow,
		"Yellow":       LightYellow,
		"blue":         LightBlue,
		"Blue":         LightBlue,
		"magenta":      LightMagenta,
		"Magenta":      LightMagenta,
		"cyan":         LightCyan,
		"Cyan":         LightCyan,
		"gray":         LightGray,
		"Gray":         LightGray,
		"white":        White,
		"White":        White,
		"lightwhite":   White,
		"LightWhite":   White,
		"lightred":     LightRed,
		"LightRed":     LightRed,
		"lightgreen":   LightGreen,
		"LightGreen":   LightGreen,
		"lightyellow":  LightYellow,
		"LightYellow":  LightYellow,
		"lightblue":    LightBlue,
		"LightBlue":    LightBlue,
		"lightmagenta": LightMagenta,
		"LightMagenta": LightMagenta,
		"lightcyan":    LightCyan,
		"LightCyan":    LightCyan,
		"lightgray":    LightGray,
		"LightGray":    LightGray,
		"darkred":      Red,
		"DarkRed":      Red,
		"darkgreen":    Green,
		"DarkGreen":    Green,
		"darkyellow":   Yellow,
		"DarkYellow":   Yellow,
		"darkblue":     Blue,
		"DarkBlue":     Blue,
		"darkmagenta":  Magenta,
		"DarkMagenta":  Magenta,
		"darkcyan":     Cyan,
		"DarkCyan":     Cyan,
		"darkgray":     DarkGray,
		"DarkGray":     DarkGray,
	}

	scache sync.Map
)

func SetXYDirect(x, y uint) {
	fmt.Printf(cursorHomeTemplate, y+1, x+1)
}

func Home() {
	fmt.Print(cursorHome)
}

func Reset() {
	fmt.Print(resetDevice)
}

func Clear() {
	fmt.Print(eraseScreen)
}

const NoColor string = "\033[0m"

func SetNoColor() {
	fmt.Print(NoColor)
}

func Init() {
	initTerminal()
	Reset()
	Clear()
	ShowCursor(false)
	SetLineWrap(false)
	EchoOff()
}

func Close() {
	SetLineWrap(true)
	ShowCursor(true)
	Home()
}

func EchoOff() {
	if echoOffHelper() {
		fmt.Print(echoOff)
	}
}

func SetLineWrap(enable bool) {
	if enable {
		fmt.Print(enableLineWrap)
	} else {
		fmt.Print(disableLineWrap)
	}
}

func ShowCursor(enable bool) {
	showCursorHelper(enable)
	if enable {
		fmt.Print(showCursor)
	} else {
		fmt.Print(hideCursor)
	}
}

// GetBackgroundColor prints a code to the terminal emulator,
// reads the results and tries to interpret it as the RGB background color.
// Returns three float64 values, and possibly an error value.
func GetBackgroundColor(tty *TTY) (float64, float64, float64, error) {
	// First try the escape code used by ie. alacritty
	if err := tty.WriteString("\033]11;?\a"); err != nil {
		return 0, 0, 0, err
	}
	result, err := tty.ReadString()
	if err != nil {
		return 0, 0, 0, err
	}
	if !strings.Contains(result, "rgb:") {
		// Then try the escape code used by ie. gnome-terminal
		if err := tty.WriteString("\033]10;?\a\033]11;?\a"); err != nil {
			return 0, 0, 0, err
		}
		result, err = tty.ReadString()
		if err != nil {
			return 0, 0, 0, err
		}
	}
	if _, after, ok := strings.Cut(result, "rgb:"); ok {
		rgb := after
		if strings.Count(rgb, "/") == 2 {
			parts := strings.SplitN(rgb, "/", 3)
			if len(parts) == 3 {
				r := new(big.Int)
				r.SetString(parts[0], 16)
				g := new(big.Int)
				g.SetString(parts[1], 16)
				b := new(big.Int)
				b.SetString(parts[2], 16)
				return float64(r.Int64() / 65535.0), float64(g.Int64() / 65535.0), float64(b.Int64() / 65535.0), nil
			}
		}
	}

	return 0, 0, 0, fmt.Errorf("could not read rgb value from terminal emulator, got: %q", result)
}

func (ac AttributeColor) Head() uint32 {
	return uint32(ac) & 0xFF
}

func (ac AttributeColor) Tail() uint32 {
	return uint32(ac) >> 8
}

// Modify color attributes so that they become background color attributes instead
func (ac AttributeColor) Background() AttributeColor {
	val := uint32(ac)
	if val >= 30 && val <= 39 {
		return AttributeColor(val + 10)
	}
	if val >= 40 && val <= 49 {
		return ac
	}
	return ac
}

// Return the VT100 terminal codes for setting this combination of attributes and color attributes
func (ac AttributeColor) String() string {
	val := uint32(ac)

	if cached, ok := scache.Load(val); ok {
		return cached.(string)
	}

	var result string
	if val > 0xFFFF {
		primary := val & 0xFFFF
		secondary := (val >> 16) & 0xFFFF
		result = fmt.Sprintf(attributeTemplate, strconv.FormatUint(uint64(primary), 10)+";"+strconv.FormatUint(uint64(secondary), 10))
	} else {
		result = fmt.Sprintf(attributeTemplate, strconv.FormatUint(uint64(val), 10))
	}

	scache.Store(val, result)
	return result
}

// Get the full string needed for outputting colored texti, with the text and stopping the color attribute
func (ac AttributeColor) StartStop(text string) string {
	return ac.String() + text + NoColor
}

// An alias for StartStop
func (ac AttributeColor) Get(text string) string {
	return ac.String() + text + NoColor
}

// Get the full string needed for outputting colored text, with the text, but don't reset the attributes at the end of the string
func (ac AttributeColor) Start(text string) string {
	return ac.String() + text
}

// Get the text and the terminal codes for resetting the attributes
func (ac AttributeColor) Stop(text string) string {
	return text + NoColor
}

var maybeNoColor *string

// Return a string for resetting the attributes
func Stop() string {
	if maybeNoColor != nil {
		return *maybeNoColor
	}
	s := NoColor
	maybeNoColor = &s
	return s
}

// Use this color to output the given text. Will reset the attributes at the end of the string. Outputs a newline.
func (ac AttributeColor) Output(text string) {
	fmt.Println(ac.Get(text))
}

// Same as output, but outputs to stderr instead of stdout
func (ac AttributeColor) Error(text string) {
	fmt.Fprintln(os.Stderr, ac.Get(text))
}

func (ac AttributeColor) Combine(other AttributeColor) AttributeColor {
	if ac == 0 {
		return other
	}
	if other == 0 {
		return ac
	}

	val1 := uint32(ac) & 0xFFFF
	val2 := uint32(other) & 0xFFFF

	combined := val1 | (val2 << 16)

	return AttributeColor(combined)
}

// Return a new AttributeColor that has "Bright" added to the list of attributes
func (ac AttributeColor) Bright() AttributeColor {
	return ac.Combine(Bright)
}

func (ac AttributeColor) Ints() []int {
	return []int{int(ac)}
}

func (ac *AttributeColor) Equal(other AttributeColor) bool {
	return *ac == other
}

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
	c.lineWrap = false
	c.SetShowCursor(c.cursorVisible)
	c.SetLineWrap(c.lineWrap)
	return c
}

// Copy creates a new Canvas struct that is a copy of this one.
// The mutex is initialized as a new mutex.
func (c *Canvas) Copy() Canvas {
	c.mut.RLock()
	defer c.mut.RUnlock()

	cc := canvasCopy{
		chars:         make([]ColorRune, len(c.chars)),
		oldchars:      make([]ColorRune, len(c.oldchars)),
		w:             c.w,
		h:             c.h,
		cursorVisible: c.cursorVisible,
		lineWrap:      c.lineWrap,
		runewise:      c.runewise,
	}
	copy(cc.chars, c.chars)
	copy(cc.oldchars, c.oldchars)

	return Canvas{
		chars:         cc.chars,
		oldchars:      cc.oldchars,
		w:             cc.w,
		h:             cc.h,
		cursorVisible: cc.cursorVisible,
		lineWrap:      cc.lineWrap,
		runewise:      cc.runewise,
		mut:           &sync.RWMutex{},
	}
}

// Change the background color for each character
func (c *Canvas) FillBackground(bg AttributeColor) {
	converted := bg.Background()
	c.mut.Lock()
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
			r := cr.r
			if cr.r == rune(0) {
				r = ' '
				//continue
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

func (c *Canvas) Width() uint {
	return c.w
}

func (c *Canvas) Height() uint {
	return c.h
}

// Move cursor to the given position (0,0 is top left)
func SetXY(x, y uint) {
	SetXYDirect(x, y)
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

func (c *Canvas) SetLineWrap(enable bool) {
	c.mut.Lock()
	defer c.mut.Unlock()
	SetLineWrap(enable)
}

func (c *Canvas) SetShowCursor(enable bool) {
	c.mut.Lock()
	defer c.mut.Unlock()
	c.cursorVisible = enable
	ShowCursor(enable)
}

func (c *Canvas) W() uint {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.w
}

func (c *Canvas) H() uint {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.h
}

func (c *Canvas) HideCursor() {
	c.SetShowCursor(false)
}

func (c *Canvas) ShowCursor() {
	c.SetShowCursor(true)
}

func (c *Canvas) SetRunewise(b bool) {
	c.mut.Lock()
	defer c.mut.Unlock()
	c.runewise = b
}

// DrawAndSetCursor draws the entire canvas and then places the cursor at x,y
func (c *Canvas) DrawAndSetCursor(x, y uint) {
	c.Draw()
	// Reposition the cursor
	SetXY(x, y)
}

// HideCursorAndDraw will hide the cursor and then draw the entire canvas
func (c *Canvas) HideCursorAndDraw() {

	c.cursorVisible = false
	c.SetShowCursor(false)

	var (
		lastfg = Default // AttributeColor
		lastbg = Default // AttributeColor
		cr     ColorRune
		oldcr  ColorRune
		sb     strings.Builder
	)

	cr.fg = Default
	cr.bg = Default
	oldcr.fg = Default
	oldcr.bg = Default

	// NOTE: If too many runes are written to the screen, the contents will scroll up,
	// and it will appear like the first line(s) are lost!

	c.mut.RLock()

	if len((*c).chars) == 0 {
		c.mut.RUnlock()
		return
	}

	firstRun := len(c.oldchars) == 0
	skipAll := !firstRun // true by default, except for the first run

	size := c.w*c.h - 1
	sb.Grow(int(size))

	if !firstRun {
		for index := range size {
			cr = (*c).chars[index]
			oldcr = (*c).oldchars[index]
			if cr.fg.Equal(lastfg) && cr.fg.Equal(oldcr.fg) && cr.bg.Equal(lastbg) && cr.bg.Equal(oldcr.bg) && cr.r == oldcr.r {
				// One is not skippable, can not skip all
				skipAll = false
			}
			// Only output a color code if it's different from the last character, or it's the first one
			if (index == 0) || !lastfg.Equal(cr.fg) || !lastbg.Equal(cr.bg) {
				// Write to the string builder
				sb.WriteString(cr.fg.Combine(cr.bg).String())
			}
			// Write the character
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
			// Only output a color code if it's different from the last character, or it's the first one
			if (index == 0) || !lastfg.Equal(cr.fg) || !lastbg.Equal(cr.bg) {
				// Write to the string builder
				sb.WriteString(cr.fg.Combine(cr.bg).String())
			}
			// Write the character
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

	// The screenfull so far is correct (sb.String())

	if skipAll {
		return
	}

	// Enable line wrap, temporarily, if it's disabled
	reDisableLineWrap := false
	if !c.lineWrap {
		c.SetLineWrap(true)
		reDisableLineWrap = true
	}

	// Draw each and every line, or push one large string to screen?
	if c.runewise {

		Clear()
		c.PlotAll()

	} else {
		c.mut.Lock()
		SetXY(0, 0)
		os.Stdout.Write([]byte(sb.String()))
		c.mut.Unlock()
	}

	// Restore the line wrap, if it was temporarily enabled
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
	var (
		lastfg = Default // AttributeColor
		lastbg = Default // AttributeColor
		cr     ColorRune
		oldcr  ColorRune
		sb     strings.Builder
	)

	cr.fg = Default
	cr.bg = Default
	oldcr.fg = Default
	oldcr.bg = Default

	// NOTE: If too many runes are written to the screen, the contents will scroll up,
	// and it will appear like the first line(s) are lost!

	c.mut.RLock()

	if len((*c).chars) == 0 {
		c.mut.RUnlock()
		return
	}

	firstRun := len(c.oldchars) == 0
	skipAll := !firstRun // true by default, except for the first run

	size := c.w*c.h - 1
	sb.Grow(int(size))

	if !firstRun {
		for index := range size {
			cr = (*c).chars[index]
			oldcr = (*c).oldchars[index]
			if cr.fg.Equal(lastfg) && cr.fg.Equal(oldcr.fg) && cr.bg.Equal(lastbg) && cr.bg.Equal(oldcr.bg) && cr.r == oldcr.r {
				// One is not skippable, can not skip all
				skipAll = false
			}
			// Only output a color code if it's different from the last character, or it's the first one
			if (index == 0) || !lastfg.Equal(cr.fg) || !lastbg.Equal(cr.bg) {
				// Write to the string builder
				sb.WriteString(cr.fg.Combine(cr.bg).String())
			}
			// Write the character
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
			// Only output a color code if it's different from the last character, or it's the first one
			if (index == 0) || !lastfg.Equal(cr.fg) || !lastbg.Equal(cr.bg) {
				// Write to the string builder
				sb.WriteString(cr.fg.Combine(cr.bg).String())
			}
			// Write the character
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

	// The screenfull so far is correct (sb.String())

	if skipAll {
		return
	}

	// Output the combined string, also disable the color codes

	// Hide the cursor, temporarily, if it's visible
	reEnableCursor := false
	if c.cursorVisible {
		c.SetShowCursor(false)
		reEnableCursor = true
	}

	// Enable line wrap, temporarily, if it's disabled
	reDisableLineWrap := false
	if !c.lineWrap {
		c.SetLineWrap(true)
		reDisableLineWrap = true
	}

	// Draw each and every line, or push one large string to screen?
	if c.runewise {

		Clear()
		c.PlotAll()

	} else {
		c.mut.Lock()
		SetXY(0, 0)
		os.Stdout.Write([]byte(sb.String()))
		c.mut.Unlock()
	}

	// Restore the cursor, if it was temporarily hidden
	if reEnableCursor {
		c.SetShowCursor(true)
	}

	// Restore the line wrap, if it was temporarily enabled
	if reDisableLineWrap {
		c.SetLineWrap(false)
	}

	// Save the current state to oldchars
	c.mut.Lock()
	c.oldchars = make([]ColorRune, len(c.chars))
	copy(c.oldchars, c.chars)
	c.mut.Unlock()
}

func (c *Canvas) Redraw() {
	c.mut.Lock()
	for i := range c.chars {
		c.chars[i].drawn = false
	}
	c.mut.Unlock()
	c.Draw()
}

func (c *Canvas) HideCursorAndRedraw() {
	c.mut.Lock()
	for i := range c.chars {
		c.chars[i].drawn = false
	}
	c.mut.Unlock()
	c.HideCursorAndDraw()
}

// At returns the rune at the given coordinates, or an error if out of bounds
func (c *Canvas) At(x, y uint) (rune, error) {
	c.mut.RLock()
	defer c.mut.RUnlock()
	chars := (*c).chars
	index := y*c.w + x
	if index < uint(0) || index >= uint(len(chars)) {
		return rune(0), errors.New("out of bounds")
	}
	return chars[index].r, nil
}

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

// WriteString will write a string to the canvas.
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

func (c *Canvas) Write(x, y uint, fg, bg AttributeColor, s string) {
	c.WriteString(x, y, fg, bg, s)
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

// WriteRuneB will write a colored rune to the canvas
// The x and y must be within range (x < c.w and y < c.h)
func (c *Canvas) WriteRuneB(x, y uint, fg, bgb AttributeColor, r rune) {
	index := y*c.w + x
	c.mut.Lock()
	defer c.mut.Unlock()
	(*c).chars[index] = ColorRune{fg, bgb, r, false}
}

// WriteRuneBNoLock will write a colored rune to the canvas
// The x and y must be within range (x < c.w and y < c.h)
// The canvas mutex is not locked
func (c *Canvas) WriteRuneBNoLock(x, y uint, fg, bgb AttributeColor, r rune) {
	(*c).chars[y*c.w+x] = ColorRune{fg, bgb, r, false}
}

// WriteBackground will write a background color to the canvas
// The x and y must be within range (x < c.w and y < c.h)
func (c *Canvas) WriteBackground(x, y uint, bg AttributeColor) {
	index := y*c.w + x
	c.mut.Lock()
	defer c.mut.Unlock()
	(*c).chars[index].bg = bg
	(*c).chars[index].drawn = false
}

// WriteBackgroundAddRuneIfEmpty will write a background color to the canvas
// The x and y must be within range (x < c.w and y < c.h)
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

// WriteBackgroundNoLock will write a background color to the canvas
// The x and y must be within range (x < c.w and y < c.h)
// The canvas mutex is not locked
func (c *Canvas) WriteBackgroundNoLock(x, y uint, bg AttributeColor) {
	index := y*c.w + x
	(*c).chars[index].bg = bg
	(*c).chars[index].drawn = false
}

func (c *Canvas) Lock() {
	c.mut.Lock()
}

func (c *Canvas) Unlock() {
	c.mut.Unlock()
}

// WriteRunesB will write repeated colored runes to the canvas.
// This is the same as WriteRuneB, but bg.Background() has already been called on
// the background attribute.
// The x and y must be within range (x < c.w and y < c.h). x + count must be within range too.
func (c *Canvas) WriteRunesB(x, y uint, fg, bgb AttributeColor, r rune, count uint) {
	startIndex := y*c.w + x
	afterLastIndex := startIndex + count
	c.mut.Lock()
	chars := (*c).chars
	for i := startIndex; i < afterLastIndex; i++ {
		chars[i] = ColorRune{fg, bgb, r, false}
	}
	c.mut.Unlock()
}

func (c *Canvas) Resize() {
	w, h := MustTermSize()
	c.mut.Lock()
	if (w != c.w) || (h != c.h) {
		// Resize to the new size
		c.w = w
		c.h = h
		c.chars = make([]ColorRune, w*h)
		c.mut = &sync.RWMutex{}
	}
	c.mut.Unlock()
}

// Check if the canvas was resized, and adjust values accordingly.
// Returns a new canvas, or nil.
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
		// Plot in the old characters
		for y := uint(0); y < umin(oldc.h, h); y++ {
			for x := uint(0); x < umin(oldc.w, w); x++ {
				oldIndex := y*oldc.w + x
				index := y*nc.w + x
				if oldIndex > index {
					break OUT
				}
				// Copy over old characters, and mark them as not drawn
				cr := oldc.chars[oldIndex]
				cr.drawn = false
				nc.chars[index] = cr
			}
		}
		// Return the new canvas
		return nc
	}
	return nil
}
