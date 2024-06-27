package engine

// This source file is for the special case of serving a single file.

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
	"github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/utils"
	"github.com/xyproto/datablock"
)

const (
	defaultStaticCacheSize            = 128 * utils.MiB
	maxAttemptsAtIncreasingPortNumber = 128
	delayBeforeLaunchingBrowser       = time.Millisecond * 200
)

// nextPort increases the port number by 1
func nextPort(colonPort string) (string, error) {
	if !strings.HasPrefix(colonPort, ":") {
		return colonPort, errors.New("colonPort does not start with a colon! \"" + colonPort + "\"")
	}
	num, err := strconv.Atoi(colonPort[1:])
	if err != nil {
		return colonPort, errors.New("Could not convert port number to string: \"" + colonPort[1:] + "\"")
	}
	// Increase the port number by 1, add a colon, convert to string and return
	return ":" + strconv.Itoa(num+1), nil
}

// This is a bit hacky, but it's only used when serving a single static file
func (ac *Config) openAfter(wait time.Duration, hostname, colonPort string, https bool, cancelChannel chan bool) {
	// Wait a bit
	time.Sleep(wait)
	select {
	case <-cancelChannel:
		// Got a message on the cancelChannel:
		// don't open the URL with an external application.
		return
	case <-time.After(delayBeforeLaunchingBrowser):
		// Got timeout, assume the port was not busy
		ac.OpenURL(hostname, colonPort, https)
	}
}

// shortInfo outputs a short string about which file is served where
func (ac *Config) shortInfoAndOpen(filename, colonPort string, cancelChannel chan bool) {
	hostname := "localhost"
	if ac.serverHost != "" {
		hostname = ac.serverHost
	}
	logrus.Infof("Serving %s on http://%s%s", filename, hostname, colonPort)

	if ac.openURLAfterServing {
		go ac.openAfter(delayBeforeLaunchingBrowser, hostname, colonPort, false, cancelChannel)
	}
}

// ServeStaticFile is a convenience function for serving only a single file.
// It can be used as a quick and easy way to view a README.md file.
// Will also serve local images if the resulting HTML contains them.
func (ac *Config) ServeStaticFile(filename, colonPort string) error {
	logrus.Info("Single file mode. Not using the regular parameters.")

	cancelChannel := make(chan bool, 1)

	ac.shortInfoAndOpen(filename, colonPort, cancelChannel)

	mux := http.NewServeMux()

	// 64 MiB cache, use cache compression, no per-file size limit, use best gzip compression, compress for size not for speed
	ac.cache = datablock.NewFileCache(defaultStaticCacheSize, true, 0, false, 0)

	if ac.markdownMode {
		// Discover all local images mentioned in the Markdown document
		var localImages []string
		if markdownData, err := ac.cache.Read(filename, true); err == nil { // success
			// Create a Markdown parser with the desired extensions
			mdParser := parser.NewWithExtensions(enabledMarkdownExtensions)
			// Convert from Markdown to HTML
			mdContent := markdownData.Bytes()
			htmlData := markdown.ToHTML(mdContent, mdParser, nil)

			// Add a script for rendering MathJax, but only if at least one mathematical formula is present
			if containsFormula(mdContent) {
				js := append([]byte(`<script id="MathJax-script">`), []byte(mathJaxScript)...)
				htmlData = InsertScriptTag(htmlData, js) // also adds the closing </script> tag
			}

			localImages = utils.ExtractLocalImagePaths(string(htmlData))
		}

		// Serve all local images mentioned in the Markdown document.
		// If a file is not found, then the FilePage function will handle it.
		for _, localImage := range localImages {
			mux.HandleFunc("/"+localImage, func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Server", ac.versionString)
				ac.FilePage(w, req, localImage, defaultLuaDataFilename)
			})
		}
	}

	// Prepare to serve the given filename

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Server", ac.versionString)
		ac.FilePage(w, req, filename, defaultLuaDataFilename)
	})

	HTTPserver := ac.NewGracefulServer(mux, false, ac.serverHost+colonPort)

	// Attempt to serve the handler functions above
	if errServe := HTTPserver.ListenAndServe(); errServe != nil {
		// If it fails, try several times, increasing the port by 1 each time
		for i := 0; i < maxAttemptsAtIncreasingPortNumber; i++ {
			if errServe = HTTPserver.ListenAndServe(); errServe != nil {
				cancelChannel <- true
				if !strings.HasSuffix(errServe.Error(), "already in use") {
					// Not a problem with address already being in use
					ac.fatalExit(errServe)
				}
				logrus.Warn("Address already in use. Using next port number.")
				if newPort, errNext := nextPort(colonPort); errNext != nil {
					ac.fatalExit(errNext)
				} else {
					colonPort = newPort
				}

				// Make a new cancel channel, and use the new URL
				cancelChannel = make(chan bool, 1)
				ac.shortInfoAndOpen(filename, colonPort, cancelChannel)

				HTTPserver = ac.NewGracefulServer(mux, false, ac.serverHost+colonPort)
			}
		}
		// Several attempts failed
		return errServe
	}

	return nil
}
