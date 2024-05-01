package ollamaclient

import "strings"

// Massage will try to extract a shorter message from a longer LLM output
// using pretty "hacky" string manipulation techniques.
func Massage(generatedOutput string) string {
	s := generatedOutput
	// Keep the part after ":", if applicable
	if strings.Contains(s, ":") {
		parts := strings.SplitN(s, ":", 2)
		rightPart := strings.TrimSpace(parts[1])
		if rightPart != "" {
			s = rightPart
		}
	}
	// Keep the part within double quotes, if applicable
	if strings.Count(s, "\"") == 2 {
		parts := strings.SplitN(s, "\"", 3)
		rightPart := strings.TrimSpace(parts[1])
		if rightPart != "" {
			s = rightPart
		}
	}
	// Keep the part within single quotes, if applicable
	if strings.Contains(s, " '") && strings.HasSuffix(s, "'") {
		parts := strings.SplitN(s, "'", 3)
		rightPart := strings.TrimSpace(parts[1])
		if rightPart != "" {
			s = rightPart
		}
	}
	// Keep the part after ":", if applicable
	if strings.Contains(s, ":") {
		parts := strings.SplitN(s, ":", 2)
		rightPart := strings.TrimSpace(parts[1])
		if rightPart != "" {
			s = rightPart
		}
	}
	// Remove stray quotes
	s = strings.TrimPrefix(s, "\"")
	s = strings.TrimPrefix(s, "'")
	s = strings.TrimSuffix(s, "\"")
	s = strings.TrimSuffix(s, "'")
	// Keep the last line
	if strings.Count(s, "\n") > 1 {
		lines := strings.Split(s, "\n")
		s = lines[len(lines)-1]
	}
	// Keep the part after the last ".", if applicable
	if strings.Contains(s, ".") {
		parts := strings.SplitN(s, ".", 2)
		rightPart := strings.TrimSpace(parts[1])
		if rightPart != "" {
			s = rightPart
		}
	}
	// Keep the part before the exclamation mark, if applicable
	if strings.Contains(s, "!") {
		parts := strings.SplitN(s, "!", 2)
		leftPart := strings.TrimSpace(parts[0]) + "!"
		if leftPart != "" {
			s = leftPart
		}
	}
	// No results so far, just return the original string, trimmed
	if len(s) < 4 {
		return strings.TrimSpace(generatedOutput)
	}
	// Let the first letter be uppercase
	s = strings.ToUpper(string([]rune(s)[0])) + string([]rune(s)[1:])

	// Trim spaces
	s = strings.TrimSpace(s)

	// Remove trailing </s>
	s = strings.TrimSuffix(s, "</s>")

	// Trim spaces
	s = strings.TrimSpace(s)

	return s
}
