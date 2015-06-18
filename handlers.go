package main

import (
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/didip/tollbooth"
	"github.com/xyproto/mime"
	"github.com/xyproto/pinterface"
)

const pathsep = string(os.PathSeparator)

var (
	// List of filenames that should be displayed instead of a directory listing
	indexFilenames = []string{"index.lua", "index.html", "index.md", "index.txt", "index.amber"}

	mimereader *mime.Reader
)

// When serving a file. The file must exist. Must be given a full filename.
func filePage(w http.ResponseWriter, req *http.Request, filename string, perm pinterface.IPermissions, luapool *lStatePool, cache *FileCache) {

	// Mimetypes
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {

	// Markdown pages are handled differently
	case ".md":

		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		var markdowndata []byte
		var err error
		markdowndata, err = cache.read(filename, shouldCache(ext))
		if err != nil {
			if debugMode {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
			} else {
				log.Errorf("Unable to read %s: %s", filename, err)
			}
			return
		}

		// Render the markdown page
		markdownPage(w, markdowndata, filename, cache)

		return

	case ".amber":

		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		amberdata, err := cache.read(filename, shouldCache(ext))
		if err != nil {
			if debugMode {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
			} else {
				log.Errorf("Unable to read %s: %s", filename, err)
			}
			return
		}
		// Try reading data.lua as well, if possible
		luafilename := filepath.Join(filepath.Dir(filename), "data.lua")
		luadata, err := cache.read(luafilename, shouldCache(ext))
		if err != nil {
			// Could not find and/or read data.lua
			luadata = []byte{}
		}
		// Make functions from the given Lua data available
		funcs := make(template.FuncMap)
		if len(luadata) > 0 {
			// There was Lua code available. Now make the functions and
			// variables available for the template.
			funcs, err = luaFunctionMap(w, req, luadata, luafilename, perm, luapool, cache)
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
				s := "These functions from " + luafilename
				s += " areselable for " + filename + ": "
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
		amberPage(w, filename, luafilename, amberdata, funcs, cache)

		return

	case ".gcss":

		w.Header().Add("Content-Type", "text/css; charset=utf-8")
		gcssdata, err := cache.read(filename, shouldCache(ext))
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

	case ".jsx":

		w.Header().Add("Content-Type", "text/javascript; charset=utf-8")
		jsxdata, err := cache.read(filename, shouldCache(ext))
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

	case ".lua":

		// If in debug mode, let the Lua script print to a buffer first, in
		// case there are errors that should be displayed instead.

		// If debug mode is enabled
		if debugMode {
			// Use a buffered ResponseWriter for delaying the output
			recorder := httptest.NewRecorder()
			// The flush function writes the ResponseRecorder to the ResponseWriter
			flushFunc := func() {
				writeRecorder(w, recorder)
				Flush(w)
			}
			// Run the lua script, without the possibility to flush
			if err := runLua(recorder, req, filename, perm, luapool, flushFunc, cache); err != nil {
				errortext := err.Error()
				filedata, err := cache.read(filename, shouldCache(ext))
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
			// The flush function just flushes the ResponseWriter
			flushFunc := func() {
				Flush(w)
			}
			// Run the lua script, with the flush feature
			if err := runLua(w, req, filename, perm, luapool, flushFunc, cache); err != nil {
				// Output the non-fatal error message to the log
				log.Error("Error in ", filename+":", err)
			}
		}

		return
	}

	// Set the correct Content-Type
	if mimereader != nil {
		mimereader.SetHeader(w, ext)
	} else {
		log.Error("Uninitialized mimereader!")
	}
	// Read the file
	fileData, err := cache.read(filename, shouldCache(ext))
	if err != nil {
		if debugMode {
			fmt.Fprintf(w, "Can't open %s: %s", filename, err)
		} else {
			log.Errorf("Can't open %s: %s", filename, err)
		}
	}
	// Serve the file
	w.Write(fileData)
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

	var htmldata []byte
	if buf.Len() > 0 {
		htmldata = []byte(easyPage(title, buf.String()))
	} else {
		htmldata = []byte(easyPage(title, "Empty directory"))
	}

	// If the auto-refresh feature has been enabled
	if autoRefreshMode {
		// Insert JavaScript for refreshing the page into the generated HTML
		htmldata = insertAutoRefresh(htmldata)
	}

	w.Write(htmldata)
}

// When serving a directory.
// The directory must exist. Must be given a full filename.
func dirPage(w http.ResponseWriter, req *http.Request, rootdir, dirname string, perm pinterface.IPermissions, luapool *lStatePool, cache *FileCache) {

	// If the URL does not end with a slash, redirect to an URL that does
	if !strings.HasSuffix(req.URL.Path, "/") {
		http.Redirect(w, req, req.URL.Path+"/", http.StatusMovedPermanently)
		return
	}
	// Handle the serving of index files, if needed
	for _, indexfile := range indexFilenames {
		filename := filepath.Join(dirname, indexfile)
		if exists(filename) {
			filePage(w, req, filename, perm, luapool, cache)
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

func initializeMime() {
	// Read in the mimetype information from the system. Set UTF-8 when setting Content-Type.
	mimereader = mime.New("/etc/mime.types", true)
}

// Serve all files in the current directory, or only a few select filetypes (html, css, js, png and txt)
func registerHandlers(mux *http.ServeMux, servedir string, perm pinterface.IPermissions, luapool *lStatePool, cache *FileCache) {
	rootdir := servedir

	// Handle all requests with this function
	allRequests := func(w http.ResponseWriter, req *http.Request) {
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
		if strings.HasSuffix(filename, pathsep) {
			noslash = filename[:len(filename)-1]
		}
		hasdir := exists(filename) && isDir(filename)
		hasfile := exists(noslash)

		// Set the server header.
		w.Header().Set("Server", versionString)

		// Share the directory or file
		if hasdir {
			dirPage(w, req, rootdir, filename, perm, luapool, cache)
			return
		} else if !hasdir && hasfile {
			// Share a single file instead of a directory
			filePage(w, req, noslash, perm, luapool, cache)
			return
		}
		// Not found
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, noPage(filename))
	}

	// Handle requests differently depending on if rate limiting is enabled or not
	if disableRateLimiting {
		mux.HandleFunc("/", allRequests)
	} else {
		limiter := tollbooth.NewLimiter(limitRequests, time.Second)
		limiter.MessageContentType = "text/html; charset=utf-8"
		limiter.Message = easyPage("Rate-limit exceeded", "<div style='color:red'>You have reached the maximum request limit.</div>")
		mux.Handle("/", tollbooth.LimitFuncHandler(limiter, allRequests))
	}
}
