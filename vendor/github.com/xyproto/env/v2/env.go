// Package env provides convenience functions for retrieving data from environment variables
package env

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Str does the same as os.Getenv, but allows the user to provide a default value (optional).
// Only the first optional argument is used, the rest is discarded.
func Str(name string, optionalDefault ...string) string {
	// Retrieve the environment variable as a (possibly empty) string
	value := getenv(name)

	// If empty and a default value was provided, return that
	if value == "" && len(optionalDefault) > 0 {
		return optionalDefault[0]
	}

	// If not, return the value of the environment variable
	return value
}

// File does the same as Str, but expands a leading "~" or "$HOME" string to the home
// directory of the current user.
func File(name string, optionalDefault ...string) string {
	return ExpandUser(Str(name, optionalDefault...))
}

// Dir does the same as File
func Dir(name string, optionalDefault ...string) string {
	return ExpandUser(Str(name, optionalDefault...))
}

// Path returns the elements in the $PATH environment variable
func Path() []string {
	return filepath.SplitList(Str("PATH"))
}

// StrAlt will return the string value of the first given environment variable name,
// or, if that is not available, use the string value of the second given environment variable.
// If none are available, the optional default string is returned.
func StrAlt(name1, name2 string, optionalDefault ...string) string {
	// Retrieve the environment variable as a (possibly empty) string
	value := getenv(name1)

	// If it is empty, try the second name
	if value == "" {
		value = getenv(name2)
	}

	// If empty and a default value was provided, return that
	if value == "" && len(optionalDefault) > 0 {
		return optionalDefault[0]
	}

	// If not, return the non-empty value
	return value
}

// Bool returns the bool value of the given environment variable name.
// Returns false if it is not declared or empty.
func Bool(envName string) bool {
	return AsBool(Str(envName))
}

// BoolSimple returns the bool value of the given environment variable name.
// Returns false if it is not declared or empty. Only "1" is true.
func BoolSimple(envName string) bool {
	return AsBoolSimple(Str(envName))
}

// Has returns true if the given environment variable name is set and not empty.
func Has(envName string) bool {
	return Str(envName) != ""
}

// No returns true if the given environment variable name is not set or empty.
func No(envName string) bool {
	return Str(envName) == ""
}

