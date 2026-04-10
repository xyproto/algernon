package vt

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

// AttributeColor represents a terminal color/attribute value
type AttributeColor uint32

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

// Bit flags used in the upper bits of AttributeColor to signal extended modes.
// Layout (uint32):
//
//	bit 31 = 1  → extended (256-color or true-color)
//	bit 30 = 1  → background, 0 = foreground
//	bit 29 = 1  → true-color (24-bit RGB in bits 0–23)
//	bit 29 = 0  → 256-color  (palette index in bits 0–7)
const (
	extendedFlag  = uint32(1 << 31)
	bgFlag        = uint32(1 << 30)
	trueColorFlag = uint32(1 << 29)
)

// DarkColorMap maps color names to AttributeColor values for dark terminals
var DarkColorMap = map[string]AttributeColor{
	"black":        Black,
	"red":          Red,
	"green":        Green,
	"yellow":       Yellow,
	"blue":         Blue,
	"magenta":      Magenta,
	"cyan":         Cyan,
	"gray":         DarkGray,
	"white":        LightGray,
	"lightwhite":   White,
	"darkred":      Red,
	"darkgreen":    Green,
	"darkyellow":   Yellow,
	"darkblue":     Blue,
	"darkmagenta":  Magenta,
	"darkcyan":     Cyan,
	"darkgray":     DarkGray,
	"lightred":     LightRed,
	"lightgreen":   LightGreen,
	"lightyellow":  LightYellow,
	"lightblue":    LightBlue,
	"lightmagenta": LightMagenta,
	"lightcyan":    LightCyan,
	"lightgray":    LightGray,
}

// LightColorMap maps color names to AttributeColor values for light terminals
var LightColorMap = map[string]AttributeColor{
	"black":        Black,
	"red":          LightRed,
	"green":        LightGreen,
	"yellow":       LightYellow,
	"blue":         LightBlue,
	"magenta":      LightMagenta,
	"cyan":         LightCyan,
	"gray":         LightGray,
	"white":        White,
	"lightwhite":   White,
	"lightred":     LightRed,
	"lightgreen":   LightGreen,
	"lightyellow":  LightYellow,
	"lightblue":    LightBlue,
	"lightmagenta": LightMagenta,
	"lightcyan":    LightCyan,
	"lightgray":    LightGray,
	"darkred":      Red,
	"darkgreen":    Green,
	"darkyellow":   Yellow,
	"darkblue":     Blue,
	"darkmagenta":  Magenta,
	"darkcyan":     Cyan,
	"darkgray":     DarkGray,
}

// ansiEscapes holds the pre-computed VT100 escape sequence for every standard
// ANSI attribute code 0–255 (e.g. index 31 → "\033[31m"). All values are
// populated once by init() so String() can return them with a single array
// index and no allocation.
var ansiEscapes [256]string

// extCache caches escape sequences for AttributeColor values outside the 0–255
// range: true-color (bit 31 + bit 29 set), 256-color (bit 31 set), and
// combined two-attribute values (val > 0xFFFF, bit 31 clear).
var extCache sync.Map

func init() {
	for i := range ansiEscapes {
		ansiEscapes[i] = fmt.Sprintf(attributeTemplate, strconv.FormatUint(uint64(i), 10))
	}
}

func (ac AttributeColor) Head() uint32 {
	return uint32(ac) & 0xFF
}

func (ac AttributeColor) Tail() uint32 {
	return uint32(ac) >> 8
}

// Background converts a foreground color to the corresponding background attribute
func (ac AttributeColor) Background() AttributeColor {
	val := uint32(ac)
	if val&extendedFlag != 0 {
		// 256-color or true-color: set the bg flag
		return AttributeColor(val | bgFlag)
	}
	if val >= 30 && val <= 39 {
		return AttributeColor(val + 10)
	}
	if val >= 40 && val <= 49 {
		return ac
	}
	return ac
}

