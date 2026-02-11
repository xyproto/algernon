//go:build unix

package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// getPID tries to find the PID, given a process name, similar to pgrep
func GetPID(name string) (int64, error) {
	lowerName := strings.ToLower(name)
	procDir, err := os.ReadDir("/proc")
	if err != nil {
		return 0, err
	}
	for _, entry := range procDir {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		exePath, err := GetProcPath(pid, "exe")
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(filepath.Base(exePath)), lowerName) {
			return int64(pid), nil
		}
	}
	return 0, os.ErrNotExist
}

// HasProcess returns true if a valid PID for the given process name is found in /proc, similar to how pgrep works
func HasProcess(name string) bool {
	pid, err := GetPID(name)
	return err == nil && pid > 0
}

// Pkill tries to find and kill all processes that match the given name.
// Returns the number of processes killed, and an error, if anything went wrong.
func Pkill(name string) (int, error) {
	lowerName := strings.ToLower(name)
	procDir, err := os.ReadDir("/proc")
	if err != nil {
		return 0, err
	}
	count := 0
	for _, entry := range procDir {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid == os.Getpid() {
			continue
		}
		exePath, err := GetProcPath(pid, "exe")
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(filepath.Base(exePath)), lowerName) {
			if err := syscall.Kill(pid, syscall.SIGKILL); err == nil {
				count++
			}
		}
	}
	if count == 0 {
		return 0, fmt.Errorf("no process named %q found", name)
	}
	return count, nil
}

// GetProcPath resolves and returns the specified path (e.g., "exe", "cwd") for the process identified by pid.
// It returns an error if the path cannot be resolved.
func GetProcPath(pid int, suffix string) (string, error) {
	return os.Readlink(fmt.Sprintf("/proc/%d/%s", pid, suffix))
}
