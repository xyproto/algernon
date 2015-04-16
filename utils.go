package main

import (
	"bytes"
	log "github.com/Sirupsen/logrus"
	"os"
	"strings"
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
		return dirname + sep
	}
	if strings.HasPrefix(urlpath, "/") {
		return dirname + sep + urlpath[1:]
	}
	return dirname + sep + urlpath
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
	return "<!doctype html><html><head>" + font + "<title>" + title + "</title><style>" + style + "</style><head><body><h1>" + title + "</h1>" + body + "</body></html>"
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

// Add a link to a stylesheet in the given Amber code
// TODO: Replace with a regex
func linkToStyle(amberdata *[]byte, url string) {
	// If the given url is not already mentioned
	if !bytes.Contains(*amberdata, []byte(url)) {
		// If there is a head section
		if bytes.Contains(*amberdata, []byte("head")) {
			// Find the line that contains head
			var byteline []byte
			found := false
			foundsep := ""
			// Try finding the line with "head", using \n as the separator
			for _, byteline = range bytes.Split(*amberdata, []byte("\n")) {
				if bytes.Contains(byteline, []byte("head")) {
					found = true
					foundsep = "\n"
					break
				}
			}
			if !found {
				// Try finding the line with "head", using sep as the separator
				for _, byteline = range bytes.Split(*amberdata, []byte(sep)) {
					if bytes.Contains(byteline, []byte("head")) {
						found = true
						foundsep = sep
						break
					}
				}
			}
			fields := bytes.Split(byteline, []byte("head"))
			spaces := ""
			if len(fields) > 0 {
				spaces = string(fields[0])
			}
			if found {
				// Add the link to the stylesheet
				*amberdata = bytes.Replace(*amberdata, []byte("head"+foundsep), []byte("head"+foundsep+spaces+"\t"+`link[href="`+url+`"][rel="stylesheet"]`), 1)
			}
		}
	}
}
