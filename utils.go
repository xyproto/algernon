package main

import (
	"bytes"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	everyInstance = -1 // Used when replacing strings

	// KiB is a kilobyte
	KiB = 1024
	// MiB is a megabyte
	MiB = 1024 * 1024
)

// Output can be used for temporarily silencing stdout by redirecting to /dev/null or NUL
type Output struct {
	stdout  *os.File
	enabled bool
}

// Disable output to stdout
func (o *Output) disable() {
	if !o.enabled {
		o.stdout = os.Stdout
		os.Stdout, _ = os.OpenFile(os.DevNull, os.O_WRONLY, 0644)
		o.enabled = true
	}
}

// Enable output to stdout
func (o *Output) enable() {
	if o.enabled {
		os.Stdout = o.stdout
	}
}

// Translate a given URL path to a probable full filename
func url2filename(dirname, urlpath string) string {
	if strings.Contains(urlpath, "..") {
		log.Warn("Someone was trying to access a directory with .. in the URL")
		return dirname + pathsep
	}
	if strings.HasPrefix(urlpath, "/") {
		if strings.HasSuffix(dirname, pathsep) {
			return dirname + urlpath[1:]
		}
		return dirname + pathsep + urlpath[1:]
	}
	return dirname + "/" + urlpath
}

// Get a list of filenames from a given directory name (that must exist)
func getFilenames(dirname string) []string {
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

// Build up a string on the form "functionname(arg1, arg2, arg3)"
func infostring(functionName string, args []string) string {
	s := functionName + "("
	if len(args) > 0 {
		s += "\"" + strings.Join(args, "\", \"") + "\""
	}
	return s + ")"
}

// Find one level of whitespace, given indented data
// and a keyword to extract the whitespace in front of
func oneLevelOfIndentation(data *[]byte, keyword string) string {
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

// Filter []byte slices into two groups, depending on the given filter function
func filterIntoGroups(bytelines [][]byte, filterfunc func([]byte) bool) ([][]byte, [][]byte) {
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

// Given a source file, extract keywords and values into the given map.
// The map must be filled with keywords to look for.
// The keywords in the data must be on the form "keyword: value",
// and can be within single-line HTML comments (<-- ... -->).
// Returns the data for the lines that does not contain any of the keywords.
func extractKeywords(data []byte, special map[string]string) []byte {
	bnl := []byte("\n")
	// Find and separate the lines starting with one of the keywords in the special map
	_, regular := filterIntoGroups(bytes.Split(data, bnl), func(byteline []byte) bool {
		// Check if the current line has one of the special keywords
		for keyword := range special {
			// Check for lines starting with the keyword and a ":"
			if bytes.HasPrefix(byteline, []byte(keyword+":")) {
				// Set (possibly overwrite) the value in the map, if the keyword is found.
				// Trim the surrounding whitespace and skip the letters of the keyword itself.
				special[keyword] = strings.TrimSpace(string(byteline)[len(keyword)+1:])
				return true
			}
			// Check for lines that starts with "<!--", ends with "-->" and contains the keyword and a ":"
			if bytes.HasPrefix(byteline, []byte("<!--")) && bytes.HasSuffix(byteline, []byte("-->")) {
				// Strip away the comment markers
				stripped := strings.TrimSpace(string(byteline[5 : len(byteline)-3]))
				// Check if one of the relevant keywords are present
				if strings.HasPrefix(stripped, keyword+":") {
					// Set (possibly overwrite) the value in the map, if the keyword is found.
					// Trim the surrounding whitespace and skip the letters of the keyword itself.
					special[keyword] = strings.TrimSpace(stripped[len(keyword)+1:])
					return true
				}
			}

		}
		// Not special
		return false
	})
	// Use the regular lines as the new data (remove the special lines)
	return bytes.Join(regular, bnl)
}

// Fatal exit
func (ac *algernonConfig) fatalExit(err error) {
	// Log to file, if a log file is used
	if ac.serverLogFile != "" {
		log.Error(err)
	}
	// Then switch to stderr and log the message there as well
	log.SetOutput(os.Stderr)
	// Use the standard formatter
	log.SetFormatter(&log.TextFormatter{})
	// Log and exit
	log.Fatalln(err)
}

// Abrupt exit
func (ac *algernonConfig) abruptExit(msg string) {
	// Log to file, if a log file is used
	if ac.serverLogFile != "" {
		log.Info(msg)
	}
	// Then switch to stderr and log the message there as well
	log.SetOutput(os.Stderr)
	// Use the standard formatter
	log.SetFormatter(&log.TextFormatter{})
	// Log and exit
	log.Info(msg)
	os.Exit(0)
}

// Quit after a short duration
func (ac *algernonConfig) quitSoon(msg string, soon time.Duration) {
	time.Sleep(soon)
	ac.abruptExit(msg)
}

// Convert time.Duration to milliseconds, as a string (without "ms")
func durationToMS(d time.Duration, multiplier float64) string {
	return strconv.Itoa(int(d.Seconds() * 1000.0 * multiplier))
}

// Convert byte to KiB or MiB
func describeBytes(size int64) string {
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
