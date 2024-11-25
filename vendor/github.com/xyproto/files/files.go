package files

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xyproto/binary"
	"github.com/xyproto/env/v2"
)

// cache for storing file existence results
var (
	cacheMutex      sync.RWMutex
	existsCache     = make(map[string]bool)
	whichCache      = make(map[string]string)
	executableCache = make(map[string]bool)
)

// Exists checks if the given path exists
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsFile checks if the given path exists and is a regular file
func IsFile(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular()
}

// IsSymlink checks if the given path exists and is a symbolic link
func IsSymlink(path string) bool {
	fi, err := os.Lstat(path)
	return err == nil && fi.Mode()&os.ModeSymlink != 0
}

// IsFileOrSymlink checks if the given path exists and is a regular file or a symbolic link
func IsFileOrSymlink(path string) bool {
	// use Lstat instead of Stat to avoid following the symlink
	fi, err := os.Lstat(path)
	return err == nil && (fi.Mode().IsRegular() || (fi.Mode()&os.ModeSymlink != 0))
}

// IsDir checks if the given path exists and is a directory
func IsDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsDir()
}

// Which tries to find the given executable name in the $PATH
// Returns an empty string if not found.
func Which(executable string) string {
	if p, err := exec.LookPath(executable); err == nil { // success
		return p
	}
	return ""
}

// WhichCached tries to find the given executable name in the $PATH, using a cache for faster access.
// Assumes that the $PATH environment variable has not changed since the last check.
func WhichCached(executable string) string {
	cacheMutex.RLock()
	cachedResult, cached := whichCache[executable]
	cacheMutex.RUnlock()
	if cached {
		return cachedResult
	}
	// If not cached, perform the lookup
	path := Which(executable)
	// Cache the result
	cacheMutex.Lock()
	whichCache[executable] = path
	cacheMutex.Unlock()
	return path
}

// PathHas checks if the given executable is in $PATH
func PathHas(executable string) bool {
	_, err := exec.LookPath(executable)
	return err == nil
}

// PathHasCached checks if the given executable is in $PATH (looks in the cache first and then caches the result)
func PathHasCached(executable string) bool {
	return WhichCached(executable) != ""
}

// BinDirectory will check if the given filename is in one of these directories:
// /bin, /sbin, /usr/bin, /usr/sbin, /usr/local/bin, /usr/local/sbin, ~/.bin, ~/bin, ~/.local/bin
func BinDirectory(filename string) bool {
	p, err := filepath.Abs(filepath.Dir(filename))
	if err != nil {
		return false
	}
	switch p {
	case "/bin", "/sbin", "/usr/bin", "/usr/sbin", "/usr/local/bin", "/usr/local/sbin":
		return true
	}
	homeDir := env.HomeDir()
	switch p {
	case filepath.Join(homeDir, ".bin"), filepath.Join(homeDir, "bin"), filepath.Join("local", "bin"):
		return true
	}
	return false
}

// DataReadyOnStdin checks if data is ready on stdin
func DataReadyOnStdin() bool {
	fileInfo, err := os.Stdin.Stat()
	return err == nil && !(fileInfo.Mode()&os.ModeNamedPipe == 0)
}

// IsBinary returns true if the given filename can be read and is a binary file
func IsBinary(filename string) bool {
	isBinary, err := binary.File(filename)
	return err == nil && isBinary
}

// FilterOutBinaryFiles filters out files that are either binary or can not be read
func FilterOutBinaryFiles(filenames []string) []string {
	var nonBinaryFilenames []string
	for _, filename := range filenames {
		if isBinary, err := binary.File(filename); !isBinary && err == nil {
			nonBinaryFilenames = append(nonBinaryFilenames, filename)
		}
	}
	return nonBinaryFilenames
}

// TimestampedFilename prefixes the given filename with a timestamp
func TimestampedFilename(filename string) string {
	now := time.Now()
	year, month, day := now.Date()
	hour, minute, second := now.Clock()
	return fmt.Sprintf("%04d-%02d-%02dT%02d-%02d-%02d-%s", year, int(month), day, hour, minute, second, filename)
}

