package vt100

import (
	"fmt"
	"image/color"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Color aliases, for ease of use, not for performance

type AttributeColor []byte

var (
	// Non-color attributes
	ResetAll   = NewAttributeColor("Reset all attributes")
	Bright     = NewAttributeColor("Bright")
	Dim        = NewAttributeColor("Dim")
	Underscore = NewAttributeColor("Underscore")
	Blink      = NewAttributeColor("Blink")
	Reverse    = NewAttributeColor("Reverse")
	Hidden     = NewAttributeColor("Hidden")

	None AttributeColor

	// There is also: reset, dim, underscore, reverse and hidden

	// Dark foreground colors (+ light gray)
	Black     = NewAttributeColor("Black")
	Red       = NewAttributeColor("Red")
	Green     = NewAttributeColor("Green")
	Yellow    = NewAttributeColor("Yellow")
	Blue      = NewAttributeColor("Blue")
	Magenta   = NewAttributeColor("Magenta")
	Cyan      = NewAttributeColor("Cyan")
	LightGray = NewAttributeColor("White")

	// Light foreground colors (+ dark gray)
	DarkGray     = NewAttributeColor("90")
	LightRed     = NewAttributeColor("91")
	LightGreen   = NewAttributeColor("92")
	LightYellow  = NewAttributeColor("93")
	LightBlue    = NewAttributeColor("94")
	LightMagenta = NewAttributeColor("95")
	LightCyan    = NewAttributeColor("96")
	White        = NewAttributeColor("97")

	// Aliases
	Pink = LightMagenta
	Gray = DarkGray

	// Dark background colors (+ light gray)
	BackgroundBlack     = NewAttributeColor("40")
	BackgroundRed       = NewAttributeColor("41")
	BackgroundGreen     = NewAttributeColor("42")
	BackgroundYellow    = NewAttributeColor("43")
	BackgroundBlue      = NewAttributeColor("44")
	BackgroundMagenta   = NewAttributeColor("45")
	BackgroundCyan      = NewAttributeColor("46")
	BackgroundLightGray = NewAttributeColor("47")

	// Aliases
	BackgroundWhite = BackgroundLightGray
	BackgroundGray  = BackgroundLightGray

	// Default colors (usually gray)
	Default           = NewAttributeColor("39")
	DefaultBackground = NewAttributeColor("49")
	BackgroundDefault = NewAttributeColor("49")

	// Lookup tables

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
)

func s2b(attribute string) byte {
	switch attribute {
	case "Reset all attributes", "Reset", "reset", "reset all attributes":
		return 0
	case "Bright", "bright":
		return 1
	case "Dim", "dim":
		return 2
	case "Underscore", "underscore":
		return 4
	case "Blink", "blink":
		return 5
	case "Reverse", "reverse":
		return 7
	case "Hidden", "hidden":
		return 8
	case "Black", "black":
		return 30
	case "Red", "red":
		return 31
	case "Green", "green":
		return 32
	case "Yellow", "yellow":
		return 33
	case "Blue", "blue":
		return 34
	case "Magenta", "magenta":
		return 35
	case "Cyan", "cyan":
		return 36
	case "White", "white":
		return 37
	}
	num, err := strconv.Atoi(attribute)
	if err != nil {
		// Not an int and not one of the words above
		return 0
	}
	return byte(num)
}

func NewAttributeColor(attributes ...string) AttributeColor {
	return AttributeColor(mapSB(attributes, s2b))
}

func (ac AttributeColor) Head() byte {
	// no error checking
	return ac[0]
}

func (ac AttributeColor) Tail() []byte {
	// no error checking
	return ac[1:]
}

// Modify color attributes so that they become background color attributes instead
func (ac AttributeColor) Background() AttributeColor {
	newA := make(AttributeColor, 0, len(ac))
	foundOne := false
	for _, attr := range ac {
		if (30 <= attr) && (attr <= 39) {
			// convert foreground color to background color attribute
			newA = append(newA, attr+10)
			foundOne = true
		}
		// skip the rest
	}
	// Did not find a background attribute to convert, keep any existing background attributes
	if !foundOne {
		for _, attr := range ac {
			if (40 <= attr) && (attr <= 49) {
				newA = append(newA, attr)
			}
		}
	}
	return newA
}

func b2s(b byte) string {
	return strconv.Itoa(int(b))
}

var (
	scache = make(map[string]string)
	smut   = &sync.RWMutex{}
)

// Return the VT100 terminal codes for setting this combination of attributes and color attributes
func (ac AttributeColor) String() string {
	id := string(ac)

	smut.RLock()
	if s, has := scache[id]; has {
		smut.RUnlock()
		return s
	}
	smut.RUnlock()

	attributeString := strings.Join(mapBS(ac, b2s), ";")
	// Replace '{attr1};...;{attrn}' with the generated attribute string and return
	s := get(specVT100, "Set Attribute Mode", map[string]string{"{attr1};...;{attrn}": attributeString})

	// Store the value in the cache
	if len(s) > 0 {
		smut.Lock()
		scache[id] = s
		smut.Unlock()
	}

	return s
}

// Get the full string needed for outputting colored texti, with the text and stopping the color attribute
func (ac AttributeColor) StartStop(text string) string {
	return ac.String() + text + NoColor()
}

// An alias for StartStop
func (ac AttributeColor) Get(text string) string {
	return ac.String() + text + NoColor()
}

// Get the full string needed for outputting colored text, with the text, but don't reset the attributes at the end of the string
func (ac AttributeColor) Start(text string) string {
	return ac.String() + text
}

// Get the text and the terminal codes for resetting the attributes
func (ac AttributeColor) Stop(text string) string {
	return text + NoColor()
}

var maybeNoColor *string

// Return a string for resetting the attributes
func Stop() string {
	if maybeNoColor != nil {
		return *maybeNoColor
	}
	s := NoColor()
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
	for _, a1 := range ac {
		a2has := false
		for _, a2 := range other {
			if a1 == a2 {
				a2has = true
				break
			}
		}
		if !a2has {
			other = append(other, a1)
		}
	}
	return AttributeColor(other)
}

// Return a new AttributeColor that has "Bright" added to the list of attributes
func (ac AttributeColor) Bright() AttributeColor {
	return AttributeColor(append(ac, Bright.Head()))
}

// Output a string at x, y with the given colors
func Write(x, y int, text string, fg, bg AttributeColor) {
	SetXY(uint(x), uint(y))
	fmt.Print(fg.Combine(bg).Get(text))
}

// Output a rune at x, y with the given colors
func WriteRune(x, y int, r rune, fg, bg AttributeColor) {
	SetXY(uint(x), uint(y))
	fmt.Print(fg.Combine(bg).Get(string(r)))
}

func (ac AttributeColor) Ints() []int {
	il := make([]int, len(ac))
	for index, b := range ac {
		il[index] = int(b)
	}
	return il
}

// This is not part of the VT100 spec, but an easteregg for displaying 24-bit
// "true color" on some terminals. Example use:
// fmt.Println(vt100.TrueColor(color.RGBA{0xa0, 0xe0, 0xff, 0xff}, "TrueColor"))
func TrueColor(fg color.Color, text string) string {
	c := color.NRGBAModel.Convert(fg).(color.NRGBA)
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", c.R, c.G, c.B, text)
}

// Equal checks if two colors have the same attributes, in the same order.
// The values that are being compared must have at least 1 byte in them.
func (ac *AttributeColor) Equal(other AttributeColor) bool {
	return string(*ac) == string(other)
}
