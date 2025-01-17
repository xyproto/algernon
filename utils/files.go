package utils

import (
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

// URL2filename translates a given URL path to a probable full filename
func URL2filename(dirname, urlpath string) string {
	if strings.Contains(urlpath, "..") {
		logrus.Warn("Someone was trying to access a directory with .. in the URL")
		return dirname + Pathsep
	}
	if strings.HasPrefix(urlpath, "/") {
		if strings.HasSuffix(dirname, Pathsep) {
			return dirname + urlpath[1:]
		}
		return dirname + Pathsep + urlpath[1:]
	}
	return dirname + "/" + urlpath
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

// DurationToMS converts time.Duration to milliseconds, as a string,
// (just the number as a string, no "ms" suffix).
func DurationToMS(d time.Duration, multiplier float64) string {
	return strconv.Itoa(int(d.Seconds() * 1000.0 * multiplier))
}

// DescribeBytes converts bytes to KiB or MiB. Returns a string.
func DescribeBytes(size int64) string {
	if size < MiB {
		return strconv.Itoa(int(round(float64(size)*100.0/KiB)/100)) + " KiB"
	}
	return strconv.Itoa(int(round(float64(size)*100.0/MiB)/100)) + " MiB"
}

// Round a float64 to the nearest integer
func round(x float64) int64 {
	return int64(math.Round(x))
}