// String returns the VT100 escape sequence for this color/attribute.
// Standard ANSI codes (0–255) are served from a pre-computed array with no
// allocation. Extended values (true-color, 256-color, combined attributes) are
// computed once and memoized in extCache.
func (ac AttributeColor) String() string {
	val := uint32(ac)

	// Fast path: standard ANSI attribute/color codes (the vast majority of calls)
	if val < 256 {
		return ansiEscapes[val]
	}

	if cached, ok := extCache.Load(val); ok {
		return cached.(string)
	}

	var result string
	if val&extendedFlag != 0 {
		isBg := val&bgFlag != 0
		if val&trueColorFlag != 0 {
			// True-color (24-bit RGB): bits 0–23 hold R, G, B
			r := uint8((val >> 16) & 0xFF)
			g := uint8((val >> 8) & 0xFF)
			b := uint8(val & 0xFF)
			if Has256Colors() {
				if isBg {
					result = fmt.Sprintf("\033[48;2;%d;%d;%dm", r, g, b)
				} else {
					result = fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
				}
			} else {
				// Terminal only supports 16 colors: find the nearest ANSI match
				nearest := nearestANSI16(r, g, b)
				if isBg {
					result = nearest.Background().String()
				} else {
					result = nearest.String()
				}
			}
		} else {
			// 256-color mode: bits 0–7 are the palette index
			idx := val & 0xFF
			if isBg {
				result = fmt.Sprintf("\033[48;5;%dm", idx)
			} else {
				result = fmt.Sprintf("\033[38;5;%dm", idx)
			}
		}
	} else if val > 0xFFFF {
		// Combined two-attribute value: primary in bits 0–15, secondary in bits 16–31
		primary := val & 0xFFFF
		secondary := (val >> 16) & 0xFFFF
		result = fmt.Sprintf(attributeTemplate, strconv.FormatUint(uint64(primary), 10)+";"+strconv.FormatUint(uint64(secondary), 10))
	} else {
		// Single attribute code outside 0–255 (uncommon)
		result = fmt.Sprintf(attributeTemplate, strconv.FormatUint(uint64(val), 10))
	}

	extCache.Store(val, result)
	return result
}

// Color256 returns an AttributeColor for the given xterm 256-color foreground index (0–255).
// Use Has256Colors() to check whether the terminal supports this.
func Color256(n uint8) AttributeColor {
	return AttributeColor(uint32(1<<31) | uint32(n))
}

// Background256 returns an AttributeColor for the given xterm 256-color background index (0–255).
// Use Has256Colors() to check whether the terminal supports this.
func Background256(n uint8) AttributeColor {
	return AttributeColor(uint32(1<<31) | uint32(1<<30) | uint32(n))
}

// Wrap returns text wrapped with this color's escape sequence and a trailing reset
func (ac AttributeColor) Wrap(text string) string {
	return ac.String() + text + NoColor
}

// StartStop is an alias for Wrap
func (ac AttributeColor) StartStop(text string) string {
	return ac.Wrap(text)
}

// Get is an alias for Wrap
func (ac AttributeColor) Get(text string) string {
	return ac.Wrap(text)
}

// Start returns the escape sequence followed by text, without resetting
func (ac AttributeColor) Start(text string) string {
	return ac.String() + text
}

// Stop returns text followed by the reset escape sequence
func (ac AttributeColor) Stop(text string) string {
	return text + NoColor
}

// Output prints text with this color to stdout, followed by a newline
func (ac AttributeColor) Output(text string) {
	fmt.Println(ac.Wrap(text))
}

// Error prints text with this color to stderr, followed by a newline
func (ac AttributeColor) Error(text string) {
	fmt.Fprintln(os.Stderr, ac.Wrap(text))
}

// Combine packs two AttributeColor values into one
func (ac AttributeColor) Combine(other AttributeColor) AttributeColor {
	if ac == 0 {
		return other
	}
	if other == 0 {
		return ac
	}

	val1 := uint32(ac) & 0xFFFF
	val2 := uint32(other) & 0xFFFF

	return AttributeColor(val1 | (val2 << 16))
}

