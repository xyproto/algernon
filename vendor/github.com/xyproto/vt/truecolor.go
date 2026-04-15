package vt

import (
	"fmt"
	"math"
	"strings"
)

// IsTrueColor reports whether ac was created with TrueColor or TrueBackground
// (i.e. holds a 24-bit RGB value)
func IsTrueColor(ac AttributeColor) bool {
	val := uint32(ac)
	return val&extendedFlag != 0 && val&trueColorFlag != 0
}

// Is256Color reports whether ac was created with Color256 or Background256
// (i.e. holds an xterm-256color palette index)
func Is256Color(ac AttributeColor) bool {
	val := uint32(ac)
	return val&extendedFlag != 0 && val&trueColorFlag == 0
}

// ToRGB extracts the RGB components of any AttributeColor:
//   - TrueColor / TrueBackground → exact 24-bit values, ok=true
//   - 256-color / background256  → palette-derived values via Color256ToRGB, ok=true
//   - Standard ANSI 16 foreground (30–37, 90–97) → approximate values from ansi16Palette, ok=true
//   - Anything else (attributes, Default, …)      → 0, 0, 0, false
func ToRGB(ac AttributeColor) (r, g, b uint8, ok bool) {
	val := uint32(ac)
	if val&extendedFlag != 0 {
		if val&trueColorFlag != 0 {
			return uint8((val >> 16) & 0xFF), uint8((val >> 8) & 0xFF), uint8(val & 0xFF), true
		}
		// 256-color: ignore the bg flag, index is in bits 0–7
		r, g, b = Color256ToRGB(uint8(val & 0xFF))
		return r, g, b, true
	}
	// Standard ANSI 16: foreground codes 30–37 map to palette indices 0–7,
	// bright foreground 90–97 map to indices 8–15
	if val >= 30 && val <= 37 {
		e := ansi16Palette[val-30]
		return e.r, e.g, e.b, true
	}
	if val >= 90 && val <= 97 {
		e := ansi16Palette[val-90+8]
		return e.r, e.g, e.b, true
	}
	return 0, 0, 0, false
}

// ToHex returns the hex color string ("#rrggbb") for any AttributeColor whose
// RGB can be determined. Returns "#000000" for non-color attributes.
func ToHex(ac AttributeColor) string {
	r, g, b, _ := ToRGB(ac)
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// clampF converts a [0.0, 1.0] float64 to a uint8, clamping at the boundaries
func clampF(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return uint8(math.Round(v * 255))
}

// isBgColor reports whether ac is a background color (extended bg flag, or
// standard background codes 40–49 / 100–107)
func isBgColor(ac AttributeColor) bool {
	val := uint32(ac)
	if val&extendedFlag != 0 {
		return val&bgFlag != 0
	}
	return (val >= 40 && val <= 49) || (val >= 100 && val <= 107)
}

// asTrueColor builds a TrueColor or TrueBackground AttributeColor with the
// given RGB, preserving the fg/bg sense of src
func asTrueColor(r, g, b uint8, src AttributeColor) AttributeColor {
	if isBgColor(src) {
		return TrueBackground(r, g, b)
	}
	return TrueColor(r, g, b)
}

// Lighten returns a lighter version of ac by blending its RGB toward white.
// amount is in [0.0, 1.0]: 0.0 = unchanged, 1.0 = white.
// For non-color attributes, ac is returned unchanged.
func Lighten(ac AttributeColor, amount float64) AttributeColor {
	r, g, b, ok := ToRGB(ac)
	if !ok {
		return ac
	}
	fr, fg, fb := float64(r)/255.0, float64(g)/255.0, float64(b)/255.0
	fr = fr + (1.0-fr)*amount
	fg = fg + (1.0-fg)*amount
	fb = fb + (1.0-fb)*amount
	return asTrueColor(clampF(fr), clampF(fg), clampF(fb), ac)
}

// Darken returns a darker version of ac by blending its RGB toward black.
// amount is in [0.0, 1.0]: 0.0 = unchanged, 1.0 = black.
// For non-color attributes, ac is returned unchanged.
func Darken(ac AttributeColor, amount float64) AttributeColor {
	r, g, b, ok := ToRGB(ac)
	if !ok {
		return ac
	}
	fr, fg, fb := float64(r)/255.0, float64(g)/255.0, float64(b)/255.0
	fr = fr * (1.0 - amount)
	fg = fg * (1.0 - amount)
	fb = fb * (1.0 - amount)
	return asTrueColor(clampF(fr), clampF(fg), clampF(fb), ac)
}

// Blend linearly interpolates between colors a and b.
// t=0.0 returns a; t=1.0 returns b.
// The result is always a TrueColor (or TrueBackground if a is a background color).
// If neither a nor b has a representable RGB, a is returned unchanged.
func Blend(a, b AttributeColor, t float64) AttributeColor {
	ra, ga, ba, okA := ToRGB(a)
	rb, gb, bb, okB := ToRGB(b)
	if !okA && !okB {
		return a
	}
	if !okA {
		return b
	}
	if !okB {
		return a
	}
	lerp := func(u, v uint8) uint8 {
		return clampF(float64(u)/255.0*(1.0-t) + float64(v)/255.0*t)
	}
	return asTrueColor(lerp(ra, rb), lerp(ga, gb), lerp(ba, bb), a)
}

// linearize converts a sRGB channel value [0, 255] to linear light for
// WCAG 2.1 relative luminance calculation.
func linearize(c uint8) float64 {
	f := float64(c) / 255.0
	if f <= 0.04045 {
		return f / 12.92
	}
	return math.Pow((f+0.055)/1.055, 2.4)
}

// Luminance returns the WCAG 2.1 relative luminance of ac in [0.0, 1.0].
// Returns 0.0 for non-color attributes.
func Luminance(ac AttributeColor) float64 {
	r, g, b, ok := ToRGB(ac)
	if !ok {
		return 0
	}
	return 0.2126*linearize(r) + 0.7152*linearize(g) + 0.0722*linearize(b)
}

// ContrastRatio returns the WCAG 2.1 contrast ratio between fg and bg.
// The result is in [1.0, 21.0]: 1.0 = no contrast, 21.0 = black on white.
func ContrastRatio(fg, bg AttributeColor) float64 {
	l1 := Luminance(fg)
	l2 := Luminance(bg)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

// HasSufficientContrast reports whether fg and bg meet the WCAG 2.1 AA
// minimum contrast ratio of 4.5 for normal text.
func HasSufficientContrast(fg, bg AttributeColor) bool {
	return ContrastRatio(fg, bg) >= 4.5
}

// BestColorFromHex parses a "#rrggbb" (or "#rgb" / "rrggbb" / "rgb") hex string
// and returns the most faithful foreground AttributeColor the terminal supports,
// using the same terminal-capability detection as BestColor.
// Returns Default on parse error.
func BestColorFromHex(s string) AttributeColor {
	s = strings.TrimPrefix(s, "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	r, g, b, err := parseHexColor(s)
	if err != nil {
		return Default
	}
	return BestColor(r, g, b)
}

// BestBackgroundFromHex is the background-color variant of BestColorFromHex.
// Returns DefaultBackground on parse error.
func BestBackgroundFromHex(s string) AttributeColor {
	s = strings.TrimPrefix(s, "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	r, g, b, err := parseHexColor(s)
	if err != nil {
		return DefaultBackground
	}
	return BestBackground(r, g, b)
}
