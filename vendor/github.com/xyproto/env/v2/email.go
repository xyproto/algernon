package env

import (
	"os/exec"
	"strings"
)

// Email returns the most likely email address for the current user.
// It checks $EMAIL first, then "git config user.email", and finally
// returns an empty string if no email address was found.
func Email() string {
	if email := Str("EMAIL"); email != "" {
		return email
	}
	if gitEmail, err := exec.Command("git", "config", "user.email").Output(); err == nil {
		if email := strings.TrimSpace(string(gitEmail)); email != "" {
			return email
		}
	}
	return ""
}
