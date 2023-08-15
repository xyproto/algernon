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

// IsDir checks if the given path exists and is a directory
func IsDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsDir()
}

// IsFileOrSymlink checks if the given path exists and is a regular file or a symbolic link
func IsFileOrSymlink(path string) bool {
	// use Lstat instead of Stat to avoid following the symlink
	fi, err := os.Lstat(path)
	return err == nil && (fi.Mode().IsRegular() || (fi.Mode()&os.ModeSymlink != 0))
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
	if err != nil {
		return false
	}
	return !(fileInfo.Mode()&os.ModeNamedPipe == 0)
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

// ShortPath replaces the home directory with ~ in a given path
func ShortPath(path string) string {
	homeDir, _ := os.UserHomeDir()
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
// Does not use the cache.  Returns an empty string if there were errors.
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
