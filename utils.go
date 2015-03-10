package main

import (
	"log"
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
	sep := string(os.PathSeparator)
	if strings.Contains(urlpath, "..") {
		log.Println("Trying to access URL with ..")
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
	if err != nil {
		log.Fatalf("Could not open directory: %s (%s)", dirname, err)
		return []string{}
	}
	filenames, err := dir.Readdirnames(-1)
	if err != nil {
		log.Fatalf("Could not read filenames from directory: %s (%s)", dirname, err)
		return []string{}
	}
	return filenames
}

func easyPage(title, body string) string {
	return "<!doctype html><html><head>" + font + "<title>" + title + "</title><style>" + style + "</style><head><body><h1>" + title + "</h1>" + body + "</body></html>"
}

// TODO: Check if handling "# title <tags" on the first line is valid Markdown or not. Submit a patch to blackfriday if it is.
func markdownPage(title, htmlbody string) string {
	h1title := ""
	if strings.HasPrefix(htmlbody, "<p>#") {
		fields := strings.Split(htmlbody, "<")
		if len(fields) > 2 {
			h1title = fields[1][2:]
			htmlbody = htmlbody[len("<p>"+h1title):]
			if strings.HasPrefix(h1title, "#") {
				h1title = h1title[1:]
			}
		}
	}
	return "<!doctype html><html><head><title>" + title + "</title><style>" + style + "</style><head><body><h1>" + h1title + "</h1>" + htmlbody + "</body></html>"
}

func easyLink(text, url string, isDirectory bool) string {
	if isDirectory {
		text += "/"
	}
	return "<a href=\"/" + url + "\">" + text + "</a><br>"
}
