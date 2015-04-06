// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris

package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"
	"strings"
)

/*

ANSI banner HOWTO
-----------------

1. Find an image.
2. Convert to png, if needed:
   convert image.jpg image.png
3. Crop and resize the image until it is approximately 22 pixels in width. Gimp works.
4. Install the transmogrify executable from the transmogrifier package (for converting to ANSI):
   npm -g install transmogrifier
5. Transform, compress and encode:
   transmogrify image.png | gzip -c -9 | base64 -w0 > output.b64
6. Copy and paste the base64 encoded data as the image constant below.

*/

// Drawn in Inkscape
const image = `H4sIAKy5IlUCA91XSQ6DMAy85wu95Ak4K4in9A38/1qpakMb2yQxSYXKjRGKZ8ZLjNb5c7u7efWrsXF7vizbCwlTDsQSYOzyRlAcrTQXG0UC6IMYO3N8FMMFQtEGFMVJuO3HENQS/8ArYhO3SzhW9pGthPhSiiXSeAKxQp7fD1PsZzAhogYTdTlRXzQAkS4D2FdZ9RLt9ZVZPmmdoiWTFecXYbIIIRJKSa2wuqJdLd0f5XknTOOU2xhhkI2hQ/VeBaBGgeNTmebA0lDpP6jrKKzreVxd97pdcWFjubjU0RAGsP85lu2gsZyiqbarNQzcE2r67syi4JvUyJc7vMsRV9PZ7e7Ka/DB7tn9h+ABr+hfa40MAAA=`

// From a photo of Algernon Charles Swinburne, the poet
//const image = `H4sICLDeClUCA2FsZ2Vybm9uLmFuc2kA1Vi5ccQwDMzZwiVXAgkCBDVXytVw/aeObM8IkLCCqcChdsQPi8X3ePN8yYuYPs/n483b53GM9B0yg+9WKUSI96tIqkH2q9Tssu1PYjH/SPgi4gb8M8z9zCrpAPJzn/INdXNFCKEUwgaZgBNobI6+AcgA7jOA+4zQWdqckfv8utwZFQo8S2+jC/Bnqxzrz5ZkB0FeoYB2KKEdtVQw4CsaH+7QxSkqehxsREK/9MjRlE56who2FMuGyAI5HUEo9dKYC0f/CDkNCGtAeOzIqhEiepw9yslRIyNSqwqbkq1/O6viJGiYaDndaAZxkrT8DSmHNYKVklNGMPAKhJsR7mwzYMwEwlUHSrcW3mUZDw3KiDNhdmxnQGxxcFrDjEUYQJbpYabq4YbXBidx29ZJdQEL9uFNVttvNQ9x2csJxBNwBTLjvNIR/SPkxGLlOB0oYFWOE+pEohkBnQPdZp9+f+BB2CknBRkD4b4BJouHB94ABEgtkghHUL6mJSHhIlJOytMazJSSxR8D/TTUVAL6d87ShLazyklSgVT4FahqJSSwA31Lan7lVNTzeiXsSAuKzYtyhfdSzYyigAGuYzFn556aN9SwT0lxcWdpa7hoqRbbmadtmTYcilCS2QcZ65ok5KmAUuxc+qd8AZBU6SmjGAAA`

// Decompress text that has first been gzipped and then base64 encoded
func decompress(asciigfx string) string {
	unbasedBytes, err := base64.StdEncoding.DecodeString(asciigfx)
	if err != nil {
		panic("Could not decode base64: " + err.Error())
	}
	buf := bytes.NewBuffer(unbasedBytes)
	decompressorReader, err := gzip.NewReader(buf)
	if err != nil {
		panic("Could not read buffer: " + err.Error())
	}
	decompressedBytes, err := ioutil.ReadAll(decompressorReader)
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

// Return ANSI graphics with the current version number embedded in the text
func banner() string {
	s := decompress(image)
	tabs := "\t\t\t\t"
	s = tabs + strings.Replace(s, "\n", "\n"+tabs, -1)
	s = insertText(s, tabs, 5, 2, "\x1b[32;1m"+version_string+"\x1b[0m", 1)
	s = insertText(s, tabs, 7, 1, "\x1b[30;1m"+description+"\x1b[0m", 2)
	return s
}