// Is returns true if the given environment variable is the given string value.
// The whitespace of both values are trimmed before the comparison.
func Is(envName, value string) bool {
	return strings.TrimSpace(Str(envName)) == strings.TrimSpace(value)
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

// Int32 returns the number stored in the environment variable, or the provided default value.
func Int32(envName string, defaultValue int32) int32 {
	i32, err := strconv.ParseInt(Str(envName), 10, 32)
	if err != nil {
		return defaultValue
	}
	return int32(i32)
}

// Int16 returns the number stored in the environment variable, or the provided default value.
func Int16(envName string, defaultValue int16) int16 {
	i16, err := strconv.ParseInt(Str(envName), 10, 16)
	if err != nil {
		return defaultValue
	}
	return int16(i16)
}

// Int8 returns the number stored in the environment variable, or the provided default value.
func Int8(envName string, defaultValue int8) int8 {
	i8, err := strconv.ParseInt(Str(envName), 10, 8)
	if err != nil {
		return defaultValue
	}
	return int8(i8)
}

// UInt returns the number stored in the environment variable, or the provided default value.
func UInt(envName string, defaultValue uint) uint {
	ui, err := strconv.ParseUint(Str(envName), 10, 64)
	if err != nil {
		return defaultValue
	}
	return uint(ui)
}

// UInt64 returns the number stored in the environment variable, or the provided default value.
func UInt64(envName string, defaultValue uint64) uint64 {
	ui64, err := strconv.ParseUint(Str(envName), 10, 64)
	if err != nil {
		return defaultValue
	}
	return ui64
}

// UInt32 returns the number stored in the environment variable, or the provided default value.
func UInt32(envName string, defaultValue uint32) uint32 {
	ui32, err := strconv.ParseUint(Str(envName), 10, 32)
	if err != nil {
		return defaultValue
	}
	return uint32(ui32)
}

// UInt16 returns the number stored in the environment variable, or the provided default value.
func UInt16(envName string, defaultValue uint16) uint16 {
	ui16, err := strconv.ParseUint(Str(envName), 10, 16)
	if err != nil {
		return defaultValue
	}
	return uint16(ui16)
}

// UInt8 returns the number stored in the environment variable, or the provided default value.
func UInt8(envName string, defaultValue uint8) uint8 {
	ui8, err := strconv.ParseUint(Str(envName), 10, 8)
	if err != nil {
		return defaultValue
	}
	return uint8(ui8)
}

// Float64 returns the number stored in the environment variable, or the provided default value.
func Float64(envName string, defaultValue float64) float64 {
	f64, err := strconv.ParseFloat(Str(envName), 64)
	if err != nil {
		return defaultValue
	}
	return f64
}

// Float32 returns the number stored in the environment variable, or the provided default value.
func Float32(envName string, defaultValue float32) float32 {
	f32, err := strconv.ParseFloat(Str(envName), 32)
	if err != nil {
		return defaultValue
	}
	return float32(f32)
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

// Contains checks if the given environment variable contains the given string
func Contains(envName string, value string) bool {
	return strings.Contains(Str(envName), value)
}

// True checks if the given string is likely to be interpreted as a "true" value
func True(s string) bool {
	switch s {
	case "1", "ABSOLUTELY", "AFFIRMATIVE", "Absolutely", "Affirmative", "ENABLE", "ENABLED", "Enable", "Enabled", "POSITIVE", "Positive", "T", "TRUE", "True", "Y", "YES", "Yes", "absolutely", "affirmative", "enable", "enabled", "positive", "t", "true", "y", "yes":
		return true
	}
	return false
}

// False checks if the given string is likely to be interpreted as a "false" value
func False(s string) bool {
	switch s {
	case "", "0", "BLANK", "Blank", "DENIED", "DISABLE", "DISABLED", "Denied", "Disable", "Disabled", "F", "FALSE", "False", "N", "NEGATIVE", "NIL", "NO", "NOPE", "NULL", "Negative", "Nil", "No", "Nope", "Null", "blank", "denied", "disable", "disabled", "f", "false", "n", "negative", "nil", "no", "nope", "null":
		return true
	}
	return false
}

// AsBool can be used to interpret a string value as either true or false. Examples of true values are "yes" and "1".
func AsBool(s string) bool {
	return True(s)
}

// AsBoolSimple can be used to interpret a string value as either true or false. Only "1" is true, anything else is false.
func AsBoolSimple(s string) bool {
	return s == "1"
}

// CurrentUser returns the value of LOGNAME, USER or just the string "user", in that order
func CurrentUser() string {
	return StrAlt("LOGNAME", "USER", "user")
}

// HomeDir returns the path to the home directory of the user, if available.
// If not available, the username is to construct a path starting with /home/.
// If a username is not available, then "/tmp" is returned.
// The returned string is what the home directory should have been named, if it would have existed.
// No checks are made for if the directory exists.
func HomeDir() string {
	if homeDir, err := userHomeDir(); err == nil { // success, use the home directory
		return homeDir
	}
	userName := CurrentUser()
	switch userName {
	case "root":
		// If the user name is "root", use /root
		return "/root"
	case "":
		// If the user name is not available, use either $HOME or /tmp
		return Str("HOME", "/tmp")
	default:
		// Use $HOME if it's available, and a constructed home directory path if not
		return Str("HOME", "/home/"+userName)
	}
}

// ExpandUser replaces a leading ~ or $HOME with the path
// to the home directory of the current user.
// If no expansion is done, then the original given path is returned.
func ExpandUser(path string) string {
	// this is a simpler alternative to using os.UserHomeDir (which requires Go 1.12 or later)
	if strings.HasPrefix(path, "~") {
		// Expand ~ to the home directory
		path = strings.Replace(path, "~", HomeDir(), 1)
	} else if strings.HasPrefix(path, "$HOME") {
		// Expand a leading $HOME variable to the home directory
		path = strings.Replace(path, "$HOME", HomeDir(), 1)
	}
	// Return the original given path
	return path
}

// Environ returns either the cached environment, or os.Environ()
func Environ() []string {
	if useCaching {
		var xs []string
		mut.RLock()
		defer mut.RUnlock()
		for k, v := range environment {
			xs = append(xs, k+"="+v)
		}
		return xs
	}
	return os.Environ()
}

// Keys returns the all the environment variable names as a sorted string slice
func Keys() []string {
	var keys []string
	if useCaching {
		mut.RLock()
		defer mut.RUnlock()
		for k := range environment {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return keys
	}
	for _, keyAndValue := range os.Environ() {
		pair := strings.SplitN(keyAndValue, "=", 2)
		keys = append(keys, pair[0])
	}
	sort.Strings(keys)
	return keys
}

// Map returns the current environment variables as a map from name to value
func Map() map[string]string {
	if useCaching {
		return environment
	}
	m := make(map[string]string)
	for _, keyAndValue := range os.Environ() {
		pair := strings.SplitN(keyAndValue, "=", 2)
		m[pair[0]] = pair[1]
	}
	return m
}
