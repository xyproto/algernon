package vt

import (
	"fmt"
	"os"
	"strconv"
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

// scache caches the rendered escape sequences for AttributeColor values
var scache sync.Map

func (ac AttributeColor) Head() uint32 {
	return uint32(ac) & 0xFF
}

func (ac AttributeColor) Tail() uint32 {
	return uint32(ac) >> 8
}

// Background converts a foreground color to the corresponding background attribute
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

// String returns the VT100 escape sequence for this color/attribute
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
