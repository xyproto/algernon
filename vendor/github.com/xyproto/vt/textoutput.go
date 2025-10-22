package vt

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/xyproto/env/v2"
)

// CharAttribute is a rune and a color attribute
type CharAttribute struct {
	A AttributeColor
	R rune
}

// TextOutput keeps state about verbosity and if colors are enabled
type TextOutput struct {
	lightReplacer *strings.Replacer
	darkReplacer  *strings.Replacer
	color         bool
	enabled       bool
}

// EnvNoColor respects the NO_COLOR environment variable
var EnvNoColor = env.Bool("NO_COLOR")

// NewTextOutput can initialize a new TextOutput struct,
// which can have colors turned on or off and where the
// output can be enabled (verbose) or disabled (silent).
// If NO_COLOR is set, colors are disabled, regardless.
func NewTextOutput(color, enabled bool) *TextOutput {
	if EnvNoColor {
		color = false
	}
	o := &TextOutput{nil, nil, color, enabled}
	o.initializeTagReplacers()
	return o
}

// New can initialize a new TextOutput struct,
// which can have colors turned on or off and where the
// output can be enabled (verbose) or disabled (silent).
// If NO_COLOR is set, colors are disabled.
func New() *TextOutput {
	o := &TextOutput{nil, nil, !EnvNoColor, true}
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

func Println(msg ...interface{})               { New().Println(msg...) }
func Print(msg ...interface{})                 { New().Print(msg...) }
func Printf(format string, msg ...interface{}) { New().Printf(format, msg...) }

// Println writes a message to stdout if output is enabled
func (o *TextOutput) Println(msg ...interface{}) {
	if o.enabled {
		fmt.Println(o.InterfaceTags(msg...))
	}
}

// Print writes a message to stdout if output is enabled
func (o *TextOutput) Print(msg ...interface{}) {
	if o.enabled {
		fmt.Print(o.InterfaceTags(msg...))
	}
}

// Printf writes a formatted message to stdout if output is enabled
func (o *TextOutput) Printf(format string, args ...interface{}) {
	if o.enabled {
		fmt.Print(o.Tags(fmt.Sprintf(format, args...)))
	}
}

// Disable text output
func (o *TextOutput) Disable() {
	o.enabled = false
}

// Enable text output
func (o *TextOutput) Enable() {
	o.enabled = true
}

// Err writes an error message in red to stderr if output is enabled
func (o *TextOutput) Err(msg string) {
	if o.enabled {
		if o.color {
			Red.Error(msg)
		} else {
			Default.Error(msg)
		}
	}
}

// ErrExit writes an error message to stderr and quit with exit code 1
func (o *TextOutput) ErrExit(msg string) {
	o.Err(msg)
	os.Exit(1)
}

func (o *TextOutput) LightBlue(s string) string {
	if o.color {
		return LightBlue.Get(s)
	}
	return s
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

func (o *TextOutput) initializeTagReplacers() {
	// Initialize tag replacement tables, with as few memory allocations as possible (no append)
	off := NoColor
	rs := make([]string, len(LightColorMap)*4+2)
	i := 0
	if o.color {
		for key, value := range LightColorMap {
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

// ExtractToSlice iterates over an ANSI encoded string, parsing out color codes and places it in
// a slice of CharAttribute. Each CharAttribute in the slice represents a character in the
// input string and its corresponding color attributes. This function handles escaping sequences
// and converts ANSI color codes to AttributeColor structs.
// The returned uint is the number of stored elements.
func (o *TextOutput) ExtractToSlice(s string, pcc *[]CharAttribute) uint {
	var (
		escaped      bool
		colorcode    strings.Builder
		currentColor AttributeColor
	)
	counter := uint(0)
	for _, r := range s {
		switch {
		case escaped && r == 'm':
			colorAttributes := strings.Split(strings.TrimPrefix(colorcode.String(), "["), ";")
			if len(colorAttributes) != 1 || colorAttributes[0] != "0" {
				var primaryAttr, secondaryAttr AttributeColor
				for i, attribute := range colorAttributes {
					if attributeNumber, err := strconv.Atoi(attribute); err == nil {
						if i == 0 {
							primaryAttr = AttributeColor(attributeNumber)
						} else {
							secondaryAttr = AttributeColor(attributeNumber)
							break // Only handle two attributes for now
						}
					}
				}
				if secondaryAttr != 0 {
					currentColor = primaryAttr.Combine(secondaryAttr)
				} else {
					currentColor = primaryAttr
				}
			} else {
				currentColor = AttributeColor(0)
			}
			colorcode.Reset()
			escaped = false
		case r == '\033':
			escaped = true
		case escaped && r != 'm':
			colorcode.WriteRune(r)
		default:
			if counter >= uint(len(*pcc)) {
				// Extend the slice
				newSlice := make([]CharAttribute, len(*pcc)*2+1)
				copy(newSlice, *pcc)
				*pcc = newSlice
			}
			(*pcc)[counter] = CharAttribute{currentColor, r}
			counter++
		}
	}
	return counter
}
