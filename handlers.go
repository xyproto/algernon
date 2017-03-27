package main

import (
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/didip/tollbooth"
	"github.com/xyproto/datablock"
	"github.com/xyproto/mime"
	"github.com/xyproto/pinterface"
)

const (
	// Path separator
	pathsep = string(filepath.Separator)

	// Gzip content over this size
	gzipThreshold = 4096

	// Pretty soon
	defaultSoonDuration = time.Second * 3
)

var (
	// List of filenames that should be displayed instead of a directory listing
	indexFilenames = []string{"index.lua", "index.html", "index.md", "index.txt", "index.pongo2", "index.amber", "index.tmpl", "index.po2"}

	// Used for setting mime types
	mimereader *mime.Reader

	// Placed in the header when responding
	// luaVersionString = lua.PackageName + "/" + lua.PackageVersion
)

// Check if the client supports gzip compressed responses
func clientCanGzip(req *http.Request) bool {
	return strings.Contains(req.Header.Get("Accept-Encoding"), "gzip")
}

func pongoHandler(w http.ResponseWriter, req *http.Request, filename string, ext string, luaDataFilename string, perm pinterface.IPermissions, luapool *lStatePool, cache *datablock.FileCache, pongomutex *sync.RWMutex) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	pongoblock, err := cache.Read(filename, shouldCache(ext))
	if err != nil {
		if debugMode {
			fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
		} else {
			log.Errorf("Unable to read %s: %s", filename, err)
		}
		return
	}

	// Make the functions in luaDataFilename available for the Pongo2 template

	luafilename := filepath.Join(filepath.Dir(filename), luaDataFilename)
	if fs.Exists(luaDataFilename) {
		luafilename = luaDataFilename
	}
	if fs.Exists(luafilename) {
		// Extract the function map from luaDataFilenname in a goroutine
		errChan := make(chan error)
		funcMapChan := make(chan template.FuncMap)

		go lua2funcMap(w, req, filename, luafilename, ext, perm, luapool, cache, pongomutex, errChan, funcMapChan)
		funcs := <-funcMapChan
		err = <-errChan

		if err != nil {
			if debugMode {
				// Try reading luaDataFilename as well, if possible
				luablock, luablockErr := cache.Read(luafilename, shouldCache(ext))
				if luablockErr != nil {
					// Could not find and/or read luaDataFilename
					luablock = datablock.EmptyDataBlock
				}
				// Use the Lua filename as the title
				prettyError(w, req, luafilename, luablock.MustData(), err.Error(), "lua")
			} else {
				log.Error(err)
			}
			return
		}

		// Render the Pongo2 page, using functions from luaDataFilename, if available
		pongomutex.Lock()
		pongoPage(w, req, filename, pongoblock.MustData(), funcs, cache)
		pongomutex.Unlock()

		return
	}

	// Output a warning if something different from default has been given
	if !strings.HasSuffix(luafilename, defaultLuaDataFilename) {
		log.Warn("Could not read ", luafilename)
	}

	// Use the Pongo2 template without any Lua functions
	pongomutex.Lock()
	funcs := make(template.FuncMap)
	pongoPage(w, req, filename, pongoblock.MustData(), funcs, cache)
	pongomutex.Unlock()

	return
}

