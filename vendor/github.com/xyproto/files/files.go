package files

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xyproto/binary"
	"github.com/xyproto/env/v2"
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
