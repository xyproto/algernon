package main

import (
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/didip/tollbooth"
	"github.com/xyproto/datablock"
	"github.com/xyproto/mime"
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

func (ac *algernonConfig) pongoHandler(w http.ResponseWriter, req *http.Request, filename, ext string) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	pongoblock, err := ac.cache.Read(filename, ac.shouldCache(ext))
	if err != nil {
		if ac.debugMode {
			fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
		} else {
			log.Errorf("Unable to read %s: %s", filename, err)
		}
		return
	}

	// Make the functions in luaDataFilename available for the Pongo2 template

	luafilename := filepath.Join(filepath.Dir(filename), ac.defaultLuaDataFilename)
	if fs.Exists(ac.defaultLuaDataFilename) {
		luafilename = ac.defaultLuaDataFilename
	}
	if fs.Exists(luafilename) {
		// Extract the function map from luaDataFilenname in a goroutine
		errChan := make(chan error)
		funcMapChan := make(chan template.FuncMap)

		go ac.lua2funcMap(w, req, filename, luafilename, ext, errChan, funcMapChan)
		funcs := <-funcMapChan
		err = <-errChan

		if err != nil {
			if ac.debugMode {
				// Try reading luaDataFilename as well, if possible
				luablock, luablockErr := ac.cache.Read(luafilename, ac.shouldCache(ext))
				if luablockErr != nil {
					// Could not find and/or read luaDataFilename
					luablock = datablock.EmptyDataBlock
				}
				// Use the Lua filename as the title
				ac.prettyError(w, req, luafilename, luablock.MustData(), err.Error(), "lua")
			} else {
				log.Error(err)
			}
			return
		}

		// Render the Pongo2 page, using functions from luaDataFilename, if available
		ac.pongomutex.Lock()
		ac.pongoPage(w, req, filename, pongoblock.MustData(), funcs)
		ac.pongomutex.Unlock()

		return
	}

	// Output a warning if something different from default has been given
	if !strings.HasSuffix(luafilename, ac.defaultLuaDataFilename) {
		log.Warn("Could not read ", luafilename)
	}

	// Use the Pongo2 template without any Lua functions
	ac.pongomutex.Lock()
	funcs := make(template.FuncMap)
	ac.pongoPage(w, req, filename, pongoblock.MustData(), funcs)
	ac.pongomutex.Unlock()

	return
}

func (ac *algernonConfig) readAndLogErrors(w http.ResponseWriter, filename, ext string) (*datablock.DataBlock, error) {
	byteblock, err := ac.cache.Read(filename, ac.shouldCache(ext))
	if err != nil {
		if ac.debugMode {
			fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
		} else {
			log.Errorf("Unable to read %s: %s", filename, err)
		}
	}
	return byteblock, err
}

