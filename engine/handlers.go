package engine

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/didip/tollbooth"
	"github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/themes"
	"github.com/xyproto/algernon/utils"
	"github.com/xyproto/datablock"
	"github.com/xyproto/ollamaclient/v2"
	"github.com/xyproto/sheepcounter"
	"github.com/xyproto/simpleform"
	"github.com/xyproto/unzip"
)

const (
	// Gzip content over this size
	gzipThreshold = 4096

	// Used for deciding how long to wait before quitting when only serving a single file and starting a browser
	defaultSoonDuration = time.Second * 3

	contentType = "Content-Type"
	htmlUTF8    = "text/html;charset=utf-8"
	textUTF8    = "text/plain;charset=utf-8"
)

var oc *ollamaclient.Config

// ClientCanGzip checks if the client supports gzip compressed responses
func (ac *Config) ClientCanGzip(req *http.Request) bool {
	// Curl does not use --compressed by default. This causes problems when
	// serving gzipped contents when curl is run without --compressed!
	// The wrong data, of the same size, will be downloaded. Beware!
	if ac.curlSupport {
		return strings.Contains(req.Header.Get("Accept-Encoding"), "gzip")
	}
	// Modern browsers support gzip
	return true
}

// PongoHandler renders and serves a Pongo2 template
func (ac *Config) PongoHandler(w http.ResponseWriter, req *http.Request, filename, ext string) {
	w.Header().Add(contentType, htmlUTF8)
	pongoblock, err := ac.cache.Read(filename, ac.shouldCache(ext))
	if err != nil {
		if ac.debugMode {
			fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
		} else {
			logrus.Errorf("Unable to read %s: %s", filename, err)
		}
		return
	}

	// Make the functions in luaDataFilename available for the Pongo2 template

	luafilename := defaultLuaDataFilename
	if !ac.fs.Exists(luafilename) {
		luafilename = filepath.Join(filepath.Dir(filename), defaultLuaDataFilename)
	}

	if ac.fs.Exists(luafilename) {
		// Extract the function map from luaDataFilenname in a goroutine
		errChan := make(chan error)
		funcMapChan := make(chan template.FuncMap)

		go ac.Lua2funcMap(w, req, filename, luafilename, ext, errChan, funcMapChan)
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
				ac.PrettyError(w, req, luafilename, luablock.Bytes(), err.Error(), "lua")
			} else {
				logrus.Error(err)
			}
			return
		}

		// Render the Pongo2 page, using functions from luaDataFilename, if available
		ac.pongomutex.Lock()
		ac.PongoPage(w, req, filename, pongoblock.Bytes(), funcs)
		ac.pongomutex.Unlock()

		return
	}

	// Output a warning if something different from default has been given
	// TODO: Do not only check for a suffix, check for the filename
	if !strings.HasSuffix(luafilename, defaultLuaDataFilename) {
		logrus.Warn("Could not read ", luafilename)
	}

	// Use the Pongo2 template without any Lua functions
	ac.pongomutex.Lock()
	funcs := make(template.FuncMap)
	ac.PongoPage(w, req, filename, pongoblock.Bytes(), funcs)
	ac.pongomutex.Unlock()
}

// ReadAndLogErrors tries to read a file, and logs an error if it could not be read
func (ac *Config) ReadAndLogErrors(w http.ResponseWriter, filename, ext string) (*datablock.DataBlock, error) {
	byteblock, err := ac.cache.Read(filename, ac.shouldCache(ext))
	if err != nil {
		if ac.debugMode {
			fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
		} else {
			logrus.Errorf("Unable to read %s: %s", filename, err)
		}
	}
	return byteblock, err
}

