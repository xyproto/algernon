package vt

import "github.com/xyproto/env/v2"

// hasTrueColorEnv is true when the terminal advertises 24-bit color support
// via the COLORTERM environment variable.
var hasTrueColorEnv = func() bool {
	ct := env.Str("COLORTERM")
	return ct == "truecolor" || ct == "24bit"
}()

// HasTrueColor returns true when the terminal supports 24-bit true color
// (i.e. $COLORTERM is "truecolor" or "24bit").
func HasTrueColor() bool {
	return hasTrueColorEnv
}

// Color256ToRGB returns the approximate RGB values for an xterm-256color palette index.
// Indices 0–15 return the standard ANSI colors, 16–231 the 6×6×6 color cube,
// and 232–255 the 24-step grayscale ramp.
func Color256ToRGB(n uint8) (r, g, b uint8) {
	switch {
	case n < 16:
		e := ansi16Palette[n]
		return e.r, e.g, e.b
	case n < 232:
		// 6×6×6 color cube: index = 16 + 36*r + 6*g + b, each component 0–5
		idx := n - 16
		cubeLevel := func(v uint8) uint8 {
			if v == 0 {
				return 0
			}
			return 55 + 40*v
		}
		return cubeLevel(idx / 36), cubeLevel((idx % 36) / 6), cubeLevel(idx % 6)
	default:
		// Grayscale ramp 232–255: value = 8 + 10*(n-232)
		v := 8 + 10*(n-232)
		return v, v, v
	}
}

// NearestColor256 returns the AttributeColor for the xterm-256color palette entry
// whose RGB value is closest to (r, g, b) by squared Euclidean distance.
func NearestColor256(r, g, b uint8) AttributeColor {
	best := uint8(0)
	bestDist := ^uint32(0)
	for i := range 256 {
		pr, pg, pb := Color256ToRGB(uint8(i))
		dr := int32(r) - int32(pr)
		dg := int32(g) - int32(pg)
		db := int32(b) - int32(pb)
		dist := uint32(dr*dr + dg*dg + db*db)
		if dist < bestDist {
			bestDist = dist
			best = uint8(i)
		}
		if dist == 0 {
			break
		}
	}
	return Color256(best)
}

// Grayscale256 returns a 256-color foreground AttributeColor from the 24-step
// grayscale ramp (palette indices 232–255). level 0 is near-black (rgb 8,8,8)
// and level 23 is near-white (rgb 238,238,238). Values above 23 are clamped.
func Grayscale256(level uint8) AttributeColor {
	if level > 23 {
		level = 23
	}
	return Color256(232 + level)
}

// ColorCube returns a 256-color foreground AttributeColor from the 6×6×6 RGB
// color cube (palette indices 16–231). Each of r, g, b must be 0–5; values
// above 5 are clamped.
func ColorCube(r, g, b uint8) AttributeColor {
	if r > 5 {
		r = 5
	}
	if g > 5 {
		g = 5
	}
	if b > 5 {
		b = 5
	}
	return Color256(16 + 36*r + 6*g + b)
}

// BestColor returns the most faithful foreground AttributeColor for (r, g, b)
// that the current terminal can display:
//   - Default (no color) if NO_COLOR is set
//   - 24-bit true color if $COLORTERM is "truecolor" or "24bit"
//   - nearest xterm-256color entry if $TERM contains "256color" or is "xterm-kitty"
//   - nearest ANSI-16 color otherwise
func BestColor(r, g, b uint8) AttributeColor {
	if EnvNoColor {
		return Default
	}
	if hasTrueColorEnv {
		return TrueColor(r, g, b)
	}
	if Has256Colors() {
		return NearestColor256(r, g, b)
	}
	return nearestANSI16(r, g, b)
}

// BestBackground is the background-color variant of BestColor.
func BestBackground(r, g, b uint8) AttributeColor {
	if EnvNoColor {
		return DefaultBackground
	}
	return BestColor(r, g, b).Background()
}
