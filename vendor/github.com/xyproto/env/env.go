package env

import (
	"os"
	"strconv"
)

// Str does the same as os.Getenv, but allows the user to provide a default value (optional).
// Only the first optional argument is used, the rest is discarded.
func Str(name string, optionalDefault ...string) string {
	// Retrieve the environment variable as a (possibly empty) string
	value := os.Getenv(name)

	// If empty and a default value was provided, return that
	if value == "" && len(optionalDefault) > 0 {
		return optionalDefault[0]
	}

	// If not, return the value of the environment variable
	return value
}

// AsBool can be used to interpret a string value as either true or false. Examples of true values are "yes" and "1".
func AsBool(s string) bool {
	switch s {
	case "yes", "1", "true", "YES", "TRUE", "True", "Yes", "Y", "y", "enable", "Enable", "ENABLE", "enabled", "Enabled", "ENABLED", "affirmative", "Affirmative", "AFFIRMATIVE":
		return true
	case "", "no", "0", "false", "NO", "FALSE", "False", "No", "N", "n", "disable", "Disable", "DISABLE", "disabled", "Disabled", "DISABLED", "denied", "Denied", "DENIED":
		fallthrough
	default:
		return false
	}
}

// Bool returns the bool value of the given environment variable name.
// Returns false if it is not declared or empty.
func Bool(envName string) bool {
	return AsBool(Str(envName))
}

// Has returns true if the given environment variable name is set and not empty.
func Has(envName string) bool {
	return Str(envName) != ""
}

// Int returns the number stored in the environment variable, or the provided default value.
func Int(envName string, defaultValue int) int {
	i, err := strconv.Atoi(Str(envName))
	if err != nil {
		return defaultValue
	}
	return i
}
