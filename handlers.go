package main

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"

	"github.com/xyproto/mime"
	"github.com/xyproto/permissions2"
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
			if DEBUG_MODE {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
			} else {
				log.Errorf("Unable to read %s: %s", filename, err)
			}
			return
		}
		markdownPage(w, b, filename)
		return
	} else if ext == ".amber" {
		w.Header().Add("Content-Type", "text/html")
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			if DEBUG_MODE {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
			} else {
				log.Errorf("Unable to read %s: %s", filename, err)
			}
			return
		}
		amberPage(w, b, filename)
		return
	} else if ext == ".gcss" {
		w.Header().Add("Content-Type", "text/css")
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			if DEBUG_MODE {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
			} else {
				log.Errorf("Unable to read %s: %s", filename, err)
			}
			return
		}
		gcssPage(w, b, filename)
		return
	} else if ext == ".lua" {
		// If in debug mode, let the Lua script print to a buffer first, in
		// case there are errors that should be displayed instead.
		if DEBUG_MODE {
			// Use a buffered ResponseWriter for delaying the output
			recorder := httptest.NewRecorder()
			// Run the lua script
			if err := runLua(recorder, req, filename, perm, luapool); err != nil {
				errortext := err.Error()
				// If there were errors, display an error page
				prettyError(w, filename, errortext, "lua")
			} else {
				// If things went well, write to the ResponseWriter
				writeRecorder(w, recorder)
			}
		} else {
			// Run the lua script
			if err := runLua(w, req, filename, perm, luapool); err != nil {
				// Output the non-fatal error message to the log
				log.Error("Error in ", filename + ":", err)
			}
		}
		return
	}
	// Set the correct Content-Type
	mimereader.SetHeader(w, ext)
	// Write to the ResponseWriter, from the File
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		if DEBUG_MODE {
			fmt.Fprintf(w, "Can't open %s: %s", filename, err)
		} else {
			log.Errorf("Can't open %s: %s", filename, err)
		}
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
func registerHandlers(mux *http.ServeMux, servedir string, perm *permissions.Permissions, luapool *lStatePool) {
	// Read in the mimetype information from the system. Set UTF-8 when setting Content-Type.
	mimereader := mime.New("/etc/mime.types", true)
	rootdir := servedir

	// Handle all requests with this function
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if perm.Rejected(w, req) {
			// Get and call the Permission Denied function
			perm.DenyFunction()(w, req)
			// Reject the request by returning
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