// When serving a file. The file must exist. Must be given a full filename.
func (ac *algernonConfig) filePage(w http.ResponseWriter, req *http.Request, filename, dataFilename string) {

	if ac.quitAfterFirstRequest {
		go ac.quitSoon("Quit after first request", defaultSoonDuration)
	}

	// Mimetypes
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {

	// HTML pages are handled differently, if auto-refresh has been enabled
	case ".html", ".htm":
		w.Header().Add("Content-Type", "text/html; charset=utf-8")

		// Read the file (possibly in compressed format, straight from the cache)
		htmlblock, err := ac.readAndLogErrors(w, filename, ext)
		if err != nil {
			return
		}

		// If the auto-refresh feature has been enabled
		if ac.autoRefreshMode {
			// Get the bytes from the datablock
			htmldata := htmlblock.MustData()
			// Insert JavaScript for refreshing the page, into the HTML
			htmldata = ac.insertAutoRefresh(req, htmldata)
			// Write the data to the client
			dataToClient(w, req, filename, htmldata)
		} else {
			// Serve the file
			htmlblock.ToClient(w, req, filename, clientCanGzip(req), gzipThreshold)
		}

		return

	// Markdown pages are handled differently
	case ".md", ".markdown":
		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		if markdownblock, err := ac.readAndLogErrors(w, filename, ext); err == nil {
			// Render the markdown page
			ac.markdownPage(w, req, markdownblock.MustData(), filename)
		}
		return

	case ".amber", ".amb":
		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		amberblock, err := ac.readAndLogErrors(w, filename, ext)
		if err != nil {
			return
		}

		// Try reading luaDataFilename as well, if possible
		luafilename := filepath.Join(filepath.Dir(filename), ac.defaultLuaDataFilename)
		luablock, err := ac.cache.Read(luafilename, ac.shouldCache(ext))
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
			funcs, err = ac.luaFunctionMap(w, req, luablock.MustData(), luafilename)
			if err != nil {
				if ac.debugMode {
					// Use the Lua filename as the title
					ac.prettyError(w, req, luafilename, luablock.MustData(), err.Error(), "lua")
				} else {
					log.Error(err)
				}
				return
			}
			if ac.debugMode && ac.verboseMode {
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
		ac.amberPage(w, req, filename, amberblock.MustData(), funcs)

		return

	case ".po2", ".pongo2", ".tpl", ".tmpl":
		ac.pongoHandler(w, req, filename, ext)
		return

	case ".gcss":
		w.Header().Add("Content-Type", "text/css; charset=utf-8")
		if gcssblock, err := ac.readAndLogErrors(w, filename, ext); err == nil {
			// Render the GCSS page as CSS
			ac.gcssPage(w, req, filename, gcssblock.MustData())
		}
		return

	case ".scss":
		w.Header().Add("Content-Type", "text/css; charset=utf-8")
		if scssblock, err := ac.readAndLogErrors(w, filename, ext); err == nil {
			// Render the SASS page as CSS
			ac.scssPage(w, req, filename, scssblock.MustData())
		}
		return

	case ".jsx":
		w.Header().Add("Content-Type", "text/javascript; charset=utf-8")
		if jsxblock, err := ac.readAndLogErrors(w, filename, ext); err == nil {
			// Render the JSX page as JavaScript
			ac.jsxPage(w, req, filename, jsxblock.MustData())
		}
		return

	case ".lua":
		// If in debug mode, let the Lua script print to a buffer first, in
		// case there are errors that should be displayed instead.

		// If debug mode is enabled
		if ac.debugMode {
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
			if err := ac.runLua(recorder, req, filename, flushFunc, httpStatus); err != nil {
				errortext := err.Error()
				fileblock, err := ac.cache.Read(filename, ac.shouldCache(ext))
				if err != nil {
					// If the file could not be read, use the error message as the data
					// Use the error as the file contents when displaying the error message
					// if reading the file failed.
					fileblock = datablock.NewDataBlock([]byte(err.Error()), true)
				}
				// If there were errors, display an error page
				ac.prettyError(w, req, filename, fileblock.MustData(), errortext, "lua")
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
			if err := ac.runLua(w, req, filename, flushFunc, nil); err != nil {
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
	if dataBlock, err := ac.readAndLogErrors(w, filename, ext); err == nil {
		// Serve the file
		dataBlock.ToClient(w, req, filename, clientCanGzip(req), gzipThreshold)
	}

}

// For communicating information about the underlying software
// func powerHeader(w http.ResponseWriter, name, version string) {
//	w.Header().Set("X-Powered-By", name+"/"+version)
// }

// Server headers that are set before anything else
func (ac *algernonConfig) serverHeaders(w http.ResponseWriter) {
	w.Header().Set("Server", ac.serverHeaderName)
	if !ac.autoRefreshMode {
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Content-Security-Policy", "connect-src 'self'; object-src 'self'; form-action 'self'")
	}
}

// When a file is not found
func noPage(filename, theme string) string {
	return messagePage("Not found", "File not found: "+filename, theme)
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
func (ac *algernonConfig) registerHandlers(mux *http.ServeMux, handlePath, servedir string, addDomain bool) {

	// Handle all requests with this function
	allRequests := func(w http.ResponseWriter, req *http.Request) {
		// Rejecting requests is handled by the permission system, which
		// in turn requires a database backend.
		if ac.perm != nil {
			if ac.perm.Rejected(w, req) {
				// Get and call the Permission Denied function
				ac.perm.DenyFunction()(w, req)
				// Reject the request by returning
				return
			}
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
		if !ac.noHeaders {
			ac.serverHeaders(w)
		}

		// Share the directory or file
		if hasdir {
			ac.dirPage(w, req, servedir, dirname, ac.defaultTheme)
			return
		} else if !hasdir && hasfile {
			// Share a single file instead of a directory
			ac.filePage(w, req, noslash, ac.defaultLuaDataFilename)
			return
		}
		// Not found
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, noPage(filename, ac.defaultTheme))
	}

	// Handle requests differently depending on if rate limiting is enabled or not
	if ac.disableRateLimiting {
		mux.HandleFunc(handlePath, allRequests)
	} else {
		limiter := tollbooth.NewLimiter(ac.limitRequests, time.Second)
		limiter.MessageContentType = "text/html; charset=utf-8"
		limiter.Message = messagePage("Rate-limit exceeded", "<div style='color:red'>You have reached the maximum request limit.</div>", ac.defaultTheme)
		mux.Handle(handlePath, tollbooth.LimitFuncHandler(limiter, allRequests))
	}
}
