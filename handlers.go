package main

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"

	"github.com/xyproto/mime"
	"github.com/xyproto/permissions2"
)

const (
	pathsep = string(os.PathSeparator)
)

var (
	mimereader *mime.Reader
)

// When serving a file. The file must exist. Must be given a full filename.
func filePage(w http.ResponseWriter, req *http.Request, filename string, perm *permissions.Permissions, luapool *lStatePool) {

	// Mimetypes
	ext := path.Ext(filename)

	// Markdown pages are handled differently
	if ext == ".md" {

		w.Header().Add("Content-Type", "text/html")
		b, err := read(filename)
		if err != nil {
			if debugMode {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
			} else {
				log.Errorf("Unable to read %s: %s", filename, err)
			}
			return
		}

		// Render the markdown page
		markdownPage(w, b, filename)

		return

	} else if ext == ".amber" {

		w.Header().Add("Content-Type", "text/html")
		amberdata, err := read(filename)
		if err != nil {
			if debugMode {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
			} else {
				log.Errorf("Unable to read %s: %s", filename, err)
			}
			return
		}
		// Try reading data.lua as well, if possible
		luafilename := path.Join(path.Dir(filename), "data.lua")
		luadata, err := read(luafilename)
		if err != nil {
			// Could not find and/or read data.lua
			luadata = []byte{}
		}
		// Make functions from the given Lua data available
		funcs := make(template.FuncMap)
		if len(luadata) > 0 {
			// There was Lua code available. Now make the functions and
			// variables available for the template.
			funcs, err = luaFunctionMap(w, req, luadata, luafilename, perm, luapool)
			if err != nil {
				if debugMode {
					// Use the Lua filename as the title
					prettyError(w, luafilename, luadata, err.Error(), "lua")
				} else {
					log.Error(err)
				}
				return
			}
			if debugMode && verboseMode {
				s := "The following functions from " + luafilename + "\n"
				s += "are made available for use in " + filename + ":\n\t"
				// Create a comma separated list of the available functions
				for key := range funcs {
					s += key + ", "
				}
				// Remove the final comma
				if strings.HasSuffix(s, ", ") {
					s = s[:len(s)-2]
				}
				// Output the message
				log.Info(s)
			}
		}

		// Render the Amber page, using functions from data.lua, if available
		amberPage(w, filename, luafilename, amberdata, funcs)

		return

	} else if ext == ".gcss" {

		w.Header().Add("Content-Type", "text/css")
		gcssdata, err := read(filename)
		if err != nil {
			if debugMode {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
			} else {
				log.Errorf("Unable to read %s: %s", filename, err)
			}
			return
		}

		// Render the GCSS page as CSS
		gcssPage(w, filename, gcssdata)

		return

	} else if ext == ".jsx" {

		w.Header().Add("Content-Type", "text/javascript")
		jsxdata, err := read(filename)
		if err != nil {
			if debugMode {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
			} else {
				log.Errorf("Unable to read %s: %s", filename, err)
			}
			return
		}

		// Render the JSX page as JavaScript
		jsxPage(w, filename, jsxdata)

		return

	} else if ext == ".lua" {

		// If in debug mode, let the Lua script print to a buffer first, in
		// case there are errors that should be displayed instead.

		// If debug mode is enabled and the file can be buffered
		if debugMode && (path.Base(filename) != "stream.lua") {
			// Use a buffered ResponseWriter for delaying the output
			recorder := httptest.NewRecorder()
			// Run the lua script
			if err := runLua(recorder, req, filename, perm, luapool, false); err != nil {
				errortext := err.Error()
				filedata, err := read(filename)
				if err != nil {
					// Use the error as the file contents when displaying the error message
					// if reading the file failed.
					filedata = []byte(err.Error())
				}
				// If there were errors, display an error page
				prettyError(w, filename, filedata, errortext, "lua")
			} else {
				// If things went well, write to the ResponseWriter
				writeRecorder(w, recorder)
			}
		} else {

			// Run the lua script
			if err := runLua(w, req, filename, perm, luapool, true); err != nil {
				// Output the non-fatal error message to the log
				log.Error("Error in ", filename+":", err)
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
		if debugMode {
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
		buf.WriteString(easyLink(filename, urlpath, isDir(fullFilename)))
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
	if buf.Len() > 0 {
		fmt.Fprint(w, easyPage(title, buf.String()))
	} else {
		fmt.Fprint(w, easyPage(title, "Empty directory"))
	}
}

// When serving a directory. The directory must exist. Must be given a full filename.
func dirPage(w http.ResponseWriter, req *http.Request, rootdir, dirname string, perm *permissions.Permissions, luapool *lStatePool) {
	// Handle the serving of index files, if needed
	for _, indexfile := range indexFilenames {
		filename := path.Join(dirname, indexfile)
		if exists(filename) {
			filePage(w, req, filename, perm, luapool)
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
	mimereader = mime.New("/etc/mime.types", true)
	rootdir := servedir

	// Handle all requests with this function
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if perm.Rejected(w, req) {
			// Get and call the Permission Denied function
			perm.DenyFunction()(w, req)
			// Reject the request by returning
			return
		}

		// TODO: HTTP Basic Auth check goes here, see "scoreserver"

		urlpath := req.URL.Path
		filename := url2filename(servedir, urlpath)
		// Remove the trailing slash from the filename, if any
		noslash := filename
		if strings.HasSuffix(filename, pathsep) {
			noslash = filename[:len(filename)-1]
		}
		hasdir := exists(filename) && isDir(filename)
		hasfile := exists(noslash)

		// TODO: Only set the server header if configured to do so

		// Set the server header.
		w.Header().Set("Server", "Algernon")

		// Share the directory or file
		if hasdir {
			dirPage(w, req, rootdir, filename, perm, luapool)
			return
		} else if !hasdir && hasfile {
			// Share a single file instead of a directory
			filePage(w, req, noslash, perm, luapool)
			return
		}

		// Not found
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, noPage(filename))
	})
}
