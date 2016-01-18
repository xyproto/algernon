package main

import (
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/yuin/gopher-lua"
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"time"

	"github.com/didip/tollbooth"
	"github.com/xyproto/mime"
	"github.com/xyproto/pinterface"
)

const (
	// Path separator
	pathsep = string(filepath.Separator)

	// Gzip content over this size
	gzipThreshold = 4096
)

var (
	// List of filenames that should be displayed instead of a directory listing
	indexFilenames = []string{"index.lua", "index.html", "index.md", "index.txt", "index.pongo2", "index.amber", "index.tmpl", "index.po2"}

	// Used for setting mime types
	mimereader *mime.Reader

	// Placed in the header when responding
	luaVersionString = lua.PackageName + "/" + lua.PackageVersion
)

// Check if the client supports gzip compressed responses
func clientCanGzip(req *http.Request) bool {
	return strings.Contains(req.Header.Get("Accept-Encoding"), "gzip")
}

// When serving a file. The file must exist. Must be given a full filename.
func filePage(w http.ResponseWriter, req *http.Request, filename string, perm pinterface.IPermissions, luapool *lStatePool, cache *FileCache) {

	// Mimetypes
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {

	// HTML pages are handled differently, if auto-refresh has been enabled
	case ".html", ".htm":
		w.Header().Add("Content-Type", "text/html; charset=utf-8")

		// Read the file (possibly in compressed format, straight from the cache)
		dataBlock, err := cache.read(filename, shouldCache(ext))
		if err != nil {
			if debugMode {
				fmt.Fprintf(w, "Can't open %s: %s", filename, err)
			} else {
				log.Errorf("Can't open %s: %s", filename, err)
			}
		}

		// If the auto-refresh feature has been enabled
		if autoRefreshMode {
			// Get the bytes from the datablock
			htmldata := dataBlock.MustData()
			// Insert JavaScript for refreshing the page, into the HTML
			htmldata = insertAutoRefresh(req, htmldata)
			// Write the data to the client
			NewDataBlock(htmldata).ToClient(w, req)
		} else {
			// Serve the file
			dataBlock.ToClient(w, req)
		}

	// Markdown pages are handled differently
	case ".md", ".markdown":

		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		markdownblock, err := cache.read(filename, shouldCache(ext))
		if err != nil {
			if debugMode {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
			} else {
				log.Errorf("Unable to read %s: %s", filename, err)
			}
			return
		}

		// Render the markdown page
		markdownPage(w, req, markdownblock.MustData(), filename, cache)

		return

	case ".amber", ".amb":

		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		amberblock, err := cache.read(filename, shouldCache(ext))
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
		luablock, err := cache.read(luafilename, shouldCache(ext))
		if err != nil {
			// Could not find and/or read data.lua
			luablock = EmptyDataBlock
		}
		// Make functions from the given Lua data available
		funcs := make(template.FuncMap)
		// luablock can be empty if there was an error or if the file was empty
		if luablock.HasData() {
			// There was Lua code available. Now make the functions and
			// variables available for the template.
			funcs, err = luaFunctionMap(w, req, luablock.MustData(), luafilename, perm, luapool, cache)
			if err != nil {
				if debugMode {
					// Use the Lua filename as the title
					prettyError(w, req, luafilename, luablock.MustData(), err.Error(), "lua")
				} else {
					log.Error(err)
				}
				return
			}
			if debugMode && verboseMode {
				s := "These functions from " + luafilename
				s += " are useable for " + filename + ": "
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
		amberPage(w, req, filename, luafilename, amberblock.MustData(), funcs, cache)

		return

	case ".tmpl", ".pongo2", ".po2":

		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		pongoblock, err := cache.read(filename, shouldCache(ext))
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
		luablock, err := cache.read(luafilename, shouldCache(ext))
		if err != nil {
			// Could not find and/or read data.lua
			luablock = EmptyDataBlock
		}
		// Make functions from the given Lua data available
		funcs := make(template.FuncMap)
		// luablock can be empty if there was an error or if the file was empty
		if luablock.HasData() {
			// There was Lua code available. Now make the functions and
			// variables available for the template.
			funcs, err = luaFunctionMap(w, req, luablock.MustData(), luafilename, perm, luapool, cache)
			if err != nil {
				if debugMode {
					// Use the Lua filename as the title
					prettyError(w, req, luafilename, luablock.MustData(), err.Error(), "lua")
				} else {
					log.Error(err)
				}
				return
			}
			if debugMode && verboseMode {
				s := "These functions from " + luafilename
				s += " are useable for " + filename + ": "
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

		// Render the Pongo2 page, using functions from data.lua, if available
		pongoPage(w, req, filename, luafilename, pongoblock.MustData(), funcs, cache)

		return

	case ".gcss":

		w.Header().Add("Content-Type", "text/css; charset=utf-8")
		gcssblock, err := cache.read(filename, shouldCache(ext))
		if err != nil {
			if debugMode {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
			} else {
				log.Errorf("Unable to read %s: %s", filename, err)
			}
			return
		}

		// Render the GCSS page as CSS
		gcssPage(w, req, filename, gcssblock.MustData())

		return

	case ".jsx":

		w.Header().Add("Content-Type", "text/javascript; charset=utf-8")
		jsxblock, err := cache.read(filename, shouldCache(ext))
		if err != nil {
			if debugMode {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
			} else {
				log.Errorf("Unable to read %s: %s", filename, err)
			}
			return
		}

		// Render the JSX page as JavaScript
		jsxPage(w, req, filename, jsxblock.MustData())

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
				fileblock, err := cache.read(filename, shouldCache(ext))
				if err != nil {
					// If the file could not be read, use the error message as the data
					// Use the error as the file contents when displaying the error message
					// if reading the file failed.
					fileblock = errorToDataBlock(err)
				}
				// If there were errors, display an error page
				prettyError(w, req, filename, fileblock.MustData(), errortext, "lua")
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

	// Read the file (possibly in compressed format, straight from the cache)
	dataBlock, err := cache.read(filename, shouldCache(ext))
	if err != nil {
		if debugMode {
			fmt.Fprintf(w, "Can't open %s: %s", filename, err)
		} else {
			log.Errorf("Can't open %s: %s", filename, err)
		}
	}

	// Serve the file
	dataBlock.ToClient(w, req)
}

// For communicating information about the underlying software
func powerHeader(w http.ResponseWriter, name, version string) {
	w.Header().Set("X-Powered-By", name+"/"+version)
}

// Server headers that are set before anything else
func serverHeaders(w http.ResponseWriter) {
	w.Header().Set("Server", serverHeaderName)
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

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

// When a file is not found
func noPage(filename string) string {
	return easyPage("Not found", "File not found: "+filename)
}

func initializeMime() {
	// Read in the mimetype information from the system. Set UTF-8 when setting Content-Type.
	mimereader = mime.New("/etc/mime.types", true)
}

// Return the domain of a request (up to ":", if any)
func getDomain(req *http.Request) string {
	for i, r := range req.Host {
		if r == ':' {
			return req.Host[:i]
		}
	}
	return req.Host
}

// Serve all files in the current directory, or only a few select filetypes (html, css, js, png and txt)
func registerHandlers(mux *http.ServeMux, handlePath, servedir string, perm pinterface.IPermissions, luapool *lStatePool, cache *FileCache, addDomain bool) {

	// Handle all requests with this function
	allRequests := func(w http.ResponseWriter, req *http.Request) {
		if perm.Rejected(w, req) {
			// Get and call the Permission Denied function
			perm.DenyFunction()(w, req)
			// Reject the request by returning
			return
		}

		// Local to this function
		servedir := servedir

		// Look for the directory that is named the same as the host
		if addDomain {
			servedir = filepath.Join(servedir, getDomain(req))
		}

		urlpath := req.URL.Path
		filename := url2filename(servedir, urlpath)
		// Remove the trailing slash from the filename, if any
		noslash := filename
		if strings.HasSuffix(filename, pathsep) {
			noslash = filename[:len(filename)-1]
		}
		hasdir := fs.exists(filename) && fs.isDir(filename)
		dirname := filename
		hasfile := fs.exists(noslash)

		// Set the server header.
		serverHeaders(w)

		// Share the directory or file
		if hasdir {
			dirPage(w, req, servedir, dirname, perm, luapool, cache)
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
		mux.HandleFunc(handlePath, allRequests)
	} else {
		limiter := tollbooth.NewLimiter(limitRequests, time.Second)
		limiter.MessageContentType = "text/html; charset=utf-8"
		limiter.Message = easyPage("Rate-limit exceeded", "<div style='color:red'>You have reached the maximum request limit.</div>")
		mux.Handle(handlePath, tollbooth.LimitFuncHandler(limiter, allRequests))
	}
}
