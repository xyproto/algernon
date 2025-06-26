//go:build windows
// +build windows

package vt

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mgutz/ansi"
	"github.com/xyproto/env/v2"
)

// CharAttribute is a rune and a color attribute
type CharAttribute struct {
	R rune
	A ANSIColor
}

// TextOutput keeps state about verbosity and if colors are enabled
type TextOutput struct {
	color   bool
	enabled bool
	// Tag replacement structs, for performance
	lightReplacer *strings.Replacer
	darkReplacer  *strings.Replacer
}

// Respect the NO_COLOR environment variable
var EnvNoColor = env.Bool("NO_COLOR")

// New creates a new TextOutput struct, which is
// enabled by default and with colors turned on.
// If the NO_COLOR environment variable is set, colors are disabled.
func New() *TextOutput {
	o := &TextOutput{!EnvNoColor, true, nil, nil}
	o.initializeTagReplacers()
	return o
}

// NewTextOutput can initialize a new TextOutput struct,
// which can have colors turned on or off and where the
// output can be enabled (verbose) or disabled (silent).
// If NO_COLOR is set, colors are disabled, regardless.
func NewTextOutput(color, enabled bool) *TextOutput {
	if EnvNoColor {
		color = false
	}
	o := &TextOutput{color, enabled, nil, nil}
	o.initializeTagReplacers()
	return o
}

// OutputTags will output text that may have tags like "<blue>", "</blue>" or "<off>" for
// enabling or disabling color attributes. Respects the color/enabled settings
// of this TextOutput.
func (o *TextOutput) OutputTags(colors ...string) {
	if o.enabled {
		fmt.Println(o.Tags(colors...))
	}
}

// Given a line with words and several color strings, color the words
// in the order of the colors. The last color will color the rest of the
// words.
func (o *TextOutput) OutputWords(line string, colors ...string) {
	if o.enabled {
		fmt.Println(o.Words(line, colors...))
	}
}

// Write a message to stdout if output is enabled
func (o *TextOutput) Println(msg ...interface{}) {
	if o.enabled {
		fmt.Println(o.InterfaceTags(msg...))
	}
}

// Write a message to the given io.Writer if output is enabled
func (o *TextOutput) Fprintln(w io.Writer, msg ...interface{}) {
	if o.enabled {
		fmt.Fprintln(w, o.InterfaceTags(msg...))
	}
}

// Write a message to stdout if output is enabled
func (o *TextOutput) Printf(msg ...interface{}) {
	if !o.enabled {
		return
	}
	count := len(msg)
	if count == 0 {
		return
	} else if count == 1 {
		if fmtString, ok := msg[0].(string); ok {
			fmt.Print(fmtString)
		}
	} else { // > 1
		if fmtString, ok := msg[0].(string); ok {
			fmt.Printf(o.InterfaceTags(fmtString), msg[1:]...)
		} else {
			// fail
			fmt.Printf("%v", msg...)
		}
	}
}

// Write a message to the given io.Writer if output is enabled
func (o *TextOutput) Fprintf(w io.Writer, msg ...interface{}) {
	if !o.enabled {
		return
	}
	count := len(msg)
	if count == 0 {
		return
	} else if count == 1 {
		if fmtString, ok := msg[0].(string); ok {
			fmt.Fprint(w, fmtString)
		}
	} else { // > 1
		if fmtString, ok := msg[0].(string); ok {
			fmt.Fprintf(w, o.InterfaceTags(fmtString), msg[1:]...)
		} else {
			// fail
			fmt.Fprintf(w, "%v", msg...)
		}
	}
}

// Write a message to stdout if output is enabled
func (o *TextOutput) Print(msg ...interface{}) {
	if o.enabled {
		fmt.Print(o.InterfaceTags(msg...))
	}
}

// Write a message to the given io.Writer if output is enabled
func (o *TextOutput) Fprint(w io.Writer, msg ...interface{}) {
	if o.enabled {
		fmt.Fprint(w, o.InterfaceTags(msg...))
	}
}