// FilePage tries to serve a single file. The file must exist. Must be given a full filename.
func (ac *Config) FilePage(w http.ResponseWriter, req *http.Request, filename, luaDataFilename string) {
	if ac.quitAfterFirstRequest {
		go ac.quitSoon("Quit after first request", defaultSoonDuration)
	}

	// Use the file extension for setting the mimetype
	lowercaseFilename := strings.ToLower(filename)
	ext := filepath.Ext(lowercaseFilename)

	// Filenames ending with .hyper.js or .hyper.jsx are special cases
	if strings.HasSuffix(lowercaseFilename, ".hyper.js") {
		ext = ".hyper.js"
	} else if strings.HasSuffix(lowercaseFilename, ".hyper.jsx") {
		ext = ".hyper.jsx"
	}

	// Serve the file in different ways based on the filename extension
	switch ext {

	// HTML pages are handled differently, if auto-refresh has been enabled
	case ".html", ".htm":
		w.Header().Add(contentType, htmlUTF8)

		// Read the file (possibly in compressed format, straight from the cache)
		htmlblock, err := ac.ReadAndLogErrors(w, filename, ext)
		if err != nil {
			return
		}

		// If the auto-refresh feature has been enabled
		if ac.autoRefresh {
			// Get the bytes from the datablock
			htmldata := htmlblock.Bytes()
			// Insert JavaScript for refreshing the page, into the HTML
			htmldata = ac.InsertAutoRefresh(req, htmldata)
			// Write the data to the client
			ac.DataToClient(w, req, filename, htmldata)
		} else {
			// Serve the file
			htmlblock.ToClient(w, req, filename, ac.ClientCanGzip(req), gzipThreshold)
		}

		return

	case ".md", ".markdown":
		w.Header().Add(contentType, htmlUTF8)
		if markdownblock, err := ac.ReadAndLogErrors(w, filename, ext); err == nil { // success
			// Render the markdown page
			ac.MarkdownPage(w, req, markdownblock.Bytes(), filename)
		}
		return

	case ".frm", ".form":
		w.Header().Add(contentType, htmlUTF8)
		formblock, err := ac.cache.Read(filename, ac.shouldCache(ext))
		if err != nil {
			return
		}
		// Render the form file as just the HTML body, not the surrounding document
		// (between <body> and </body>)
		html, err := simpleform.HTML(formblock.String(), false, "en")
		if err != nil {
			return
		}
		w.Write([]byte(html))
		return

	case ".amber", ".amb":
		w.Header().Add(contentType, htmlUTF8)
		amberblock, err := ac.ReadAndLogErrors(w, filename, ext)
		if err != nil {
			return
		}

		// Try reading luaDataFilename as well, if possible
		luafilename := filepath.Join(filepath.Dir(filename), luaDataFilename)
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
			funcs, err = ac.LuaFunctionMap(w, req, luablock.Bytes(), luafilename)
			if err != nil {
				if ac.debugMode {
					// Use the Lua filename as the title
					ac.PrettyError(w, req, luafilename, luablock.Bytes(), err.Error(), "lua")
				} else {
					logrus.Error(err)
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
				s = strings.TrimSuffix(s, ", ")
				// Output the message
				logrus.Info(s)
			}
		}

		// Render the Amber page, using functions from luaDataFilename, if available
		ac.AmberPage(w, req, filename, amberblock.Bytes(), funcs)

		return

	case ".po2", ".pongo2", ".tpl", ".tmpl":
		ac.PongoHandler(w, req, filename, ext)
		return

	case ".alg":
		// Assume this to be a compressed Algernon application
		webApplicationExtractionDir := "/dev/shm" // extract to memory, if possible
		testfile := filepath.Join(webApplicationExtractionDir, "canary")
		if _, err := os.Create(testfile); err == nil { // success
			os.Remove(testfile)
		} else {
			// Could not create the test file
			// Use the server temp dir (typically /tmp) instead of /dev/shm
			webApplicationExtractionDir = ac.serverTempDir
		}
		if extractErr := unzip.Extract(filename, webApplicationExtractionDir); extractErr == nil { // no error
			firstname := path.Base(filename)
			if strings.HasSuffix(filename, ".alg") {
				firstname = path.Base(filename[:len(filename)-4])
			}
			serveDir := path.Join(webApplicationExtractionDir, firstname)
			logrus.Warn(".alg web applications must be given as an argument to algernon to be served correctly")
			ac.DirPage(w, req, serveDir, serveDir, ac.defaultTheme, luaDataFilename)
		}
		return

	case ".lua", ".tl":
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
				// If things went well, check if there is a status code we should write first
				// (especially for the case of a redirect)
				if httpStatus.code != 0 {
					recorder.WriteHeader(httpStatus.code)
				} else {
					recorder.WriteHeader(http.StatusOK)
				}
				// Then write to the ResponseWriter
				n, err := WriteRecorder(w, recorder) // WriteRecorder starts out by writing the status header
				if n > 0 && err == nil {
					if flusher, ok := w.(http.Flusher); ok {
						flusher.Flush()
					}
				} else if err != nil {
					if req.Close {
						if ac.debugMode {
							logrus.Warn(filename + ": call to \"flush\" after closing the connection")
						}
					} else {
						logrus.Warn(filename + ": WriteRecorder: " + err.Error())
						// Close the connection. Works for both HTTP and HTTP/2 now, ref: https://github.com/golang/go/issues/20977
						w.Header().Add("Connection", "close")
						req.Close = true
					}
				}
			}
			// Run the lua script, without the possibility to flush
			if err := ac.RunLua(recorder, req, filename, flushFunc, httpStatus); err != nil {
				errortext := err.Error()
				fileblock, err := ac.cache.Read(filename, ac.shouldCache(ext))
				if err != nil {
					// If the file could not be read, use the error message as the data
					// Use the error as the file contents when displaying the error message
					// if reading the file failed.
					fileblock = datablock.NewDataBlock([]byte(err.Error()), true)
				}
				// If there were errors, display an error page
				ac.PrettyError(w, req, filename, fileblock.Bytes(), errortext, "lua")
			} else {
				// If things went well, check if there is a status code we should write first
				// (especially for the case of a redirect)
				if httpStatus.code != 0 {
					recorder.WriteHeader(httpStatus.code)
				} else {
					recorder.WriteHeader(http.StatusOK)
				}
				// Then write to the ResponseWriter
				_, writeErr := WriteRecorder(w, recorder) // WriteRecorder starts out by writing the status header
				// Note: no flushing here, because there was an error when running the Lua script
				if writeErr != nil {
					if req.Close {
						if ac.debugMode {
							logrus.Warn(filename + ": call to \"flush\" after closing the connection")
						}
					} else {
						logrus.Warn(filename + ": WriteRecorder: " + writeErr.Error())
						// Close the connection. Works for both HTTP and HTTP/2 now, ref: https://github.com/golang/go/issues/20977
						w.Header().Add("Connection", "close")
						req.Close = true
					}
				}
			}
		} else {
			// The flush function just flushes the ResponseWriter
			flushFunc := func() {
				w.WriteHeader(http.StatusOK)
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
			}
			// Run the lua script, with the flush feature
			if err := ac.RunLua(w, req, filename, flushFunc, nil); err != nil {
				// Output the non-fatal error message to the log
				if strings.HasPrefix(err.Error(), filename) {
					logrus.Error("Error at " + err.Error())
				} else {
					logrus.Error("Error in " + filename + ": " + err.Error())
				}
			}
		}
		return

	case ".gcss":
		if gcssblock, err := ac.ReadAndLogErrors(w, filename, ext); err == nil { // success
			w.Header().Add(contentType, "text/css;charset=utf-8")
			// Render the GCSS page as CSS
			ac.GCSSPage(w, req, filename, gcssblock.Bytes())
		}
		return

	case ".scss":
		if scssblock, err := ac.ReadAndLogErrors(w, filename, ext); err == nil { // success
			// Render the SASS page (with .scss extension) as CSS
			w.Header().Add(contentType, "text/css;charset=utf-8")
			ac.SCSSPage(w, req, filename, scssblock.Bytes())
		}
		return

	case ".happ", ".hyper", ".hyper.jsx", ".hyper.js": // hyperApp JSX -> JS, wrapped in HTML
		if jsxblock, err := ac.ReadAndLogErrors(w, filename, ext); err == nil { // success
			// Render the JSX page as HTML with embedded JavaScript
			w.Header().Add(contentType, htmlUTF8)
			ac.HyperAppPage(w, req, filename, jsxblock.Bytes())
		} else {
			logrus.Error(filename + ":" + err.Error())
		}
		return

	// This case must come after the .hyper.jsx case
	case ".jsx":
		if jsxblock, err := ac.ReadAndLogErrors(w, filename, ext); err == nil { // success
			// Render the JSX page as JavaScript
			w.Header().Add(contentType, "text/javascript;charset=utf-8")
			ac.JSXPage(w, req, filename, jsxblock.Bytes())
		} else {
			logrus.Error(filename + ":" + err.Error())
		}
		return

	// .prompt files contains a content type and a prompt that is converted to data in a reproducible way, with a newline between them
	case ".prompt":
		if promptblock, err := ac.ReadAndLogErrors(w, filename, ext); err == nil { // success
			lines := strings.Split(promptblock.String(), "\n")
			if len(lines) < 4 {
				logrus.Error(filename + " must contain a content type, a model name, a blank line and then a prompt to be usable")
			} else if strings.TrimSpace(lines[2]) != "" {
				logrus.Error(filename + " must contain a content type, a model name, a blank line and then a prompt to be usable")
			} else {
				contentType := strings.TrimSpace(lines[0])
				model := strings.TrimSpace(lines[1])
				prompt := strings.TrimSpace(strings.Join(lines[3:], "\n"))
				w.Header().Add(contentType, contentType)
				if oc == nil {
					oc = ollamaclient.New()
				}
				if oc.ModelName != model {
					oc.ModelName = model
				}
				oc.HTTPTimeout = time.Duration(ac.writeTimeout) * time.Second
				if err := oc.PullIfNeeded(true); err == nil { // success
					if output, err := oc.GetOutput(prompt); err == nil { // success
						if strings.Contains(output, "<") && strings.Contains(output, ">") {
							output = betweenInclusive(output, "<", ">")
						}
						w.Write([]byte(output))
					} else {
						logrus.Error("could not convert " + filename + " to content: " + err.Error())
					}
				} else {
					logrus.Error("could not convert " + filename + " to content: " + err.Error())
				}
			}
		} else {
			logrus.Error(filename + ":" + err.Error())
		}
		return

	// --- End of special handlers that returns early ---

	// Text and configuration files (most likely)
	case "", ".asciidoc", ".conf", ".config", ".diz", ".example", ".gitignore", ".gitmodules", ".ini", ".log", ".lst", ".me", ".nfo", ".pem", ".readme", ".sub", ".sum", ".tml", ".toml", ".txt", ".yaml", ".yml":
		// Set headers for displaying it in the browser.
		w.Header().Set(contentType, textUTF8)

	// Source files that may be used by web pages
	case ".js":
		w.Header().Add(contentType, "text/javascript;charset=utf-8")

	// JSON
	case ".json":
		w.Header().Add(contentType, "application/json;charset=utf-8")

	// Source code files for viewing
	case ".S", ".ada", ".asm", ".bash", ".bat", ".c", ".c++", ".cc", ".cl", ".clj", ".cpp", ".cs", ".cxx", ".el", ".elm", ".erl", ".fish", ".go", ".h", ".h++", ".hpp", ".hs", ".java", ".kt", ".lisp", ".mk", ".ml", ".pas", ".pl", ".py", ".r", ".rb", ".rs", ".scm", ".sh", ".ts", ".tsx":
		// Set headers for displaying it in the browser.
		w.Header().Set(contentType, textUTF8)

	// Common binary file extensions
	case ".7z", ".arj", ".bin", ".com", ".dat", ".db", ".elf", ".exe", ".gz", ".iso", ".lz", ".rar", ".tar.bz", ".tar.bz2", ".tar.gz", ".tar.xz", ".tbz", ".tbz2", ".tgz", ".txz", ".xz", ".zip":
		// Set headers for downloading the file instead of displaying it in the browser.
		w.Header().Set("Content-Disposition", "attachment")

	default:
		// If the filename starts with a ".", assume it's a plain text configuration file
		if strings.HasPrefix(filepath.Base(lowercaseFilename), ".") {
			w.Header().Set(contentType, textUTF8)
		} else {
			// Set the correct Content-Type
			if ac.mimereader != nil {
				ac.mimereader.SetHeader(w, ext)
			} else {
				logrus.Error("Uninitialized mimereader!")
			}
		}
	}

	// TODO Add support for "prettifying"/HTML-ifying some file extensions:
	// movies, music, source code etc. Wrap videos in the right html tags for playback, etc.
	// This should be placed in a separate Go module.

	// TODO: Modify ac.fs to also cache .Size(), .Name() and .ModTime()

	// Check the size of the file
	f, err := os.Open(filename)
	if err != nil {
		logrus.Error("Could not open " + filename + "! " + err.Error())
		return
	}
	defer f.Close()
	fInfo, err := f.Stat()
	if err != nil {
		logrus.Error("Could not stat " + filename + "! " + err.Error())
		return
	}

	// Check if the file is so large that it needs to be streamed directly
	fileSize := uint64(fInfo.Size())
	// Cache size can be set to a low number to trigger this behavior
	if fileSize > ac.largeFileSize {
		// logrus.Info("Streaming " + filename + " directly...")

		// http.ServeContent will first seek to the end of the file, then
		// serve the file. The alternative here is to use io.Copy(w, f),
		// but io.Copy does not support ranges.
		http.ServeContent(w, req, fInfo.Name(), fInfo.ModTime(), f)

		return
	}

	// Read the file (possibly in compressed format, straight from the cache)
	if dataBlock, err := ac.ReadAndLogErrors(w, filename, ext); err == nil { // success
		// Serve the file
		dataBlock.ToClient(w, req, filename, ac.ClientCanGzip(req), gzipThreshold)
	} else {
		logrus.Error("Could not serve " + filename + " with datablock.ToClient: " + err.Error())
		return
	}
}

