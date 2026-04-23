package engine

import (
	_ "embed"
	"strings"

	"github.com/orsinium-labs/enum"
)

/*

ANSI banner HOWTO
-----------------

1. Find an image.
2. Convert to png, if needed:
   convert image.jpg image.png
3. Crop and resize the image until it is approximately 20 pixels in width. Gimp works.
4. Install the transmogrify executable from the transmogrifier package (for converting to ANSI):
   npm -g install transmogrifier
5. Transform to ANSI and save under engine/assets/banner/<name>.ansi.

*/

// SplashImage is a typed ANSI banner, embedded at build time.
type SplashImage enum.Member[string]

// Banner images, embedded from engine/assets/banner/*.ansi
var (
	//go:embed assets/banner/gophereyes.ansi
	gopherEyesANSI string

	//go:embed assets/banner/whitegrid.ansi
	whiteGridANSI string

	//go:embed assets/banner/algernonpoet.ansi
	algernonPoetANSI string

	// Gopher Eyes
	GopherEyes = SplashImage{gopherEyesANSI}

	// A simple white grid on a black background
	WhiteGrid = SplashImage{whiteGridANSI}

	// From a photo of Algernon Charles Swinburne, the poet
	AlgernonPoet = SplashImage{algernonPoetANSI}

	// Select a random splash/banner image every time
	//splashImages = enum.New(GopherEyes, WhiteGrid, AlgernonPoet)
	//splashImage *SplashImage = splashImages.Choice(0)

	// Select the gopher eyes every time
	splashImage = GopherEyes
)

// Insert text while replacing tab characters
func insertText(s, tabs string, linenr, offset int, message string, removal int) string {
	tabcounter := 0
	for pos := 0; pos < len(s); pos++ {
		if s[pos] == '\t' {
			tabcounter++
		}
		if tabcounter == len(tabs)*linenr+offset {
			s = s[:pos] + message + s[pos+removal:]
			break
		}
	}
	return s
}

// Banner returns ANSI graphics with the current version number embedded in the text
func Banner(versionString, description string) string {
	tabs := "\t\t\t\t"
	s := tabs + strings.ReplaceAll("\n"+splashImage.Value, "\n", "\n"+tabs)
	parts := strings.Fields(versionString)

	// See https://github.com/shiena/ansicolor/blob/master/README.md for ANSI color code table
	s = insertText(s, tabs, 3, 2, "\x1b[37m"+parts[0]+"\x1b[0m", 1)
	s = insertText(s, tabs, 4, 1, "\x1b[90m"+parts[1]+"\x1b[0m\t", 1)
	s = insertText(s, tabs, 5, 1, "\x1b[94m"+description+"\x1b[0m", 1)
	return s
}