// Write an error message in red to stderr if output is enabled
func (o *TextOutput) Err(msg string) {
	if o.enabled {
		if o.color {
			fmt.Fprintln(os.Stderr, ansi.Color(msg, "red"))
		} else {
			fmt.Fprintln(os.Stderr, msg)
		}
	}
}

// Enabled returns true if any output is enabled
func (o *TextOutput) Enabled() bool {
	return o.enabled
}

// Disabled returns true if all output is disabled
func (o *TextOutput) Disabled() bool {
	return !o.enabled
}

func (o *TextOutput) DarkRed(s string) string {
	if o.color {
		return ansi.Color(s, "red")
	}
	return s
}

func (o *TextOutput) DarkGreen(s string) string {
	if o.color {
		return ansi.Color(s, "green")
	}
	return s
}

func (o *TextOutput) DarkYellow(s string) string {
	if o.color {
		return ansi.Color(s, "yellow")
	}
	return s
}

func (o *TextOutput) DarkBlue(s string) string {
	if o.color {
		return ansi.Color(s, "blue")
	}
	return s
}

func (o *TextOutput) DarkPurple(s string) string {
	if o.color {
		return ansi.Color(s, "magenta")
	}
	return s
}

func (o *TextOutput) DarkCyan(s string) string {
	if o.color {
		return ansi.Color(s, "cyan")
	}
	return s
}

func (o *TextOutput) DarkGray(s string) string {
	if o.color {
		return ansi.Color(s, "black")
	}
	return s
}

func (o *TextOutput) LightRed(s string) string {
	if o.color {
		return ansi.Color(s, "red+h")
	}
	return s
}

func (o *TextOutput) LightGreen(s string) string {
	if o.color {
		return ansi.Color(s, "green+h")
	}
	return s
}

func (o *TextOutput) LightYellow(s string) string {
	if o.color {
		return ansi.Color(s, "yellow+h")
	}
	return s
}

func (o *TextOutput) LightBlue(s string) string {
	if o.color {
		return ansi.Color(s, "blue+h")
	}
	return s
}

func (o *TextOutput) LightCyan(s string) string {
	if o.color {
		return ansi.Color(s, "cyan+h")
	}
	return s
}

func (o *TextOutput) White(s string) string {
	if o.color {
		return ansi.Color(s, "white+h")
	}
	return s
}

// Given a line with words and several color strings, color the words
// in the order of the colors. The last color will color the rest of the
// words.
func (o *TextOutput) Words(line string, colors ...string) string {
	if o.color {
		var ok bool
		words := strings.Split(line, " ")
		color := ANSIColor(ansi.ColorCode("black+h")) // LightGray
		coloredWords := make([]string, len(words))
		for i, word := range words {
			if i < len(colors) {
				prevColor := color
				color, ok = LightColorMap[colors[i]]
				if !ok {
					// Use the previous color of this color string was not found
					color = prevColor
				}
			}
			coloredWords[i] = ansi.Color(word, string(color))
		}
		return strings.Join(coloredWords, " ")
	}
	return line
}

// Change the color state in the terminal emulator
func (o *TextOutput) ColorOn(attribute1, attribute2 int) string {
	if !o.color {
		return ""
	}
	return fmt.Sprintf("\033[%d;%dm", attribute1, attribute2)
}

// Change the color state in the terminal emulator
func (o *TextOutput) ColorOff() string {
	if !o.color {
		return ""
	}
	return "\033[0m"
}

// Replace <blue> with starting a light blue color attribute and <off> with using the default attributes.
// </blue> can also be used for using the default attributes.
func (o *TextOutput) LightTags(colors ...string) string {
	return o.lightReplacer.Replace(strings.Join(colors, ""))
}

// Same as LightTags
func (o *TextOutput) Tags(colors ...string) string {
	return o.LightTags(colors...)
}

// InterfaceTags is the same as LightTags, but with interfaces
func (o *TextOutput) InterfaceTags(colors ...interface{}) string {
	var sb strings.Builder
	for _, color := range colors {
		if colorString, ok := color.(string); ok {
			sb.WriteString(colorString)
		} else {
			sb.WriteString(fmt.Sprintf("%s", color))
		}
	}
	return o.LightTags(sb.String())
}

