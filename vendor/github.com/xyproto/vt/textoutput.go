package vt

import (
	"fmt"
	"io"
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
var EnvNoColor = env.Bool("NO_COLOR") || env.Str("TERM") == "vt100"

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

// DisableColors will enable color output
func (o *TextOutput) EnableColors() {
	o.color = true
}

// DisableColors will disable color output
func (o *TextOutput) DisableColors() {
	o.color = false
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

func Println(msg ...any)                             { New().Println(msg...) }
func Print(msg ...any)                               { New().Print(msg...) }
func Printf(format string, msg ...any)               { New().Printf(format, msg...) }
func Eprintln(msg ...any)                            { New().Eprintln(msg...) }
func Eprint(msg ...any)                              { New().Eprint(msg...) }
func Eprintf(format string, msg ...any)              { New().Eprintf(format, msg...) }
func Fprintln(w io.Writer, msg ...any)               { New().Fprintln(w, msg...) }
func Fprint(w io.Writer, msg ...any)                 { New().Fprint(w, msg...) }
func Fprintf(w io.Writer, format string, msg ...any) { New().Fprintf(w, format, msg...) }

// Println writes a message to stdout if output is enabled
func (o *TextOutput) Println(msg ...any) {
	if o.enabled {
		fmt.Println(o.InterfaceTags(msg...))
	}
}

// Eprintln writes a message to stderr if output is enabled
func (o *TextOutput) Eprintln(msg ...any) {
	if o.enabled {
		fmt.Fprintln(os.Stderr, o.InterfaceTags(msg...))
	}
}

// Fprintln writes a message to the given io.Writer, if output is enabled
func (o *TextOutput) Fprintln(w io.Writer, msg ...any) {
	if o.enabled {
		fmt.Fprintln(w, o.InterfaceTags(msg...))
	}
}

// Print writes a message to stdout if output is enabled
func (o *TextOutput) Print(msg ...any) {
	if o.enabled {
		fmt.Print(o.InterfaceTags(msg...))
	}
}

// Eprint writes a message to stderr if output is enabled
func (o *TextOutput) Eprint(msg ...any) {
	if o.enabled {
		fmt.Fprint(os.Stderr, o.InterfaceTags(msg...))
	}
}

// Fprint writes a message to the given io.Writer, if output is enabled
func (o *TextOutput) Fprint(w io.Writer, msg ...any) {
	if o.enabled {
		fmt.Fprint(w, o.InterfaceTags(msg...))
	}
}

// Printf writes a formatted message to stdout if output is enabled
func (o *TextOutput) Printf(format string, args ...any) {
	if o.enabled {
		fmt.Print(o.Tags(fmt.Sprintf(format, args...)))
	}
}

// Eprintf writes a formatted message to stderr if output is enabled
func (o *TextOutput) Eprintf(format string, args ...any) {
	if o.enabled {
		fmt.Fprint(os.Stderr, o.Tags(fmt.Sprintf(format, args...)))
	}
}

// Fprintf writes a formatted message to the given io.Writer, if output is enabled
func (o *TextOutput) Fprintf(w io.Writer, format string, args ...any) {
	if o.enabled {
		fmt.Fprint(w, o.Tags(fmt.Sprintf(format, args...)))
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

// Enabled checks if the text output is enabled
func (o *TextOutput) Enabled() bool {
	return o.enabled
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
func (o *TextOutput) InterfaceTags(colors ...any) string {
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

// buildTagReplacer builds a strings.Replacer that substitutes <color>/</color>
// HTML-like tags in text. Each key in colorMap generates four pairs covering
// both <key>/</key> and <Key>/</Key>. When enabled is false every tag is
// replaced with an empty string (strip-only mode).
func buildTagReplacer(colorMap map[string]AttributeColor, enabled bool) *strings.Replacer {
	off := NoColor
	rs := make([]string, len(colorMap)*8+2)
	i := 0
	for key, value := range colorMap {
		titled := strings.ToUpper(key[:1]) + key[1:]
		var esc, reset string
		if enabled {
			esc = value.String()
			reset = off
		}
		rs[i] = "<" + key + ">"
		rs[i+1] = esc
		rs[i+2] = "</" + key + ">"
		rs[i+3] = reset
		rs[i+4] = "<" + titled + ">"
		rs[i+5] = esc
		rs[i+6] = "</" + titled + ">"
		rs[i+7] = reset
		i += 8
	}
	if enabled {
		rs[i] = "<off>"
		rs[i+1] = off
	} else {
		rs[i] = "<off>"
		rs[i+1] = ""
	}
	return strings.NewReplacer(rs...)
}

// Tag replacers are built once at package init and shared across all TextOutput
// instances; building them is O(|colorMap|) and involves string allocations, so
// doing it once avoids repeated work on every New() call.
var (
	cachedLightOnReplacer  *strings.Replacer
	cachedLightOffReplacer *strings.Replacer
	cachedDarkOnReplacer   *strings.Replacer
	cachedDarkOffReplacer  *strings.Replacer
)

func init() {
	cachedLightOnReplacer = buildTagReplacer(LightColorMap, true)
	cachedLightOffReplacer = buildTagReplacer(LightColorMap, false)
	cachedDarkOnReplacer = buildTagReplacer(DarkColorMap, true)
	cachedDarkOffReplacer = buildTagReplacer(DarkColorMap, false)
}

// initializeTagReplacers assigns pre-built singleton replacers to this
// TextOutput based on whether colors are enabled.
func (o *TextOutput) initializeTagReplacers() {
	if o.color {
		o.lightReplacer = cachedLightOnReplacer
		o.darkReplacer = cachedDarkOnReplacer
	} else {
		o.lightReplacer = cachedLightOffReplacer
		o.darkReplacer = cachedDarkOffReplacer
	}
}

// ExtractToSlice iterates over an ANSI encoded string, parsing out color codes and places it in
// a slice of CharAttribute. Each CharAttribute in the slice represents a character in the
// input string and its corresponding color attributes. This function handles escaping sequences
// and converts ANSI color codes to AttributeColor structs, including 256-color sequences
// of the form ESC[38;5;Nm (foreground) and ESC[48;5;Nm (background).
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
				// Parse the numeric fields up front
				nums := make([]int, 0, len(colorAttributes))
				for _, attribute := range colorAttributes {
					if n, err := strconv.Atoi(attribute); err == nil {
						nums = append(nums, n)
					}
				}
				switch {
				case len(nums) >= 3 && nums[0] == 38 && nums[1] == 5:
					// ESC[38;5;Nm — 256-color foreground
					currentColor = Color256(uint8(nums[2]))
				case len(nums) >= 3 && nums[0] == 48 && nums[1] == 5:
					// ESC[48;5;Nm — 256-color background
					currentColor = Background256(uint8(nums[2]))
				case len(nums) == 2:
					currentColor = AttributeColor(nums[0]).Combine(AttributeColor(nums[1]))
				case len(nums) == 1:
					currentColor = AttributeColor(nums[0])
				}
			} else {
				currentColor = AttributeColor(0)
			}
			colorcode.Reset()
			escaped = false
		case r == '\033':
			escaped = true
		case escaped && r != 'm':
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				// A letter other than 'm' terminates a non-SGR escape sequence; discard it
				colorcode.Reset()
				escaped = false
			} else {
				colorcode.WriteRune(r)
			}
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

// WriteTagged writes a tagged string ("<green>hello</green>") to the Canvas
func (c *Canvas) WriteTagged(x, y uint, bgColor AttributeColor, tagged string) {
	pcc := make([]CharAttribute, len([]rune(tagged)))
	n := New().ExtractToSlice(tagged, &pcc)
	for i := range n {
		c.WriteRune(i+x, y, pcc[i].A, bgColor, pcc[i].R)
	}
}
