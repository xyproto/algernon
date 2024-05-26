package engine

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadIgnoreFile reads the ignore patterns from the ignore.txt file.
func ReadIgnoreFile(ignoreFilePath string) ([]string, error) {
	var patterns []string
	file, err := os.Open(ignoreFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}
	return patterns, scanner.Err()
}

// ShouldIgnore checks if a given filename should be ignored based on the provided patterns.
// A warning will be printed if there are errors with an ignore pattern.
func ShouldIgnore(filename string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %s contains an unmatchable pattern %s: %v\n", filename, pattern, err)
			continue
		}
		if matched {
			return true
		}
	}
	return false
}