// ShortPath replaces the home directory with ~ in a given path.
// The given path is expected to contain the home directory path either 0 or 1 times,
// and if it contains the path to the home directory, it is expected to be at the start of the given string.
func ShortPath(path string) string {
	homeDir := env.HomeDir()
	if strings.HasPrefix(path, homeDir) {
		return strings.Replace(path, homeDir, "~", 1)
	}
	return path
}

// FileHas checks if the given file exists and contains the given string
func FileHas(path, what string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return bytes.Contains(data, []byte(what))
}

// ReadString returns the contents of the given filename as a string.
// Returns an empty string if there were errors.
func ReadString(filename string) string {
	if data, err := os.ReadFile(filename); err == nil { // success
		return string(data)
	}
	return ""
}

// CanRead checks if 1 byte can actually be read from the given filename
func CanRead(filename string) bool {
	f, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer f.Close()
	var onebyte [1]byte
	n, err := io.ReadFull(f, onebyte[:])
	// could exactly 1 byte be read?
	return err == nil && n == 1
}

// Relative takes an absolute or relative path and attempts to return it relative to the current directory.
// If there are errors, it simply returns the given path.
func Relative(path string) string {
	currentDir, err := os.Getwd()
	if err != nil {
		return path
	}
	relativePath, err := filepath.Rel(currentDir, path)
	if err != nil {
		return path
	}
	return relativePath
}

// Touch behaves like "touch" on the command line, and creates a file or updates the timestamp
func Touch(filename string) error {
	// Check if the file exists
	if Exists(filename) {
		// If the file exists, update its modification time
		currentTime := time.Now()
		return os.Chtimes(filename, currentTime, currentTime)
	}

	// If the file does not exist, create it with mode 0666
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	return file.Close()
}

// ExistsCached checks if the given path exists, using a cache for faster access.
// Assumes that the filesystem has not changed since the last check.
func ExistsCached(path string) bool {
	cacheMutex.RLock()
	cachedResult, cached := existsCache[path]
	cacheMutex.RUnlock()
	if cached {
		return cachedResult
	}
	// If not cached, check if the file exists
	exists := Exists(path)
	// Cache the result
	cacheMutex.Lock()
	existsCache[path] = exists
	cacheMutex.Unlock()
	return exists
}

// ClearCache clears all cache
func ClearCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	existsCache = make(map[string]bool)
	executableCache = make(map[string]bool)
	whichCache = make(map[string]string)
}

// RemoveFile deletes a file, but it only returns an error if the file both exists and also could not be removed
func RemoveFile(path string) error {
	err := os.Remove(path)
	if !os.IsNotExist(err) {
		return err // the file both exists and could not be removed
	}
	return nil // the file has been removed, or did not exist in the first place
}

// DirectoryWithFiles checks if the given path is a directory and contains at least one file
func DirectoryWithFiles(path string) (bool, error) {
	if fileInfo, err := os.Stat(path); err != nil {
		return false, err
	} else if !fileInfo.IsDir() {
		return false, fmt.Errorf("path is not a directory")
	}
	if entries, err := os.ReadDir(path); err != nil {
		return false, err
	} else {
		for _, entry := range entries {
			if entry.Type().IsRegular() || entry.Type()&os.ModeSymlink != 0 {
				return true, nil
			}
		}
	}
	return false, nil
}

// IsExecutable checks if the given path exists and is executable
func IsExecutable(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	mode := fi.Mode()
	return mode.IsRegular() && mode&0111 != 0
}

// IsExecutableCached checks if the given path exists and is an executable file, with cache support.
// Assumes that the filesystem permissions have not changed since the last check.
func IsExecutableCached(path string) bool {
	cacheMutex.RLock()
	cachedResult, cached := executableCache[path]
	cacheMutex.RUnlock()
	if cached {
		return cachedResult
	}
	isExecutable := IsExecutable(path)
	cacheMutex.Lock()
	executableCache[path] = isExecutable
	cacheMutex.Unlock()
	return isExecutable
}
