package vt

import "math"

// LowContrast check if there is a low color contrast between the given foreground and background color,
// given lightBackground, indicating if this is a "light" or "dark" terminal color scheme.
func LowContrast(fg, bg AttributeColor, lightBackground bool) bool {
	// defaultContrastThreshold is a relaxed minimum contrast ratio
	const defaultContrastThreshold = 3.5
	return lowContrastLevelWithThreshold(fg, bg, lightBackground, defaultContrastThreshold)
}

// LowContrastLevelWithThreshold classifies contrast using a custom minimum ratio
func lowContrastLevelWithThreshold(fg, bg AttributeColor, lightBackground bool, minRatio float64) bool {
	if minRatio <= 0 {
		return false
	}
	fr, fgG, fb := colorForContrast(fg, true, lightBackground)
	br, bgG, bb := colorForContrast(bg, false, lightBackground)
	if isGray(fr, fgG, fb) && isGray(br, bgG, bb) {
		return true
	}
	if contrastRatioFromRGB(fr, fgG, fb, br, bgG, bb) < minRatio {
		return true
	}
	return false
}

// contrastRatio returns the WCAG contrast ratio between the two colors
func contrastRatio(fg, bg AttributeColor, lightBackground bool) float64 {
	fr, fgG, fb := colorForContrast(fg, true, lightBackground)
	br, bgG, bb := colorForContrast(bg, false, lightBackground)
	return contrastRatioFromRGB(fr, fgG, fb, br, bgG, bb)
}

func contrastRatioFromRGB(fr, fgG, fb, br, bgG, bb float64) float64 {
	l1 := luminance(fr, fgG, fb)
	l2 := luminance(br, bgG, bb)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

func isGray(r, g, b float64) bool {
	return r == g && g == b
}

func colorForContrast(ac AttributeColor, isForeground bool, lightBackground bool) (r, g, b float64) {
	code, ok := extractColorCode(ac, isForeground)
	if !ok || code == Default || code == DefaultBackground {
		return defaultColor(isForeground, lightBackground)
	}
	c := ansiCodeToColor(code, isForeground)
	return float64(c.R), float64(c.G), float64(c.B)
}

func defaultColor(isForeground bool, lightBackground bool) (r, g, b float64) {
	if isForeground {
		if lightBackground {
			return 0, 0, 0
		}
		return 229, 229, 229
	}
	if lightBackground {
		return 255, 255, 255
	}
	return 0, 0, 0
}

func extractColorCode(ac AttributeColor, isForeground bool) (AttributeColor, bool) {
	if code, ok := normalizeColorCode(ac, isForeground); ok {
		return code, true
	}
	if uint32(ac) <= 0xFFFF {
		return 0, false
	}
	low := AttributeColor(uint32(ac) & 0xFFFF)
	if code, ok := normalizeColorCode(low, isForeground); ok {
		return code, true
	}
	high := AttributeColor(uint32(ac) >> 16)
	if code, ok := normalizeColorCode(high, isForeground); ok {
		return code, true
	}
	return 0, false
}

func normalizeColorCode(ac AttributeColor, isForeground bool) (AttributeColor, bool) {
	switch {
	case isForeground && (isForegroundCode(ac) || isDefaultCode(ac)):
		return ac, true
	case !isForeground && (isBackgroundCode(ac) || isDefaultCode(ac)):
		return ac, true
	case isForeground && isBackgroundCode(ac):
		return AttributeColor(uint32(ac) - 10), true
	case !isForeground && isForegroundCode(ac):
		return ac.Background(), true
	default:
		return 0, false
	}
}

func isForegroundCode(ac AttributeColor) bool {
	return (ac >= 30 && ac <= 37) || (ac >= 90 && ac <= 97)
}

func isBackgroundCode(ac AttributeColor) bool {
	return (ac >= 40 && ac <= 47) || (ac >= 100 && ac <= 107)
}

func isDefaultCode(ac AttributeColor) bool {
	return ac == Default || ac == DefaultBackground
}

func luminance(r, g, b float64) float64 {
	return 0.2126*srgbToLinear(r) + 0.7152*srgbToLinear(g) + 0.0722*srgbToLinear(b)
}

func srgbToLinear(c float64) float64 {
	c /= 255.0
	if c <= 0.03928 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}
