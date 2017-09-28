package engine

// Directory Index

import (
	"bytes"
	"net/http"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/utils"
)

var (
	// List of filenames that should be displayed instead of a directory listing
	indexFilenames = []string{"index.lua", "index.html", "index.md", "index.txt", "index.pongo2", "index.amber", "index.tmpl", "index.po2", "index.happ", "index.hyper", "index.hyper.js", "index.hyper.jsx"}

	doubleP  = utils.Pathsep + utils.Pathsep /* // */
	dotSlash = "." + utils.Pathsep           /* ./ */
)

// DirectoryListing serves the given directory as a web page with links the the contents
func (ac *Config) DirectoryListing(w http.ResponseWriter, req *http.Request, rootdir, dirname, theme string) {
	var (
		buf          bytes.Buffer
		fullFilename string
		URLpath      string
		title        = dirname
	)

	// Fill the coming HTML body with a list of all the filenames in `dirname`
	for _, filename := range utils.GetFilenames(dirname) {

		// Find the full name
		fullFilename = dirname

		// Add a "/" after the directory name, if missing
		if !strings.HasSuffix(fullFilename, utils.Pathsep) {
			fullFilename += utils.Pathsep
		}

		// Add the filename at the end
		fullFilename += filename

		// Remove the root directory from the link path
		URLpath = fullFilename[len(rootdir)+1:]

		// Output different entries for files and directories
		buf.WriteString(utils.HTMLLink(filename, URLpath, ac.fs.IsDir(fullFilename)))
	}

	// Strip the leading "./"
	title = strings.TrimPrefix(title, dotSlash)

	// Strip double "/" at the end, just keep one
	if strings.Contains(title, doubleP) {
		// Replace "//" with just "/"
		title = strings.Replace(title, doubleP, utils.Pathsep, utils.EveryInstance)
	}

	// Check if the current page contents are empty
	if buf.Len() == 0 {
		buf.WriteString("Empty directory")
	}

	htmldata := utils.MessagePageBytes(title, buf.Bytes(), theme)

	// If the auto-refresh feature has been enabled
	if ac.autoRefreshMode {
		// Insert JavaScript for refreshing the page into the generated HTML
		htmldata = ac.InsertAutoRefresh(req, htmldata)
	}

	// Serve the page
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	DataToClient(w, req, dirname, htmldata)
}

// DirPage serves a directory, using index.* files, if present.
// The directory must exist.
// rootdir is the base directory (can be ".")
// dirname is the specific directory that is to be served (should never be ".")
func (ac *Config) DirPage(w http.ResponseWriter, req *http.Request, rootdir, dirname, theme string) {

	// Check if we are instructed to quit after serving the first file
	if ac.quitAfterFirstRequest {
		go ac.quitSoon("Quit after first request", defaultSoonDuration)
	}

	// If the URL does not end with a slash, redirect to an URL that does
	if !strings.HasSuffix(req.URL.Path, "/") {
		if req.Method == "POST" {
			log.Warn("Redirecting a POST request: " + req.URL.Path + " -> " + req.URL.Path + "/.")
			log.Warn("Header data may be lost! Please add the missing slash.")
		}
		http.Redirect(w, req, req.URL.Path+"/", http.StatusMovedPermanently)
		return
	}

	// Handle the serving of index files, if needed
	var filename string
	for _, indexfile := range indexFilenames {
		filename = filepath.Join(dirname, indexfile)
		if ac.fs.Exists(filename) {
			ac.FilePage(w, req, filename, ac.defaultLuaDataFilename)
			return
		}
	}

	// Serve a directory listing of no index file is found
	ac.DirectoryListing(w, req, rootdir, dirname, theme)
}
