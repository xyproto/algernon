package burnfont

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strings"
)

// SavePNG will render the given text to an image and save
// it as a .png image (using the given filename).
func SavePNG(text, filename string) error {

	spacesPerTab := 4

	// Find the longest line
	maxlen := 0
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if len(line) > maxlen {
			maxlen = len(line)
		}
	}

	lineHeight := 14
	marginRight := 4 * lineHeight
	width := 8*(maxlen+1) + marginRight
	height := (len(lines) * lineHeight) + 2*lineHeight

	dimension := image.Rectangle{image.Point{}, image.Point{width, height}}

	textImage := image.NewRGBA(dimension)
	finalImage := image.NewRGBA(dimension)

	darkgray := color.NRGBA{0x10, 0x10, 0x10, 0xff}
	white := color.NRGBA{0xff, 0xff, 0xff, 0xff}

	draw.Draw(finalImage, finalImage.Bounds(), &image.Uniform{white}, image.Point{}, draw.Src)

	// For each line of this text document, draw the string to an image
	var contents string
	for i, line := range lines {
		// Expand tabs for each line
		contents = strings.Replace(line, "\t", strings.Repeat(" ", spacesPerTab), -1)
		// Draw the string to the textImage
		DrawString(textImage, lineHeight, (i+1)*lineHeight, contents, darkgray)
	}

	// Now overlay the text image on top of the final image with the background color
	draw.Draw(finalImage, finalImage.Bounds(), textImage, image.Point{}, draw.Over)

	// Write the PNG file
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	return png.Encode(f, finalImage)
}
