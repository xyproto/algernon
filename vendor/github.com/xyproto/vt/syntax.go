package vt

import (
	"strings"
)

// colorWrite writes a colored string to the given strings.Builder
func colorWrite(sb *strings.Builder, s string, colorIndex int) {
	switch colorIndex % 4 {
	case 0:
		sb.WriteString(s)
	case 1:
		sb.WriteString(White.Get(s))
	case 2:
		sb.WriteString(Magenta.Get(s))
	case 3:
		sb.WriteString(Cyan.Get(s))
	}
}

// Colorize comments in gray. Colorize (){}[] in an alternating way.
// This provides simple/rudimentary syntax highlighting.
func Colorize(line string) string {
	var sb strings.Builder
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") || strings.Contains(trimmed, "Leaving directory") || strings.Contains(trimmed, "Entering directory") || strings.Contains(trimmed, "Nothing to be done") {
		return Black.Get(line)
	}
	if strings.Contains(trimmed, "(ignored)") {
		return LightYellow.Get(line)
	}
	if strings.Contains(trimmed, "In file included from") {
		return LightBlue.Get(line)
	}
	if strings.HasPrefix(trimmed, "OPKG ") {
		return LightMagenta.Get(line)
	}
	if strings.HasPrefix(trimmed, "STRIP ") {
		return Cyan.Get(line)
	}
	if strings.Contains(trimmed, "***") || strings.Contains(trimmed, "===") || strings.Contains(trimmed, "No such file or directory") {
		return Red.Get(line)
	}
	if (strings.HasPrefix(trimmed, "*") && strings.HasSuffix(trimmed, "*")) || (strings.HasPrefix(trimmed, "-") && strings.HasSuffix(trimmed, "-")) || (strings.HasPrefix(trimmed, "=") && strings.HasSuffix(trimmed, "=")) || strings.Contains(trimmed, ">>>") {
		return White.Get(line)
	}

	if strings.Contains(trimmed, ": In function ") || strings.Contains(trimmed, ": In member function ") {
		elements := strings.SplitN(trimmed, ":", 2)
		fn := elements[0]
		msg := elements[1]
		if strings.Count(msg, "'") >= 2 {
			parts := strings.SplitN(msg, "'", 3)
			a := parts[0]
			signature := parts[1]
			b := parts[2]
			return LightYellow.Get(fn) + ":" + White.Get(a) + "'" + LightBlue.Get(signature) + "'" + White.Get(b)
		}
	}

	var (
		rainbowLine    strings.Builder
		curlyLevel     int
		bracketLevel   int
		parLevel       int
		closing        bool
		colorIndex     int
		word           string
		changed        bool
		singleToggle   bool
		quotingChanged bool
	)
	for _, c := range line {
		quotingChanged = false
		switch c {
		case '\'':
			if strings.Count(line, "'")%2 == 0 && !strings.Contains(line, "n't") {
				singleToggle = !singleToggle
				quotingChanged = true
				changed = true
				closing = false
			} else {
				changed = false
				closing = false
			}
		case '{':
			curlyLevel++
			colorIndex++
			changed = true
			closing = false
		case '[':
			bracketLevel++
			colorIndex++
			changed = true
			closing = false
		case '(':
			parLevel++
			colorIndex++
			changed = true
			closing = false
		case '}':
			curlyLevel--
			closing = true
			changed = true
		case ']':
			bracketLevel--
			closing = true
			changed = true
		case ')':
			parLevel--
			closing = true
			changed = true
		default:
			changed = false
			closing = false
		}
		// If the level changed, output the word we've got so far
		if changed {
			// THIS IS THE PLACE TO PROCESS THE THING BETWEEN THE BRACKETS
			colorWrite(&rainbowLine, word, colorIndex)
			// Then bump the color, if closing
			prevColor := colorIndex
			if closing && (colorIndex > 0) {
				colorIndex--
			}
			// Or bump the color, if the quoting changed
			if c == '\'' && quotingChanged {
				if singleToggle {
					colorIndex++
				} else {
					colorIndex--
				}
			}
			if c == '\'' && singleToggle {
				prevColor = colorIndex
			}
			// Then output the opening/closing element
			colorWrite(&rainbowLine, string(c), prevColor)
			// Then reset the word
			word = ""
		} else if c == ' ' {
			word += string(c)
			// Then output the opening/closing thing
			colorWrite(&rainbowLine, word, colorIndex)
			word = ""
		} else {
			// The level did not change, continue to collect the word
			word += string(c)
		}
	}
	if word != "" {
		// THIS IS THE SECOND PLACE TO PROCESS THE THING BETWEEN THE BRACKETS
		colorWrite(&rainbowLine, word, colorIndex)
	}

	line = rainbowLine.String()

	for i, word := range strings.Split(line, " ") {
		if i > 0 {
			sb.WriteString(" ")
		}
		switch word {
		case "\"GET", "\"POST":
			sb.WriteString(Cyan.Get(string(word[0])) + Blue.Get(word[1:]))
			continue
		}
		switch strings.ToLower(word) {
		case "error", "error:", "errors", "errors:", "abort", "quit", "no such file or directory", "failed", "failed:", "failed,":
			sb.WriteString(LightRed.Get(word))
		case "warning", "warning:", "removed", "deleted", "erased", "o":
			sb.WriteString(LightYellow.Get(word))
		case "note", "note:":
			sb.WriteString(LightGreen.Get(word))
		case "cc", "cxx", "ld", "rm", "make", "strip", "ccgi", "opkg", "install", "run", "running", "move", "format", "upgrading", "gcc", "g++", "clang", "clang++", "complete":
			sb.WriteString(Blue.Get(word))
		case "upgraded", "installed", "moved", "ran", "formatted", "cp", "mv", "ln":
			sb.WriteString(Magenta.Get(word))
		case "=", "==", ":=", "tar", "zip":
			sb.WriteString(White.Get(word))
		default:
			sb.WriteString(Cyan.Get(word))
		}
	}

	return sb.String()
}
