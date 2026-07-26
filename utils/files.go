package utils

import (
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	// Pathsep is the path separator for the current platform
	Pathsep = string(filepath.Separator)

	// KiB is a kilobyte (kibibyte)
	KiB = 1024

	// MiB is a megabyte (mibibyte)
	MiB = 1024 * 1024
)

// WithinDir reports whether p is the same as, or a descendant of, root.
// The comparison is lexical, so resolve symlinks first, if that matters.
func WithinDir(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+Pathsep)
}

// URL2filename translates a given URL path to a probable full filename.
// URL paths that lead outside of dirname result in dirname itself.
func URL2filename(dirname, urlpath string) string {
	if strings.Contains(urlpath, "..") {
		logrus.Warn("Someone was trying to access a directory with .. in the URL")
		return dirname + Pathsep
	}
	// Canonicalize first, so that the result can only be within dirname
	cleaned := CanonicalURLPath(urlpath)
	filename := filepath.Join(dirname, filepath.FromSlash(cleaned))
	if !WithinDir(dirname, filename) {
		logrus.Warn("Refusing to serve a filename outside of " + dirname)
		return dirname + Pathsep
	}
	// filepath.Join drops the trailing slash that tells a directory from a file
	if strings.HasSuffix(cleaned, "/") && !strings.HasSuffix(filename, Pathsep) {
		filename += Pathsep
	}
	return filename
}

// GetFilenames retrieves a list of filenames from a given directory name (that must exist)
func GetFilenames(dirname string) []string {
	dir, err := os.Open(dirname)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"dirname": dirname,
			"error":   err.Error(),
		}).Error("Could not open directory")
		return []string{}
	}
	defer dir.Close()
	filenames, err := dir.Readdirnames(-1)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"dirname": dirname,
			"error":   err.Error(),
		}).Error("Could not read filenames from directory")

		return []string{}
	}
	return filenames
}

// DescribeBytes converts bytes to KiB or MiB. Returns a string.
func DescribeBytes(size int64) string {
	if size < MiB {
		return strconv.Itoa(int(math.Round(float64(size)*100.0/KiB)/100)) + " KiB"
	}
	return strconv.Itoa(int(math.Round(float64(size)*100.0/MiB)/100)) + " MiB"
}
