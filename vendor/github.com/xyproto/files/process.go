//go:build !unix

package files

import (
	"errors"
	"fmt"
	"os"
)

var errUnimplemented = errors.New("unimplemented")

// getPID tries to find the PID, given a process name, similar to pgrep
func GetPID(name string) (int64, error) {
	return 0, errUnimplemented
}

// HasProcess returns true if a valid PID for the given process name is found in /proc, similar to how pgrep works
func HasProcess(name string) bool {
	fmt.Fprintln(os.Stderr, "unimplemented")
	return false
}

// Pkill tries to find and kill all processes that match the given name.
// Returns the number of processes killed, and an error, if anything went wrong.
func Pkill(name string) (int, error) {
	return 0, errUnimplemented
}

// GetProcPath resolves and returns the specified path (e.g., "exe", "cwd") for the process identified by pid.
// It returns an error if the path cannot be resolved.
func GetProcPath(pid int, suffix string) (string, error) {
	return "", errUnimplemented
}