// ServerHeaders sets the HTTP headers that are set before anything else
func (ac *Config) ServerHeaders(w http.ResponseWriter) {
	w.Header().Set("Server", ac.serverHeaderName)
	if !ac.autoRefresh {
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	}
	if !ac.autoRefresh && ac.stricterHeaders {
		w.Header().Set("Content-Security-Policy",
			"connect-src 'self'; object-src 'self'; form-action 'self'")
	}
	// w.Header().Set("X-Powered-By", name+"/"+version)
}

// RegisterHandlers configures the given mutex and request limiter to handle
// HTTP requests
func (ac *Config) RegisterHandlers(mux *http.ServeMux, handlePath, servedir string, addDomain bool) {
	theme := ac.defaultTheme
	// Theme aliases. Use a map if there are more than 2 aliases in the future.
	if theme == "light" {
		// The "light" theme is the "gray" theme
		theme = "gray"
	}

	// Handle all requests with this function
	allRequests := func(w http.ResponseWriter, req *http.Request) {
		// Rejecting requests is handled by the permission system, which
		// in turn requires a database backend.
		if ac.perm != nil {
			if ac.perm.Rejected(w, req) {
				// Prepare to count bytes written
				sc := sheepcounter.New(w)
				// Get and call the Permission Denied function
				ac.perm.DenyFunction()(sc, req)
				// Log the response
				ac.LogAccess(req, http.StatusForbidden, sc.Counter())
				// Reject the request by just returning
				return
			}
		}

		// Local to this function
		servedir := servedir

		// Look for the directory that is named the same as the host
		if addDomain {
			servedir = filepath.Join(servedir, utils.GetDomain(req))
		}

		urlpath := req.URL.Path

		//logrus.Infoln("Checking reverse proxy", urlpath, ac.reverseProxyConfig)
		if ac.reverseProxyConfig != nil {
			if rproxy := ac.reverseProxyConfig.FindMatchingReverseProxy(urlpath); rproxy != nil {
				//logrus.Infof("Querying reverse proxy %+v, %+v\n", rproxy, req)
				res, err := rproxy.DoProxyPass(*req)
				if err != nil {
					w.WriteHeader(http.StatusBadGateway)
					w.Write([]byte("reverse proxy error, please check your server config for AddReverseProxy calls\n"))
					return
				}
				data, err := io.ReadAll(res.Body)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				res.Body.Close()
				for k, vals := range res.Header {
					for _, v := range vals {
						w.Header().Set(k, v)
					}
				}
				w.WriteHeader(res.StatusCode)
				w.Write(data)
				return
			}
		}

		filename := utils.URL2filename(servedir, urlpath)
		// Remove the trailing slash from the filename, if any
		noslash := filename
		if strings.HasSuffix(filename, utils.Pathsep) {
			noslash = filename[:len(filename)-1]
		}
		hasdir := ac.fs.Exists(filename) && ac.fs.IsDir(filename)
		dirname := filename
		hasfile := ac.fs.Exists(noslash)

		// Set the server headers, if not disabled
		if !ac.noHeaders {
			ac.ServerHeaders(w)
		}

		// Share the directory or file
		if hasdir {
			// Prepare to count bytes written
			sc := sheepcounter.New(w)
			// Get the directory page
			ac.DirPage(sc, req, servedir, dirname, theme, defaultLuaDataFilename)
			// Log the access
			ac.LogAccess(req, http.StatusOK, sc.Counter())
			return
		} else if !hasdir && hasfile {
			// Prepare to count bytes written
			sc := sheepcounter.New(w)
			// Share a single file instead of a directory
			ac.FilePage(sc, req, noslash, defaultLuaDataFilename)
			// Log the access
			ac.LogAccess(req, http.StatusOK, sc.Counter())
			return
		}
		// Not found
		w.WriteHeader(http.StatusNotFound)
		data := themes.NoPage(filename, theme)
		ac.LogAccess(req, http.StatusNotFound, int64(len(data)))
		w.Write(data)
	}

	// Handle requests differently depending on rate limiting being enabled or not
	if ac.disableRateLimiting {
		mux.HandleFunc(handlePath, allRequests)
	} else {
		limiter := tollbooth.NewLimiter(float64(ac.limitRequests), nil)
		limiter.SetMessage(themes.MessagePage("Rate-limit exceeded", "<div style='color:red'>You have reached the maximum request limit.</div>", theme))
		limiter.SetMessageContentType(htmlUTF8)
		mux.Handle(handlePath, tollbooth.LimitFuncHandler(limiter, allRequests))
	}
}
