// Package textoutput offers a simple way to use vt100 and output colored text
package textoutput

import (
	"fmt"
	"github.com/xyproto/vt100"
	"os"
	"strconv"
	"strings"
)

// CharAttribute is a rune and a color attribute
type CharAttribute struct {
	R rune
	A vt100.AttributeColor
}

// TextOutput keeps state about verbosity and if colors are enabled
type TextOutput struct {
	color   bool
	enabled bool
	// Tag replacement structs, for performance
	lightReplacer *strings.Replacer
	darkReplacer  *strings.Replacer
}

func NewTextOutput(color, enabled bool) *TextOutput {
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
		fmt.Println(msg...)
	}
}

// Write an error message in red to stderr if output is enabled
func (o *TextOutput) Err(msg string) {
	if o.enabled {
		vt100.Red.Error(msg)
	}
}

// Write an error message to stderr and quit with exit code 1
func (o *TextOutput) ErrExit(msg string) {
	o.Err(msg)
	os.Exit(1)
}

// Checks if textual output is enabled
func (o *TextOutput) IsEnabled() bool {
	return o.enabled
}

func (o *TextOutput) DarkRed(s string) string {
	if o.color {
		return vt100.Red.Get(s)
	}
	return s
}

func (o *TextOutput) DarkGreen(s string) string {
	if o.color {
		return vt100.Green.Get(s)
	}
	return s
}

func (o *TextOutput) DarkYellow(s string) string {
	if o.color {
		return vt100.Yellow.Get(s)
	}
	return s
}

func (o *TextOutput) DarkBlue(s string) string {
	if o.color {
		return vt100.Blue.Get(s)
	}
	return s
}

func (o *TextOutput) DarkPurple(s string) string {
	if o.color {
		return vt100.Magenta.Get(s)
	}
	return s
}

func (o *TextOutput) DarkCyan(s string) string {
	if o.color {
		return vt100.Cyan.Get(s)
	}
	return s
}

func (o *TextOutput) DarkGray(s string) string {
	if o.color {
		return vt100.DarkGray.Get(s)
	}
	return s
}

func (o *TextOutput) LightRed(s string) string {
	if o.color {
		return vt100.LightRed.Get(s)
	}
	return s
}

func (o *TextOutput) LightGreen(s string) string {
	if o.color {
		return vt100.LightGreen.Get(s)
	}
	return s
}

func (o *TextOutput) LightYellow(s string) string {
	if o.color {
		return vt100.LightYellow.Get(s)
	}
	return s
}

func (o *TextOutput) LightBlue(s string) string {
	if o.color {
		return vt100.LightBlue.Get(s)
	}
	return s
}

func (o *TextOutput) LightPurple(s string) string {
	if o.color {
		return vt100.LightMagenta.Get(s)
	}
	return s
}

func (o *TextOutput) LightCyan(s string) string {
	if o.color {
		return vt100.LightCyan.Get(s)
	}
	return s
}

func (o *TextOutput) White(s string) string {
	if o.color {
		return vt100.White.Get(s)
	}
	return s
}

// Given a line with words and several color strings, color the words
// in the order of the colors. The last color will color the rest of the
// words.
func (o *TextOutput) Words(line string, colors ...string) string {
	if o.color {
		return vt100.Words(line, colors...)
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
	off := vt100.NoColor()
	rs := make([]string, len(vt100.LightColorMap)*4+2)
	i := 0
	if o.color {
		for key, value := range vt100.LightColorMap {
			rs[i] = "<" + key + ">"
			i++
			rs[i] = value.String()
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
		for key := range vt100.LightColorMap {
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
		for key, value := range vt100.DarkColorMap {
			rs[i] = "<" + key + ">"
			i++
			rs[i] = value.String()
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
		for key := range vt100.DarkColorMap {
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

// Pair takes a string with ANSI codes and returns
// a slice with two elements.
func (o *TextOutput) Extract(s string) []CharAttribute {
	escaped := false
	var colorcode strings.Builder
	var word strings.Builder
	cc := make([]CharAttribute, 0)
	var currentColor vt100.AttributeColor
	for _, r := range s {
		if r == '\033' {
			escaped = true
			w := word.String()
			if w != "" {
				//fmt.Println("cc", cc)
				word.Reset()
			}
			continue
		}
		if escaped {
			if r != 'm' {
				colorcode.WriteRune(r)
			} else if r == 'm' {
				s := colorcode.String()
				if strings.HasPrefix(s, "[") {
					s = s[1:]
				}
				attributeStrings := strings.Split(s, ";")
				if len(attributeStrings) == 1 && attributeStrings[0] == "0" {
					currentColor = []byte{}
				}
				for _, attributeString := range attributeStrings {
					attributeNumber, err := strconv.Atoi(attributeString)
					if err != nil {
						continue
					}
					currentColor = append(currentColor, byte(attributeNumber))
				}
				// Strip away leading 0 color attribute, if there are more than 1
				if len(currentColor) > 1 && currentColor[0] == 0 {
					currentColor = currentColor[1:]
				}
				// currentColor now contains the last found color attributes,
				// but as a vt100.AttributeColor.
				colorcode.Reset()
				escaped = false
			}
		} else {
			cc = append(cc, CharAttribute{r, currentColor})
		}
	}
	// if escaped is true here, there is something wrong
	return cc
}
