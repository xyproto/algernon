package utils

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// Pathsep is the path separator for the current platform
	Pathsep = string(filepath.Separator)

	// EveryInstance can be used when replacing strings
	EveryInstance = -1

	// KiB is a kilobyte (kibibyte)
	KiB = 1024

	// MiB is a megabyte (mibibyte)
	MiB = 1024 * 1024
)

// URL2filename translates a given URL path to a probable full filename
func URL2filename(dirname, urlpath string) string {
	if strings.Contains(urlpath, "..") {
		log.Warn("Someone was trying to access a directory with .. in the URL")
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
		log.WithFields(log.Fields{
			"dirname": dirname,
			"error":   err.Error(),
		}).Error("Could not open directory")
		return []string{}
	}
	defer dir.Close()
	filenames, err := dir.Readdirnames(-1)
	if err != nil {
		log.WithFields(log.Fields{
			"dirname": dirname,
			"error":   err.Error(),
		}).Error("Could not read filenames from directory")

		return []string{}
	}
	return filenames
}

// Infostring builds up a string on the form "functionname(arg1, arg2, arg3)"
func Infostring(functionName string, args []string) string {
	s := functionName + "("
	if len(args) > 0 {
		s += "\"" + strings.Join(args, "\", \"") + "\""
	}
	return s + ")"
}

// OneLevelOfIndentation finds one level of whitespace, given indented data
// and a keyword to extract the whitespace in front of.
//
// Returns either an empty string or the whitespace that represents one step of
// indentation in the given source code data.
func OneLevelOfIndentation(data *[]byte, keyword string) string {
	whitespace := ""
	kwb := []byte(keyword)
	// If there is a line that contains the given word, extract the whitespace
	if bytes.Contains(*data, kwb) {
		// Find the line that contains they keyword
		var byteline []byte
		found := false
		// Try finding the line with keyword, using \n as the newline
		for _, byteline = range bytes.Split(*data, []byte("\n")) {
			if bytes.Contains(byteline, kwb) {
				found = true
				break
			}
		}
		if found {
			// Find the whitespace in front of the keyword
			whitespaceBytes := byteline[:bytes.Index(byteline, kwb)]
			// Whitespace for one level of indentation
			whitespace = string(whitespaceBytes)
		}
	}
	// Return an empty string, or whitespace for one level of indentation
	return whitespace
}

// FilterIntoGroups filters []byte slices into two groups, depending on the given filter function
func FilterIntoGroups(bytelines [][]byte, filterfunc func([]byte) bool) ([][]byte, [][]byte) {
	var special, regular [][]byte
	for _, byteline := range bytelines {
		if filterfunc(byteline) {
			// Special
			special = append(special, byteline)
		} else {
			// Regular
			regular = append(regular, byteline)
		}
	}
	return special, regular
}

/*ExtractKeywords takes a source file as `data` and a list of keywords to
 * look for. Lines without keywords are returned, together with a map
 * from keywords to []bytes from the source `data`.
 *
 * The keywords in the data must be on the form "keyword: value",
 * and can be within single-line HTML comments (<-- ... -->).
 */
func ExtractKeywords(data []byte, keywordsToLookFor []string) ([]byte, map[string][]byte) {
	bnl := []byte("\n")
	commentStart := []byte("<!--")
	commentEnd := []byte("-->")
	var keywordColon []byte
	found := make(map[string][]byte)
	// Find and separate the lines starting with one of the keywords in the special map
	_, regular := FilterIntoGroups(bytes.Split(data, bnl), func(byteline []byte) bool {
		// Check if the current line has one of the special keywords
		for _, keyword := range keywordsToLookFor {
			keywordColon = append([]byte(keyword), ':')
			// Check for lines starting with the keyword and a ":"
			if bytes.HasPrefix(byteline, keywordColon) {
				// Set (possibly overwrite) the value in the map, if the keyword is found.
				// Trim the surrounding whitespace and skip the letters of the keyword itself.
				found[keyword] = bytes.TrimSpace(byteline[len(keywordColon):])
				return true
			}
			// Check for lines that starts with "<!--", ends with "-->" and contains the keyword and a ":"
			if bytes.HasPrefix(byteline, commentStart) && bytes.HasSuffix(byteline, commentEnd) {
				// Strip away the comment markers
				stripped := bytes.TrimSpace(byteline[5 : len(byteline)-3])
				// Check if one of the relevant keywords are present
				if bytes.HasPrefix(stripped, keywordColon) {
					// Set (possibly overwrite) the value in the map, if the keyword is found.
					// Trim the surrounding whitespace and skip the letters of the keyword itself.
					found[keyword] = bytes.TrimSpace(stripped[len(keyword)+1:])
					return true
				}
			}

		}
		// Not special
		return false
	})
	// Use the regular lines as the new data (remove the special lines)
	return bytes.Join(regular, bnl), found
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

// Round a float64 to the nearest integer and return as a float64
func roundf(x float64) float64 {
	return math.Floor(0.5 + x)
}

// Round a float64 to the nearest integer
func round(x float64) int64 {
	return int64(roundf(x))
}

// WriteStatus writes a status message to a buffer, given a name and a bool
func WriteStatus(buf *bytes.Buffer, title string, flags map[string]bool) {

	// Check that at least one of the bools are true

	found := false
	for _, value := range flags {
		if value {
			found = true
			break
		}
	}
	if !found {
		return
	}

	// Write the overview to the buffer

	buf.WriteString(title + ":")
	// Spartan way of lining up the columns
	if len(title) < 7 {
		buf.WriteString("\t")
	}
	buf.WriteString("\t\t[")
	var enabledFlags []string
	// Add all enabled flags to the list
	for name, enabled := range flags {
		if enabled {
			enabledFlags = append(enabledFlags, name)
		}
	}
	buf.WriteString(strings.Join(enabledFlags, ", "))
	buf.WriteString("]\n")
}
