package main

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"os"
	"strings"
)

const (
	everyInstance = -1 // Used when replacing strings
)

var (
	// A selection of allowed keywords for the HTML meta tag
	metaKeywords = []string{"application-name", "author", "description", "generator", "keywords", "robots", "language", "googlebot", "Slurp", "bingbot", "geo.position", "geo.placename", "geo.region", "ICBM", "viewport"}
)

// Check if a given filename is a directory
func isDir(filename string) bool {
	fs, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return fs.IsDir()
}

// Check if the given filename exists
func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
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
	defer dir.Close()
	if err != nil {
		log.WithFields(log.Fields{
			"dirname": dirname,
			"error":   err.Error(),
		}).Error("Could not open directory")
		return []string{}
	}
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

// Easy way to output a HTML page
func easyPage(title, body string) string {
	return fmt.Sprintf("<!doctype html><html><head><title>%s</title>%s<style>%s</style><head><body><h1>%s</h1>%s</body></html>", title, defaultFont, defaultStyle, title, body)
}

// Easy way to build links to directories
func easyLink(text, url string, isDirectory bool) string {
	// Add a final slash, if needed
	if isDirectory {
		text += "/"
		url += "/"
	}
	return "<a href=\"/" + url + "\">" + text + "</a><br>"
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

// Add a link to a stylesheet in the given Amber code
func linkToStyle(amberdata *[]byte, url string) {
	// If the given url is not already mentioned and the data contains "body"
	if !bytes.Contains(*amberdata, []byte(url)) && bytes.Contains(*amberdata, []byte("html")) && bytes.Contains(*amberdata, []byte("body")) {
		// Extract one level of indendation
		whitespace := oneLevelOfIndentation(amberdata, "body")
		// Check if there already is a head section
		if bytes.Contains(*amberdata, []byte("head")) {
			// Add a link to the stylesheet
			*amberdata = bytes.Replace(*amberdata, []byte("head\n"), []byte("head\n"+whitespace+whitespace+`link[href="`+url+`"][rel="stylesheet"][type="text/css"]`+"\n"), 1)

		} else if bytes.Contains(*amberdata, []byte("body")) {

			// Add a link to the stylesheet
			*amberdata = bytes.Replace(*amberdata, []byte("html\n"), []byte("html\n"+whitespace+"head\n"+whitespace+whitespace+`link[href="`+url+`"][rel="stylesheet"][type="text/css"]`+"\n"), 1)
		}
	}
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

// Add meta tag names to the given map
func addMetaKeywords(keywords map[string]string) {
	for _, keyword := range metaKeywords {
		keywords[keyword] = ""
	}
}
