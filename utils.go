package main

import (
	"bytes"
	log "github.com/Sirupsen/logrus"
	"os"
	"strings"
)

const (
	ALL = -1 // Used when replacing strings
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
		} else {
			return dirname + pathsep + urlpath[1:]
		}
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
	return "<!doctype html><html><head>" + font + "<title>" + title + "</title><style>" + defaultStyle + "</style><head><body><h1>" + title + "</h1>" + body + "</body></html>"
}

// Easy way to build links to directories
func easyLink(text, url string, isDirectory bool) string {
	if isDirectory {
		text += "/"
	}
	// NOTE: If the directory only contains one index.* file, adding "/" to
	// the URL is not needed, because no other files will be needed to be
	// accessed from that directory by the index file in question.
	return "<a href=\"/" + url + "/\">" + text + "</a><br>"
}

// Build up a string on the form "functionname(arg1, arg2, arg3)"
func infostring(functionName string, args []string) string {
	s := functionName + "("
	if len(args) > 0 {
		s += "\"" + strings.Join(args, "\", \"") + "\""
	}
	return s + ")"
}

// Add a link to a stylesheet in the given Amber code
// TODO: A bit ugly. Rewrite the function. Replace with a regex?
func linkToStyle(amberdata *[]byte, url string) {
	// If the given url is not already mentioned
	if !bytes.Contains(*amberdata, []byte(url)) {
		// If there is a head section
		if bytes.Contains(*amberdata, []byte("head")) {
			// Find the line that contains head
			var byteline []byte
			found := false
			// Try finding the line with "head", using \n as the newline
			for _, byteline = range bytes.Split(*amberdata, []byte("\n")) {
				if bytes.Contains(byteline, []byte("head")) {
					found = true
					break
				}
			}
			// Find the whitespace in front of "head"
			whitespaceBytes := byteline[:bytes.Index(byteline, []byte("head"))]
			// Whitespace for two levels of indentation
			whitespace := string(whitespaceBytes) + string(whitespaceBytes)
			if found {
				// Add the link to the stylesheet
				*amberdata = bytes.Replace(*amberdata, []byte("head\n"), []byte("head\n"+whitespace+`link[href="`+url+`"][rel="stylesheet"][type="text/css"]`), 1)
			}
		} else if bytes.Contains(*amberdata, []byte("html")) && bytes.Contains(*amberdata, []byte("body")) {

			// No head section, but a body and html section. Add both "head" and "link"

			// Find the line that contains html
			var byteline []byte
			found := false
			// Try finding the line with "body", using \n as the newline
			for _, byteline = range bytes.Split(*amberdata, []byte("\n")) {
				if bytes.Contains(byteline, []byte("body")) {
					found = true
					break
				}
			}
			// Find the whitespace in front of "body"
			whitespaceBytes := byteline[:bytes.Index(byteline, []byte("body"))]
			// Whitespace for one level of indentation
			whitespace := string(whitespaceBytes) + string(whitespaceBytes)
			if found {
				// Add the link to the stylesheet
				*amberdata = bytes.Replace(*amberdata, []byte("html\n"), []byte("html\n"+whitespace+"head\n"+whitespace+whitespace+`link[href="`+url+`"][rel="stylesheet"][type="text/css"]`), 1)
			}

		}
	}
}
