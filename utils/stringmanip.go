package utils

import (
	"bytes"
	"regexp"
	"strings"
)

// FilterIntoGroups filters []byte slices into two groups, depending on the given filter function
func FilterIntoGroups(bytelines [][]byte, filterfunc func([]byte) bool) ([][]byte, [][]byte) {
	var special, regular [][]byte
	for _, byteline := range bytelines {
		if filterfunc(byteline) {
			// Special
			special = append(special, byteline)
		} else {
			// Regular
			regular = append(regular, byteline)
		}
	}
	return special, regular
}

/*ExtractKeywords takes a source file as `data` and a list of keywords to
 * look for. Lines without keywords are returned, together with a map
 * from keywords to []bytes from the source `data`.
 *
 * The keywords in the data must be on the form "keyword: value",
 * and can be within single-line HTML comments (<-- ... -->).
 */
func ExtractKeywords(data []byte, keywordsToLookFor []string) ([]byte, map[string][]byte) {
	var (
		keywordColon      []byte
		inCommentBlock    bool
		singleLineComment bool
		lineCounter       uint64
	)
	bnl := []byte("\n")
	commentStart := []byte("<!--")
	commentEnd := []byte("-->")
	backtick := []byte("`")
	found := make(map[string][]byte)

	//if bytes.Contains(data, backtick) {
	//}

	stopLooking := false
	// Find and separate the lines starting with one of the keywords in the special map
	_, regular := FilterIntoGroups(bytes.Split(data, bnl), func(byteline []byte) bool {
		lineCounter++
		if stopLooking {
			return false
		}
		// Check if the current line has one of the special keywords
		for _, keyword := range keywordsToLookFor {
			strippedLine := bytes.TrimSpace(byteline)
			if len(strippedLine) == 0 {
				// Empty line
				return false
			}

			// If we encounter a backtick (`), stop looking for keywords (due to Markdown formatting of code)
			if bytes.Contains(strippedLine, backtick) {
				// Contains a backtick
				stopLooking = true
				return false
			}

			// Check if we are in a HTML comment block
			singleLineComment = false
			switch {
			case bytes.HasPrefix(strippedLine, commentStart) && bytes.HasSuffix(strippedLine, commentEnd):
				inCommentBlock = false
				strippedLine = bytes.TrimSpace(strippedLine[len(commentStart) : len(strippedLine)-len(commentEnd)])
				singleLineComment = true
			case bytes.HasPrefix(strippedLine, commentStart):
				inCommentBlock = true
			case bytes.HasSuffix(strippedLine, commentEnd):
				inCommentBlock = false
			}
			// fmt.Println("LINE", string(strippedLine), "IN COMMENT BLOCK", inCommentBlock, "SINGLE LINE COMMENT", singleLineComment)

			keywordColon = append([]byte(keyword), ':')
			// Lines starting with "% " can be used for specifying a title, ref pandoc
			if bytes.HasPrefix(strippedLine, []byte("% ")) {
				newTitle := bytes.TrimSpace(strippedLine[2:])
				if len(newTitle) > 0 {
					found["title"] = newTitle
					return true
				}
			}
			// Check for lines that starts with "<!--", ends with "-->" and contains the keyword and a ":"
			if singleLineComment {
				// Check if one of the relevant keywords are present
				if bytes.HasPrefix(strippedLine, keywordColon) {
					// Set (possibly overwrite) the value in the map, if the keyword is found.
					// Trim the surrounding whitespace and skip the letters of the keyword itself.
					found[keyword] = bytes.TrimSpace(strippedLine[len(keyword)+1:])
					return true
				}
				// Check for lines starting with the keyword and a ":"
			} else if bytes.HasPrefix(strippedLine, keywordColon) {
				// Check if the keyword was found either in a comment block, or on the first 10 lines
				if inCommentBlock || lineCounter < 10 {
					// Set (possibly overwrite) the value in the map, if the keyword is found.
					// Trim the surrounding whitespace and skip the letters of the keyword itself.
					found[keyword] = bytes.TrimSpace(strippedLine[len(keywordColon):])
					return true
				}
			}
		}
		// Not special
		return false
	})
	// Use the regular lines as the new data (remove the special lines)
	return bytes.Join(regular, bnl), found
}

// Infostring builds up a string on the form "functionname(arg1, arg2, arg3)"
func Infostring(functionName string, args []string) string {
	s := functionName + "("
	if len(args) > 0 {
		s += "\"" + strings.Join(args, "\", \"") + "\""
	}
	return s + ")"
}

// WriteStatus writes status messages to a string Builder
// The flags argument contains the flag names, and if they are enabled or not
func WriteStatus(sb *strings.Builder, title string, flags map[string]bool) {
	// Check that at least one of the bools are true
	found := false
	for _, value := range flags {
		if value {
			found = true
			break
		}
	}
	if !found {
		return
	}

	// Write the overview to the buffer
	sb.WriteString(title)
	sb.WriteString(":")

	// Spartan way of lining up the columns
	if len(title) < 7 {
		sb.WriteString("\t")
	}
	sb.WriteString("\t\t[")

	var enabledFlags []string
	// Add all enabled flags to the list
	for name, enabled := range flags {
		if enabled {
			enabledFlags = append(enabledFlags, name)
		}
	}
	sb.WriteString(strings.Join(enabledFlags, ", "))
	sb.WriteString("]\n")
}

// ExtractLocalImagePaths can be used to find local image filenames that are included in the given HTML
func ExtractLocalImagePaths(html string) []string {
	// Regular expression to match image paths in <img ref="...">
	r := regexp.MustCompile(`<img ref="([^"]+)"`)
	matches := r.FindAllStringSubmatch(html, -1)
	paths := make([]string, 0, len(matches))
	for _, match := range matches {
		path := match[1]
		if len(path) >= 7 && (path[:7] == "http://" || (len(path) >= 8 && path[:8] == "https://")) {
			continue
		}
		paths = append(paths, path)
	}
	return paths
}
