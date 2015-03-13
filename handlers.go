package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/russross/blackfriday"
	"github.com/xyproto/mime"
	"github.com/xyproto/permissions2"
	"github.com/yuin/gopher-lua"
)

const sep = string(os.PathSeparator)

// When serving a file. The file must exist. Must be given a full filename.
func filePage(w http.ResponseWriter, req *http.Request, filename string, perm *permissions.Permissions, mimereader *mime.MimeReader, luapool *lStatePool) {
	// Mimetypes
	ext := path.Ext(filename)
	// Markdown pages are handled differently
	if ext == ".md" {
		w.Header().Add("Content-Type", "text/html")
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
			return
		}
		markdownBody := string(blackfriday.MarkdownCommon(b))
		fmt.Fprint(w, markdownPage(filename, markdownBody))
		return
	} else if ext == ".lua" {
		runLua(w, req, filename, perm, luapool)
		return
	}
	// Set the correct Content-Type
	mimereader.SetHeader(w, ext)
	// Write to the ResponseWriter, from the File
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		fmt.Fprintf(w, "Can't open %s: %s", filename, err)
	}
	// Serve the file
	io.Copy(w, file)
	return
}

// Directory listing
func directoryListing(w http.ResponseWriter, rootdir, dirname string) {
	var buf bytes.Buffer
	for _, filename := range getFilenames(dirname) {

		// Find the full name
		full_filename := dirname
		if !strings.HasSuffix(full_filename, sep) {
			full_filename += sep
		}
		full_filename += filename

		// Remove the root directory from the link path
		urlpath := full_filename[len(rootdir)+1:]

		// Output different entries for files and directories
		buf.WriteString(easyLink(filename, urlpath, isDir(full_filename)))
	}
	title := dirname
	// Strip the leading "./"
	if strings.HasPrefix(title, "."+sep) {
		title = title[1+len(sep):]
	}
	// Use the application title for the main page
	//if title == "" {
	//	title = version_string
	//}
	if buf.Len() > 0 {
		fmt.Fprint(w, easyPage(title, buf.String()))
	} else {
		fmt.Fprint(w, easyPage(title, "Empty directory"))
	}
}

// When serving a directory. The directory must exist. Must be given a full filename.
func dirPage(w http.ResponseWriter, req *http.Request, rootdir, dirname string, perm *permissions.Permissions, mimereader *mime.MimeReader, luapool *lStatePool) {
	// Handle the serving of index files, if needed
	for _, indexfile := range indexFilenames {
		filename := path.Join(dirname, indexfile)
		if exists(filename) {
			filePage(w, req, filename, perm, mimereader, luapool)
			return
		}
	}
	// Serve a directory listing of no index file is found
	directoryListing(w, rootdir, dirname)
}

// When a file is not found
func noPage(filename string) string {
	return easyPage("Not found", "File not found: "+filename)
}

// Serve all files in the current directory, or only a few select filetypes (html, css, js, png and txt)
func registerHandlers(mux *http.ServeMux, servedir string, perm *permissions.Permissions) {
	// Read in the mimetype information from the system. Set UTF-8 when setting Content-Type.
	mimereader := mime.New("/etc/mime.types", true)
	rootdir := servedir

	// Lua LState pool
	luapool := &lStatePool{saved: make([]*lua.LState, 0, 4)}
	defer luapool.Shutdown()

	// Handle all requests with this function
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if perm.Rejected(w, req) {
			http.Error(w, "Permission denied!", http.StatusForbidden)
			return
		}
		urlpath := req.URL.Path
		filename := url2filename(servedir, urlpath)
		// Remove the trailing slash from the filename, if any
		noslash := filename
		if strings.HasSuffix(filename, sep) {
			noslash = filename[:len(filename)-1]
		}
		hasdir := exists(filename) && isDir(filename)
		hasfile := exists(noslash)
		// Share the directory or file
		if hasdir {
			dirPage(w, req, rootdir, filename, perm, mimereader, luapool)
			return
		} else if !hasdir && hasfile {
			// Share a single file instead of a directory
			filePage(w, req, noslash, perm, mimereader, luapool)
			return
		}
		// Not found
		fmt.Fprint(w, noPage(filename))
	})
}
