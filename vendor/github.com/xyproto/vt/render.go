package vt

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/xyproto/burnfont"
)

// ansiRenderPalette maps ANSI color codes to their approximate NRGBA values for
// canvas-to-image rendering. Foreground codes (30–37, 90–97) and their
// background counterparts (40–47, 100–107) are stored at their respective
// indices. An alpha of 0 signals "no entry" (unknown code → black fallback).
var ansiRenderPalette [108]color.NRGBA

func init() {
	// Standard foreground colors (30–37) and their background equivalents (40–47)
	pairs := [8]color.NRGBA{
		{0, 0, 0, 255},       // Black
		{205, 0, 0, 255},     // Red
		{0, 205, 0, 255},     // Green
		{205, 205, 0, 255},   // Yellow
		{0, 0, 238, 255},     // Blue
		{205, 0, 205, 255},   // Magenta
		{0, 205, 205, 255},   // Cyan
		{229, 229, 229, 255}, // LightGray / White
	}
	for i, c := range pairs {
		ansiRenderPalette[30+i] = c
		ansiRenderPalette[40+i] = c
	}
	// Bright foreground colors (90–97) and their background equivalents (100–107)
	bright := [8]color.NRGBA{
		{127, 127, 127, 255}, // DarkGray
		{255, 0, 0, 255},     // LightRed
		{0, 255, 0, 255},     // LightGreen
		{255, 255, 0, 255},   // LightYellow
		{0, 0, 255, 255},     // LightBlue
		{255, 0, 255, 255},   // LightMagenta
		{0, 255, 255, 255},   // LightCyan
		{255, 255, 255, 255}, // White
	}
	for i, c := range bright {
		ansiRenderPalette[90+i] = c
		ansiRenderPalette[100+i] = c
	}
}

func (c *Canvas) ToImage() (image.Image, error) {
	const charWidth, charHeight = 8, 8
	c.mut.RLock()
	defer c.mut.RUnlock()
	width, height := int(c.w)*charWidth, int(c.h)*charHeight
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	filled := false
	for y := uint(0); y < c.h; y++ {
		for x := uint(0); x < c.w; x++ {
			cr := c.chars[y*c.w+x]
			if cr.r == rune(0) {
				continue
			}
			fgColor := ansiCodeToColor(cr.fg)
			bgColor := ansiCodeToColor(cr.bg)
			if !filled {
				draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)
				filled = true
			}
			charRect := image.Rect(int(x)*charWidth, int(y)*charHeight, (int(x)+1)*charWidth, (int(y)+1)*charHeight)
			draw.Draw(img, charRect, &image.Uniform{bgColor}, image.Point{}, draw.Src)
			if cr.r != rune(0) {
				burnfont.DrawString(img, int(x)*charWidth, int(y)*charHeight, string(cr.r), fgColor)
			}
		}
	}
	return img, nil
}

// ansiCodeToColor converts an AttributeColor to its approximate NRGBA value for
// canvas-to-image rendering. True-color and 256-color values are decoded from
// their embedded bits; standard ANSI codes are served from ansiRenderPalette.
func ansiCodeToColor(ac AttributeColor) color.NRGBA {
	code := uint32(ac)
	if code&extendedFlag != 0 {
		if code&trueColorFlag != 0 {
			// True-color (24-bit RGB): bits 0–23 hold R, G, B
			return color.NRGBA{
				R: uint8((code >> 16) & 0xFF),
				G: uint8((code >> 8) & 0xFF),
				B: uint8(code & 0xFF),
				A: 255,
			}
		}
		// 256-color: palette index in bits 0–7
		r, g, b := Color256ToRGB(uint8(code & 0xFF))
		return color.NRGBA{R: r, G: g, B: b, A: 255}
	}
	if code < uint32(len(ansiRenderPalette)) {
		if c := ansiRenderPalette[code]; c.A != 0 {
			return c
		}
	}
	return color.NRGBA{0, 0, 0, 255} // fallback: black
}
