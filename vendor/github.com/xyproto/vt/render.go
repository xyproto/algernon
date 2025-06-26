package vt

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/xyproto/burnfont"
)

func (c *Canvas) ToImage() (image.Image, error) {
	const charWidth, charHeight = 8, 8
	width, height := int(c.w)*charWidth, int(c.h)*charHeight
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	filled := false
	for y := uint(0); y < c.h; y++ {
		for x := uint(0); x < c.w; x++ {
			cr := c.chars[y*c.w+x]
			if cr.r == rune(0) {
				continue
			}
			fgColor := ansiCodeToColor(cr.fg, true)
			bgColor := ansiCodeToColor(cr.bg, false)
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

func ansiCodeToColor(ac AttributeColor, isForeground bool) color.NRGBA {
	code := uint32(ac)
	if code == 0 {
		return color.NRGBA{0, 0, 0, 255} // Default black color
	}
	if isForeground {
		switch code {
		case 30: // Black
			return color.NRGBA{0, 0, 0, 255}
		case 31: // Red
			return color.NRGBA{205, 0, 0, 255}
		case 32: // Green
			return color.NRGBA{0, 205, 0, 255}
		case 33: // Yellow
			return color.NRGBA{205, 205, 0, 255}
		case 34: // Blue
			return color.NRGBA{0, 0, 238, 255}
		case 35: // Magenta
			return color.NRGBA{205, 0, 205, 255}
		case 36: // Cyan
			return color.NRGBA{0, 205, 205, 255}
		case 37: // White
			return color.NRGBA{229, 229, 229, 255}
		case 90: // Bright Black (Gray)
			return color.NRGBA{127, 127, 127, 255}
		case 91: // Bright Red
			return color.NRGBA{255, 0, 0, 255}
		case 92: // Bright Green
			return color.NRGBA{0, 255, 0, 255}
		case 93: // Bright Yellow
			return color.NRGBA{255, 255, 0, 255}
		case 94: // Bright Blue
			return color.NRGBA{0, 0, 255, 255}
		case 95: // Bright Magenta
			return color.NRGBA{255, 0, 255, 255}
		case 96: // Bright Cyan
			return color.NRGBA{0, 255, 255, 255}
		case 97: // Bright White
			return color.NRGBA{255, 255, 255, 255}
		default:
			return color.NRGBA{0, 0, 0, 255}
		}
	} else {
		// Background color mappings
		// TODO: Use a different palette than for the foreground colors
		switch code {
		case 40: // Black
			return color.NRGBA{0, 0, 0, 255}
		case 41: // Red
			return color.NRGBA{205, 0, 0, 255}
		case 42: // Green
			return color.NRGBA{0, 205, 0, 255}
		case 43: // Yellow
			return color.NRGBA{205, 205, 0, 255}
		case 44: // Blue
			return color.NRGBA{0, 0, 238, 255}
		case 45: // Magenta
			return color.NRGBA{205, 0, 205, 255}
		case 46: // Cyan
			return color.NRGBA{0, 205, 205, 255}
		case 47: // White
			return color.NRGBA{229, 229, 229, 255}
		case 100: // Bright Black (Gray)
			return color.NRGBA{127, 127, 127, 255}
		case 101: // Bright Red
			return color.NRGBA{255, 0, 0, 255}
		case 102: // Bright Green
			return color.NRGBA{0, 255, 0, 255}
		case 103: // Bright Yellow
			return color.NRGBA{255, 255, 0, 255}
		case 104: // Bright Blue
			return color.NRGBA{0, 0, 255, 255}
		case 105: // Bright Magenta
			return color.NRGBA{255, 0, 255, 255}
		case 106: // Bright Cyan
			return color.NRGBA{0, 255, 255, 255}
		case 107: // Bright White
			return color.NRGBA{255, 255, 255, 255}
		default:
			return color.NRGBA{0, 0, 0, 255}
		}
	}
}
