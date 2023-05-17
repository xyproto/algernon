// Package env provides convenience functions for retrieving data from environment variables
package env

import (
	"errors"
	"os"
	"runtime"
	"strings"
	"sync"
)

// Setting useCaching to true makes the functions below stop callig os.Getenv,
// only call os.Environ once and then use the environment map to read variables from.
var (
	useCaching  = true
	environment map[string]string
	mut         sync.RWMutex
)

// getenv calls os.Getenv if useCaching is false.
// If useCaching is true, the environment map is initialized (if needed),
// and the value is fetched from the map instead of calling os.Getenv.
func getenv(name string) string {
	if !useCaching {
		return os.Getenv(name)
	}
	mut.RLock()
	if environment == nil {
		mut.RUnlock()
		Load()
		mut.RLock()
	}
	value, ok := environment[name]
	mut.RUnlock()
	if !ok {
		return ""
	}
	return value
}

// Load reads all environment variables into the environment map. It also instructs env to use the cache.
// If a program uses os.Setenv, then Load() should be called after that, in order to read the new values.
// This function can be used both as an "init and enable cache" function and as a "reload" function.
func Load() {
	mut.Lock()
	environment = make(map[string]string)
	// Read all the environment variables into the map
	for _, statement := range os.Environ() {
		pair := strings.SplitN(statement, "=", 2)
		environment[pair[0]] = pair[1]
	}
	mut.Unlock()
	useCaching = true
}

// Unload clears the cache and configures env to not use the cache.
func Unload() {
	environment = nil
	useCaching = false
}

// Set calls os.Setenv.
// If caching is enabled, the value in the environment map is also set and there
// is no need to call Load() to re-read the environment variables from the system.
func Set(name, value string) error {
	if useCaching {
		mut.RLock()
		if environment == nil {
			mut.RUnlock()
			Load()
		} else {
			mut.RUnlock()
		}
		mut.Lock()
		if value == "" {
			delete(environment, name)
		} else {
			environment[name] = value
		}
		mut.Unlock()
	}
	return os.Setenv(name, value)
}

// Unset will clear an environment variable by calling os.Setenv(name, "").
// The cache entry will also be cleared if useCaching is true.
func Unset(name string) error {
	return Set(name, "")
}

// userHomeDir is the same as os.UserHomeDir, except that "getenv" is called instead of "os.Getenv",
// and the two switches are refactored into one.
func userHomeDir() (string, error) {
	env, enverr := "HOME", "$HOME"
	switch runtime.GOOS {
	case "android":
		return "/sdcard", nil
	case "ios":
		return "/", nil
	case "windows":
		env, enverr = "USERPROFILE", "%userprofile%"
	case "plan9":
		env, enverr = "home", "$home"
	}
	if v := getenv(env); v != "" {
		return v, nil
	}
	return "", errors.New(enverr + " is not defined")
}
