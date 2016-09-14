// Package colors strings using ANSI escape sequences.
package ansicolor

import (
	"strconv"
)

// Color functions
var (
	Clear            = newFunc(0)
	Reset            = Clear
	Bold             = newFunc(1)
	Dark             = newFunc(2)
	Italic           = newFunc(3)
	Underline        = newFunc(4)
	Blink            = newFunc(5)
	RapidBlink       = newFunc(6)
	Negative         = newFunc(7)
	Concealed        = newFunc(8)
	StrikeThrough    = newFunc(9)
	Black            = newFunc(30)
	Red              = newFunc(31)
	Green            = newFunc(32)
	Yellow           = newFunc(33)
	Blue             = newFunc(34)
	Magenta          = newFunc(35)
	Cyan             = newFunc(36)
	White            = newFunc(37)
	OnBlack          = newFunc(40)
	OnRed            = newFunc(41)
	OnGreen          = newFunc(42)
	OnYellow         = newFunc(43)
	OnBlue           = newFunc(44)
	OnMagenta        = newFunc(45)
	OnCyan           = newFunc(46)
	OnWhite          = newFunc(47)
	IntenseBlack     = newFunc(90)
	IntenseRed       = newFunc(91)
	IntenseGreen     = newFunc(92)
	IntenseYellow    = newFunc(93)
	IntenseBlue      = newFunc(94)
	IntenseMagenta   = newFunc(95)
	IntenseCyan      = newFunc(96)
	IntenseWhite     = newFunc(97)
	OnIntenseBlack   = newFunc(100)
	OnIntenseRed     = newFunc(101)
	OnIntenseGreen   = newFunc(102)
	OnIntenseYellow  = newFunc(103)
	OnIntenseBlue    = newFunc(104)
	OnIntenseMagenta = newFunc(105)
	OnIntenseCyan    = newFunc(106)
	OnIntenseWhite   = newFunc(107)
)

func newFunc(colorCode int) func(string) string {
	return func(text string) string {
		result := ""
		result += "\x1b[" + strconv.Itoa(colorCode) + "m"
		result += text
		result += "\x1b[0m"
		return result
	}
}