// Bright returns a new AttributeColor with the Bright attribute combined in
func (ac AttributeColor) Bright() AttributeColor {
	return ac.Combine(Bright)
}

func (ac AttributeColor) Ints() []int {
	return []int{int(ac)}
}

func (ac *AttributeColor) Equal(other AttributeColor) bool {
	return *ac == other
}

// TrueColor returns a true-color (24-bit) foreground AttributeColor for the given RGB values.
func TrueColor(r, g, b uint8) AttributeColor {
	return AttributeColor(extendedFlag | trueColorFlag | uint32(r)<<16 | uint32(g)<<8 | uint32(b))
}

// TrueBackground returns a true-color (24-bit) background AttributeColor for the given RGB values.
func TrueBackground(r, g, b uint8) AttributeColor {
	return AttributeColor(extendedFlag | trueColorFlag | bgFlag | uint32(r)<<16 | uint32(g)<<8 | uint32(b))
}

// parseHexColor parses a hex color string ("#rrggbb", "#rgb", "rrggbb", or "rgb")
// and returns the red, green, and blue components.
func parseHexColor(s string) (r, g, b uint8, err error) {
	s = strings.TrimPrefix(s, "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return 0, 0, 0, fmt.Errorf("invalid hex color %q: expected #rrggbb or #rgb", "#"+s)
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid hex color %q: %w", "#"+s, err)
	}
	return uint8(v >> 16), uint8((v >> 8) & 0xFF), uint8(v & 0xFF), nil
}

// ansi16Entry pairs an approximate terminal RGB value with its AttributeColor constant.
type ansi16Entry struct {
	r, g, b uint8
	color   AttributeColor
}

// ansi16Palette holds the approximate xterm RGB values for the 16 standard ANSI
// foreground colors (codes 30–37 and 90–97).
var ansi16Palette = [16]ansi16Entry{
	{0, 0, 0, Black},
	{205, 0, 0, Red},
	{0, 205, 0, Green},
	{205, 205, 0, Yellow},
	{0, 0, 238, Blue},
	{205, 0, 205, Magenta},
	{0, 205, 205, Cyan},
	{229, 229, 229, LightGray},
	{127, 127, 127, DarkGray},
	{255, 0, 0, LightRed},
	{0, 255, 0, LightGreen},
	{255, 255, 0, LightYellow},
	{0, 0, 255, LightBlue},
	{255, 0, 255, LightMagenta},
	{0, 255, 255, LightCyan},
	{255, 255, 255, White},
}

// nearestANSI16 returns the 16-color ANSI foreground AttributeColor whose
// approximate terminal RGB value is closest to (r, g, b) by squared
// Euclidean distance.
func nearestANSI16(r, g, b uint8) AttributeColor {
	best := Black
	bestDist := uint32(1<<32 - 1)
	for _, e := range ansi16Palette {
		dr := int32(r) - int32(e.r)
		dg := int32(g) - int32(e.g)
		db := int32(b) - int32(e.b)
		dist := uint32(dr*dr + dg*dg + db*db)
		if dist < bestDist {
			bestDist = dist
			best = e.color
		}
	}
	return best
}

// ColorFromHex parses a hex color string and returns a true-color foreground AttributeColor.
// Accepted formats: "#rrggbb", "#rgb", "rrggbb", "rgb".
func ColorFromHex(s string) (AttributeColor, error) {
	r, g, b, err := parseHexColor(s)
	if err != nil {
		return Default, err
	}
	return TrueColor(r, g, b), nil
}

// BackgroundFromHex parses a hex color string and returns a true-color background AttributeColor.
// Accepted formats: "#rrggbb", "#rgb", "rrggbb", "rgb".
func BackgroundFromHex(s string) (AttributeColor, error) {
	r, g, b, err := parseHexColor(s)
	if err != nil {
		return BackgroundDefault, err
	}
	return TrueBackground(r, g, b), nil
}
