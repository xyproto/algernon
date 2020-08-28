package vt100

import (
	"strings"
)

// Words takes a string with words and several color-strings, like "blue". Color the
// words with the corresponding colors and return the string.
func Words(line string, colors ...string) string {
	var ok bool
	words := strings.Split(line, " ")
	// Starting out with light gray, then using the last set color for the rest of the words
	color := LightGray
	coloredWords := make([]string, len(words))
	for i, word := range words {
		if i < len(colors) {
			prevColor := color
			color, ok = LightColorMap[colors[i]]
			if !ok {
				// Use the previous color if this color string was not found
				color = prevColor
			}
		}
		coloredWords[i] = color.Get(word)
	}
	return strings.Join(coloredWords, " ")
}

// ColorString takes a string with words to be colored and another string with colors with
// which to color the words. Example strings: "hello there" and "red blue".
func ColorString(line, colors string) string {
	return Words(line, strings.Split(colors, " ")...)
}
