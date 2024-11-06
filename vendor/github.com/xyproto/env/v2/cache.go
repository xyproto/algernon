// Package env provides convenience functions for retrieving data from environment variables
package env

import (
	"fmt"
	"os"
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
	for _, keyAndValue := range os.Environ() {
		pair := strings.SplitN(keyAndValue, "=", 2)
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
func Set(name string, values ...string) error {
	var value string
	switch len(values) {
	case 0:
		value = "1" // use "1" as the environment var value if none is given
	case 1:
		value = values[0] // use the given value if only one is given
	default:
		return fmt.Errorf("can only set %s to a maximum of 1 value", name)
	}
	if useCaching {
		mut.RLock()
		if environment == nil {
			mut.RUnlock()
			Load()
		} else {
			mut.RUnlock()
		}
		mut.Lock()
		environment[name] = value
		mut.Unlock()

	}
	return os.Setenv(name, value)
}

// Unset will clear an environment variable by calling os.Setenv(name, "").
// The cache entry will also be cleared if useCaching is true.
func Unset(name string) error {
	if useCaching {
		mut.RLock()
		if environment == nil {
			mut.RUnlock()
			Load()
		} else {
			mut.RUnlock()
		}
		mut.Lock()
		delete(environment, name)
		mut.Unlock()
	}
	return os.Setenv(name, "")
}
