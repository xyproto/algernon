package main

// Directory Index

import (
	"bytes"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/xyproto/pinterface"
)

// Directory listing
func directoryListing(w http.ResponseWriter, req *http.Request, rootdir, dirname string) {
	var buf bytes.Buffer
	for _, filename := range getFilenames(dirname) {

		// Find the full name
		fullFilename := dirname

		// Add a "/" after the directory name, if missing
		if !strings.HasSuffix(fullFilename, pathsep) {
			fullFilename += pathsep
		}

		// Add the filename at the end
		fullFilename += filename

		// Remove the root directory from the link path
		urlpath := fullFilename[len(rootdir)+1:]

		// Output different entries for files and directories
		buf.WriteString(easyLink(filename, urlpath, fs.isDir(fullFilename)))
	}
	title := dirname
	// Strip the leading "./"
	if strings.HasPrefix(title, "."+pathsep) {
		title = title[1+len(pathsep):]
	}
	// Strip double "/" at the end, just keep one
	// Replace "//" with just "/"
	if strings.Contains(title, pathsep+pathsep) {
		title = strings.Replace(title, pathsep+pathsep, pathsep, everyInstance)
	}

	// Use the application title for the main page
	//if title == "" {
	//	title = versionString
	//}

	var htmldata []byte
	if buf.Len() > 0 {
		htmldata = []byte(easyPage(title, buf.String()))
	} else {
		htmldata = []byte(easyPage(title, "Empty directory"))
	}

	// If the auto-refresh feature has been enabled
	if autoRefreshMode {
		// Insert JavaScript for refreshing the page into the generated HTML
		htmldata = insertAutoRefresh(req, htmldata)
	}

	// Serve the page
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	NewDataBlock(htmldata).ToClient(w, req)
}

// Serve a directory. The directory must exist.
// rootdir is the base directory (can be ".")
// dirname is the specific directory that is to be served (should never be ".")
func dirPage(w http.ResponseWriter, req *http.Request, rootdir, dirname string, perm pinterface.IPermissions, luapool *lStatePool, cache *FileCache) {

	// If the URL does not end with a slash, redirect to an URL that does
	if !strings.HasSuffix(req.URL.Path, "/") {
		http.Redirect(w, req, req.URL.Path+"/", http.StatusMovedPermanently)
		return
	}
	// Handle the serving of index files, if needed
	for _, indexfile := range indexFilenames {
		filename := filepath.Join(dirname, indexfile)
		if fs.exists(filename) {
			filePage(w, req, filename, perm, luapool, cache)
			return
		}
	}
	// Serve a directory listing of no index file is found
	directoryListing(w, req, rootdir, dirname)
}
