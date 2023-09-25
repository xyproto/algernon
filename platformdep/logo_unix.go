//go:build unix

package platformdep

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
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
5. Transform, compress and encode:
   transmogrify image.png | gzip -c -9 | base64 -w0 > output.b64
6. Copy and paste the base64 encoded data as the image constant below.

*/

type SplashImage enum.Member[string]

var (

	// Gopher Eyes
	GopherEyes = SplashImage{`H4sIAJB76FoCA9WWTRKDIAxG91zBjUcQSBTGo/QM3n/bbvozfFRCRGp3nW+q5PEwZLhRWHm17LdxHG4Ut+GZJIHz9oeJtUu3xDGXk/NIGYKXG3NITgYCE0oTWhpD1LiBp5RuKGjQVdbf6Obre9AfT+lfZmBwAk7Yr8dPDUNHW1BhBl1yBn1xT3fRTUXNSxtZ9H+yJOdUIov6yXJcXh4oEMs7wXuvZqv8lUjISUEuaYOypL5iO7lGV3E/V3hjAoRAFS6tmy8yMxF1m5KsYLewQMIGApc+47mIgnGQ6u0ph6Q0CFEjSznSNkpyX81cHFQ4loWSLy+lbJSN0PdUmEM9JHMUdImgp0E14bTkYuSfuu6KgLzwGw8AAA==`}

	// A simple white grid on a black background
	WhiteGrid = SplashImage{`H4sIAK/e01kCA5OONrGwNrU2MjbMVVCQjjaxzJUeFaFYhAunIkMzQgKjekaDkt5BOSoymtlHM/toZh8VIZjZAT4BWy4xCQAA`}

	// From a photo of Algernon Charles Swinburne, the poet
	AlgernonPoet = SplashImage{`H4sICLDeClUCA2FsZ2Vybm9uLmFuc2kA1Vi5ccQwDMzZwiVXAgkCBDVXytVw/aeObM8IkLCCqcChdsQPi8X3ePN8yYuYPs/n483b53GM9B0yg+9WKUSI96tIqkH2q9Tssu1PYjH/SPgi4gb8M8z9zCrpAPJzn/INdXNFCKEUwgaZgBNobI6+AcgA7jOA+4zQWdqckfv8utwZFQo8S2+jC/Bnqxzrz5ZkB0FeoYB2KKEdtVQw4CsaH+7QxSkqehxsREK/9MjRlE56who2FMuGyAI5HUEo9dKYC0f/CDkNCGtAeOzIqhEiepw9yslRIyNSqwqbkq1/O6viJGiYaDndaAZxkrT8DSmHNYKVklNGMPAKhJsR7mwzYMwEwlUHSrcW3mUZDw3KiDNhdmxnQGxxcFrDjEUYQJbpYabq4YbXBidx29ZJdQEL9uFNVttvNQ9x2csJxBNwBTLjvNIR/SPkxGLlOB0oYFWOE+pEohkBnQPdZp9+f+BB2CknBRkD4b4BJouHB94ABEgtkghHUL6mJSHhIlJOytMazJSSxR8D/TTUVAL6d87ShLazyklSgVT4FahqJSSwA31Lan7lVNTzeiXsSAuKzYtyhfdSzYyigAGuYzFn556aN9SwT0lxcWdpa7hoqRbbmadtmTYcilCS2QcZ65ok5KmAUuxc+qd8AZBU6SmjGAAA`}

	splashImages = enum.New(GopherEyes, WhiteGrid, AlgernonPoet)

	// Select a random splash/banner image every time
	//splashImage *SplashImage = splashImages.Choice(0)

	// Select the gopher eyes every time
	splashImage = GopherEyes
)

// Decompress text that has first been gzipped and then base64 encoded
func decompressImage(asciigfx string) string {
	unbasedBytes, err := base64.StdEncoding.DecodeString(asciigfx)
	if err != nil {
		panic("Could not decode base64: " + err.Error())
	}
	buf := bytes.NewBuffer(unbasedBytes)
	decompressorReader, err := gzip.NewReader(buf)
	if err != nil {
		panic("Could not read buffer: " + err.Error())
	}
	decompressedBytes, err := io.ReadAll(decompressorReader)
	decompressorReader.Close()
	if err != nil {
		panic("Could not decompress: " + err.Error())
	}
	return string(decompressedBytes)
}

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
	s := tabs + strings.ReplaceAll("\n"+decompressImage(splashImage.Value), "\n", "\n"+tabs)
	parts := strings.Fields(versionString)

	// See https://github.com/shiena/ansicolor/blob/master/README.md for ANSI color code table
	s = insertText(s, tabs, 3, 2, "\x1b[37m"+parts[0]+"\x1b[0m", 1)
	s = insertText(s, tabs, 4, 1, "\x1b[90m"+parts[1]+"\x1b[0\t", 1)
	s = insertText(s, tabs, 5, 1, "\x1b[94m"+description+"\x1b[0m", 1)
	return s
}
