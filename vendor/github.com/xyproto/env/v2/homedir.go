//go:build !android && !ios && !plan9 && !windows

package env

import "fmt"

const home = "HOME"

// userHomeDir is the same as os.UserHomeDir, except that "getenv" is called instead of "os.Getenv"
// (for caching). Also, the two switch statements are combined into just one.
func userHomeDir() (string, error) {
	value := getenv(home)
	if value == "" {
		return "", fmt.Errorf("$%s is not defined", home)
	}
	return value, nil
}