// When serving a file. The file must exist. Must be given a full filename.
func filePage(w http.ResponseWriter, req *http.Request, filename, luaDataFilename string, perm pinterface.IPermissions, luapool *lStatePool, cache *datablock.FileCache, pongomutex *sync.RWMutex) {

	if quitAfterFirstRequest {
		go quitSoon("Quit after first request", defaultSoonDuration)
	}

	// Mimetypes
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {

	// HTML pages are handled differently, if auto-refresh has been enabled
	case ".html", ".htm":

		w.Header().Add("Content-Type", "text/html; charset=utf-8")

		// Read the file (possibly in compressed format, straight from the cache)
		dataBlock, err := cache.Read(filename, shouldCache(ext))
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
			dataToClient(w, req, filename, htmldata)
		} else {
			// Serve the file
			dataBlock.ToClient(w, req, filename, clientCanGzip(req), gzipThreshold)
		}

		return

	// Markdown pages are handled differently
	case ".md", ".markdown":

		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		markdownblock, err := cache.Read(filename, shouldCache(ext))
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
		amberblock, err := cache.Read(filename, shouldCache(ext))
		if err != nil {
			if debugMode {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
			} else {
				log.Errorf("Unable to read %s: %s", filename, err)
			}
			return
		}
		// Try reading luaDataFilename as well, if possible
		luafilename := filepath.Join(filepath.Dir(filename), luaDataFilename)
		luablock, err := cache.Read(luafilename, shouldCache(ext))
		if err != nil {
			// Could not find and/or read luaDataFilename
			luablock = datablock.EmptyDataBlock
		}
		// Make functions from the given Lua data available
		funcs := make(template.FuncMap)
		// luablock can be empty if there was an error or if the file was empty
		if luablock.HasData() {
			// There was Lua code available. Now make the functions and
			// variables available for the template.
			funcs, err = luaFunctionMap(w, req, luablock.MustData(), luafilename, perm, luapool, cache, pongomutex)
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

		// Render the Amber page, using functions from luaDataFilename, if available
		amberPage(w, req, filename, amberblock.MustData(), funcs, cache)

		return

	case ".po2", ".pongo2", ".tpl", ".tmpl":

		pongoHandler(w, req, filename, ext, luaDataFilename, perm, luapool, cache, pongomutex)
		return

	case ".gcss":

		w.Header().Add("Content-Type", "text/css; charset=utf-8")
		gcssblock, err := cache.Read(filename, shouldCache(ext))
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

	case ".scss":

		w.Header().Add("Content-Type", "text/css; charset=utf-8")
		scssblock, err := cache.Read(filename, shouldCache(ext))
		if err != nil {
			if debugMode {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
			} else {
				log.Errorf("Unable to read %s: %s", filename, err)
			}
			return
		}

		// Render the SASS page as CSS
		scssPage(w, req, filename, scssblock.MustData())

		return

	case ".jsx":

		w.Header().Add("Content-Type", "text/javascript; charset=utf-8")
		jsxblock, err := cache.Read(filename, shouldCache(ext))
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
			// Create a new struct for keeping an optional http header status
			httpStatus := &FutureStatus{}
			// The flush function writes the ResponseRecorder to the ResponseWriter
			flushFunc := func() {
				writeRecorder(w, recorder)
				Flush(w)
			}
			// Run the lua script, without the possibility to flush
			if err := runLua(recorder, req, filename, perm, luapool, flushFunc, cache, httpStatus, pongomutex); err != nil {
				errortext := err.Error()
				fileblock, err := cache.Read(filename, shouldCache(ext))
				if err != nil {
					// If the file could not be read, use the error message as the data
					// Use the error as the file contents when displaying the error message
					// if reading the file failed.
					fileblock = datablock.NewDataBlock([]byte(err.Error()), true)
				}
				// If there were errors, display an error page
				prettyError(w, req, filename, fileblock.MustData(), errortext, "lua")
			} else {
				// If things went well, check if there is a status code we should write first
				// (especially for the case of a redirect)
				if httpStatus.code != 0 {
					w.WriteHeader(httpStatus.code)
				}
				// Then write to the ResponseWriter
				writeRecorder(w, recorder)
			}
		} else {
			// The flush function just flushes the ResponseWriter
			flushFunc := func() {
				Flush(w)
			}
			// Run the lua script, with the flush feature
			if err := runLua(w, req, filename, perm, luapool, flushFunc, cache, nil, pongomutex); err != nil {
				// Output the non-fatal error message to the log
				log.Error("Error in ", filename+":", err)
			}
		}

		return
	case "", ".exe", ".com", ".elf", ".tgz", ".tar.gz", ".tbz2", ".tar.bz2", ".tar.xz", ".txz", ".gz", ".zip", ".7z", ".rar", ".arj", ".lz":
		// No extension, or binary file extension
		// Set headers for downloading the file instead of displaying it in the browser.
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment")

	}

	// TODO Add support for "prettifying"/HTML-ifying some file extensions:
	// movies, music, source code etc. Wrap videos in the right html tags for playback, etc.
	// This should be placed in a separate Go module.

	// Set the correct Content-Type
	if mimereader != nil {
		mimereader.SetHeader(w, ext)
	} else {
		log.Error("Uninitialized mimereader!")
	}

	// Read the file (possibly in compressed format, straight from the cache)
	dataBlock, err := cache.Read(filename, shouldCache(ext))
	if err != nil {
		if debugMode {
			fmt.Fprintf(w, "Can't open %s: %s", filename, err)
		} else {
			log.Errorf("Can't open %s: %s", filename, err)
		}
	}

	// Serve the file
	dataBlock.ToClient(w, req, filename, clientCanGzip(req), gzipThreshold)

}

// For communicating information about the underlying software
// func powerHeader(w http.ResponseWriter, name, version string) {
//	w.Header().Set("X-Powered-By", name+"/"+version)
// }

// Server headers that are set before anything else
func serverHeaders(w http.ResponseWriter) {
	w.Header().Set("Server", serverHeaderName)
	if !autoRefreshMode {
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Content-Security-Policy", "connect-src 'self'; object-src 'self'; form-action 'self'")
	}
}

// When a file is not found
func noPage(filename, theme string) string {
	return easyPage("Not found", "File not found: "+filename, theme)
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
func registerHandlers(mux *http.ServeMux, handlePath, servedir string, perm pinterface.IPermissions, luapool *lStatePool, cache *datablock.FileCache, addDomain bool, theme string, pongomutex *sync.RWMutex) {

	// Handle all requests with this function
	allRequests := func(w http.ResponseWriter, req *http.Request) {
		// Rejecting requests is handled by the permission system, which
		// in turn requires a database backend.

		// Extra check
		if perm != nil {
			log.Fatal("No database backend!")
		}

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
		hasdir := fs.Exists(filename) && fs.IsDir(filename)
		dirname := filename
		hasfile := fs.Exists(noslash)

		// Set the server headers, if not disabled
		if !noHeaders {
			serverHeaders(w)
		}

		// Share the directory or file
		if hasdir {
			dirPage(w, req, servedir, dirname, perm, luapool, cache, theme, pongomutex)
			return
		} else if !hasdir && hasfile {
			// Share a single file instead of a directory
			filePage(w, req, noslash, defaultLuaDataFilename, perm, luapool, cache, pongomutex)
			return
		}
		// Not found
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, noPage(filename, theme))
	}

	// Handle requests differently depending on if rate limiting is enabled or not
	if disableRateLimiting {
		mux.HandleFunc(handlePath, allRequests)
	} else {
		limiter := tollbooth.NewLimiter(limitRequests, time.Second)
		limiter.MessageContentType = "text/html; charset=utf-8"
		limiter.Message = easyPage("Rate-limit exceeded", "<div style='color:red'>You have reached the maximum request limit.</div>", theme)
		mux.Handle(handlePath, tollbooth.LimitFuncHandler(limiter, allRequests))
	}
}
