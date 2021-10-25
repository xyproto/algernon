package env

import (
	"os"
	"strconv"
	"time"
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

// Int64 returns the number stored in the environment variable, or the provided default value.
func Int64(envName string, defaultValue int64) int64 {
	i64, err := strconv.ParseInt(Str(envName), 10, 64)
	if err != nil {
		return defaultValue
	}
	return i64
}

// Float64 returns the number stored in the environment variable, or the provided default value.
func Float64(envName string, defaultValue float64) float64 {
	f64, err := strconv.ParseFloat(Str(envName), 64)
	if err != nil {
		return defaultValue
	}
	return f64
}

// DurationSeconds interprets the environment variable value as seconds
// and returns a time.Duration. The given default number is interpreted
// as the number of seconds.
func DurationSeconds(envName string, defaultValue int64) time.Duration {
	i64, err := strconv.ParseInt(Str(envName), 10, 64)
	if err != nil {
		return time.Duration(defaultValue) * time.Second
	}
	return time.Duration(i64) * time.Second
}

// DurationMinutes interprets the environment variable value as minutes
// and returns a time.Duration. The given default number is interpreted
// as the number of minutes.
func DurationMinutes(envName string, defaultValue int64) time.Duration {
	i64, err := strconv.ParseInt(Str(envName), 10, 64)
	if err != nil {
		return time.Duration(defaultValue) * time.Minute
	}
	return time.Duration(i64) * time.Minute
}

// DurationHours interprets the environment variable value as hours
// and returns a time.Duration. The given default number is interpreted
// as the number of hours.
func DurationHours(envName string, defaultValue int64) time.Duration {
	i64, err := strconv.ParseInt(Str(envName), 10, 64)
	if err != nil {
		return time.Duration(defaultValue) * time.Hour
	}
	return time.Duration(i64) * time.Hour
}