// Replace <blue> with starting a light blue color attribute and <off> with using the default attributes.
// </blue> can also be used for using the default attributes.
func (o *TextOutput) DarkTags(colors ...string) string {
	return o.darkReplacer.Replace(strings.Join(colors, ""))
}

func (o *TextOutput) DisableColors() {
	o.color = false
	o.initializeTagReplacers()
}

func (o *TextOutput) EnableColors() {
	o.color = true
	o.initializeTagReplacers()
}

func (o *TextOutput) Disable() {
	o.enabled = false
}

func (o *TextOutput) Enable() {
	o.enabled = true
}

func (o *TextOutput) initializeTagReplacers() {
	// Initialize tag replacement tables, with as few memory allocations as possible (no append)
	off := ansi.ColorCode("off")
	rs := make([]string, len(LightColorMap)*4+2)
	i := 0
	if o.color {
		for key, value := range LightColorMap {
			rs[i] = "<" + key + ">"
			i++
			rs[i] = string(value)
			i++
			rs[i] = "</" + key + ">"
			i++
			rs[i] = off
			i++
		}
		rs[i] = "<off>"
		i++
		rs[i] = off
	} else {
		for key := range LightColorMap {
			rs[i] = "<" + key + ">"
			i++
			rs[i] = ""
			i++
			rs[i] = "</" + key + ">"
			i++
			rs[i] = ""
			i++
		}
		rs[i] = "<off>"
		i++
		rs[i] = ""
	}
	o.lightReplacer = strings.NewReplacer(rs...)
	// Initialize the replacer for the dark color scheme, while reusing the rs slice
	i = 0
	if o.color {
		for key, value := range DarkColorMap {
			rs[i] = "<" + key + ">"
			i++
			rs[i] = string(value)
			i++
			rs[i] = "</" + key + ">"
			i++
			rs[i] = off
			i++
		}
		rs[i] = "<off>"
		i++
		rs[i] = off
	} else {
		for key := range DarkColorMap {
			rs[i] = "<" + key + ">"
			i++
			rs[i] = ""
			i++
			rs[i] = "</" + key + ">"
			i++
			rs[i] = ""
			i++
		}
		rs[i] = "<off>"
		i++
		rs[i] = ""
	}
	o.darkReplacer = strings.NewReplacer(rs...)
}

// // Pair takes a string with ANSI codes and returns
// // a slice with two elements.
// func (o *TextOutput) Extract(s string) []CharAttribute {
// 	var (
// 		escaped      bool
// 		colorcode    strings.Builder
// 		word         strings.Builder
// 		cc           = make([]ANSIColor, 0, len(s))
// 		currentColor ANSIColor
// 	)
// 	for _, r := range s {
// 		if r == '\033' {
// 			escaped = true
// 			if len(word.String()) > 0 {
// 				//fmt.Println("cc", cc)
// 				word.Reset()
// 			}
// 			continue
// 		}
// 		if escaped {
// 			if r != 'm' {
// 				colorcode.WriteRune(r)
// 			} else if r == 'm' {
// 				s2 := strings.TrimPrefix(colorcode.String(), "[")
// 				attributeStrings := strings.Split(s2, ";")
// 				if len(attributeStrings) == 1 && attributeStrings[0] == "0" {
// 					currentColor = ""
// 				}
// 				for _, attributeString := range attributeStrings {
// 					attributeNumber, err := strconv.Atoi(attributeString)
// 					if err != nil {
// 						continue
// 					}
// 					currentColor = append(currentColor, byte(attributeNumber))
// 				}
// 				// Strip away leading 0 color attribute, if there are more than 1
// 				if len(currentColor) > 1 && currentColor[0] == 0 {
// 					currentColor = currentColor[1:]
// 				}
// 				// currentColor now contains the last found color attributes,
// 				// but as an AttributeColor.
// 				colorcode.Reset()
// 				escaped = false
// 			}
// 		} else {
// 			cc = append(cc, CharAttribute{r, currentColor})
// 		}
// 	}
// 	// if escaped is true here, there is something wrong
// 	return cc
//}
