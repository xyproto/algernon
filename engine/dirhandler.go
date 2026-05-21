package engine

// Directory Index

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-gcfg/gcfg"
	"github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/platformdep"
	"github.com/xyproto/algernon/themes"
	"github.com/xyproto/algernon/utils"
)

const (
	dotSlash = "." + utils.Pathsep           /* ./ */
	doubleP  = utils.Pathsep + utils.Pathsep /* // */
)

// List of filenames that should be displayed instead of a directory listing
var indexFilenames = []string{"index.lua", "index.html", "index.md", "index.txt", "index.pongo2", "index.tmpl", "index.po2", "index.amber", "index.jsx", "index.tsx", "index.happ", "index.hyper", "index.hyper.js", "index.hyper.jsx", "index.tl", "index.prompt"}

// DirConfig keeps a directory listing configuration
type DirConfig struct {
	Main struct {
		Title string
		Theme string
	}
}

// dirConfigEntry is a cached parsed .algernon configuration with mtime
type dirConfigEntry struct {
	config DirConfig
	mtime  time.Time
}

// dirConfigCache caches parsed .algernon configurations by filename and mtime
type dirConfigCache struct {
	mu      sync.RWMutex
	entries map[string]dirConfigEntry
}

// DirectoryListing serves the given directory as a web page with links the the contents
func (ac *Config) DirectoryListing(w http.ResponseWriter, req *http.Request, rootdir, dirname, theme string) {
	var (
		buf          bytes.Buffer
		fullFilename string
		URLpath      string
		title        = dirname
	)

	// Remove a trailing slash after the root directory, if present
	rootdir = strings.TrimSuffix(rootdir, "/")

	// Read ignore patterns if ignore.txt is present
	var ignorePatterns []string
	ignoreFilePath := filepath.Join(dirname, platformdep.IgnoreFilename)
	if ac.fs.Exists(ignoreFilePath) {
		patterns, err := ReadIgnoreFile(ignoreFilePath)
		if err != nil {
			logrus.Warn("Could not read ignore.txt: ", err)
		} else {
			ignorePatterns = patterns
		}
	}

	// Fill the coming HTML body with a list of all the filenames in `dirname`
	for _, filename := range utils.GetFilenames(dirname) {

		if filename == platformdep.DirConfFilename || filename == platformdep.IgnoreFilename {
			// Skip
			continue
		}

		// Skip dotfiles/dot-directories in listings if --hide-dotfiles is set
		if ac.hideDotfiles && strings.HasPrefix(filename, ".") {
			continue
		}

		// Check if the filename matches any of the ignore patterns
		if ShouldIgnore(filename, ignorePatterns) {
			continue
		}

		// Find the full name
		fullFilename = dirname

		// Add a "/" after the directory name, if missing
		if !strings.HasSuffix(fullFilename, utils.Pathsep) {
			fullFilename += utils.Pathsep
		}

		// Add the filename at the end
		fullFilename += filename

		// Remove the root directory from the link path
		URLpath = fullFilename[len(rootdir)+1:]

		// Output different entries for files and directories
		buf.WriteString(themes.HTMLLink(filename, URLpath, ac.fs.IsDir(fullFilename)))
	}

	// Read directory configuration, if present
	if fullDirConfFilename := filepath.Join(dirname, platformdep.DirConfFilename); ac.fs.Exists(fullDirConfFilename) {
		dirConf := ac.readDirConfig(fullDirConfFilename)
		if dirConf.Main.Title != "" {
			title = dirConf.Main.Title
		}
		if dirConf.Main.Theme != "" {
			theme = dirConf.Main.Theme
		}
	} else {
		// Strip the leading "./" from the current directory
		title = strings.TrimPrefix(title, dotSlash)

		// Replace "//" with just "/"
		title = strings.ReplaceAll(title, doubleP, utils.Pathsep)
	}

	// Check if the current page contents are empty
	if buf.Len() == 0 {
		buf.WriteString("Empty directory")
	}

	htmldata := themes.MessagePageBytes(title, buf.Bytes(), theme)

	// If the auto-refresh feature has been enabled
	if ac.autoRefresh {
		// Insert JavaScript for refreshing the page into the generated HTML
		htmldata = ac.InsertAutoRefresh(req, htmldata)
	}

	// Serve the page
	w.Header().Add(contentType, htmlUTF8)
	ac.DataToClient(w, req, dirname, htmldata)
}

// DirPage serves a directory, using index.* files, if present.
// The directory must exist.
// rootdir is the base directory (can be ".")
// dirname is the specific directory that is to be served (should never be ".")
func (ac *Config) DirPage(w http.ResponseWriter, req *http.Request, rootdir, dirname, theme, luaDataFilename string) {
	// Check if we are instructed to quit after serving the first file
	if ac.quitAfterFirstRequest {
		go ac.quitSoon("Quit after first request", defaultSoonDuration)
	}

	// If the URL does not end with a slash, redirect to an URL that does
	if !strings.HasSuffix(req.URL.Path, "/") {
		if req.Method == "POST" {
			logrus.Warn("Redirecting a POST request: " + req.URL.Path + " -> " + req.URL.Path + "/.")
			logrus.Warn("Header data may be lost! Please add the missing slash.")
		}
		http.Redirect(w, req, req.URL.Path+"/", http.StatusMovedPermanently)
		return
	}

	// Handle the serving of index files, if needed
	var filename string
	for _, indexfile := range indexFilenames {
		filename = filepath.Join(dirname, indexfile)
		if ac.fs.Exists(filename) {
			ac.FilePage(w, req, filename, luaDataFilename)
			return
		}
	}

	// Serve handler.lua, if found in parent directories (within server root only)
	rootAbs, err := filepath.Abs(rootdir)
	if err != nil {
		rootAbs = rootdir
	}
	ancestor, err := filepath.Abs(dirname)
	if err != nil {
		ancestor = dirname
	}
	for {
		// Stop before leaving the configured server root
		rel, relErr := filepath.Rel(rootAbs, ancestor)
		if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			break
		}
		filename = filepath.Join(ancestor, "handler.lua")
		if ac.fs.Exists(filename) {
			ac.FilePage(w, req, filename, luaDataFilename)
			return
		}
		if ancestor == rootAbs {
			break
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor { // hit filesystem root
			break
		}
		ancestor = parent
	}

	// Serve a directory listing if no index file is found
	ac.DirectoryListing(w, req, rootdir, dirname, theme)
}

// readDirConfig reads and parses the .algernon configuration file, caching the
// parsed result by filename and mtime. Returns an empty DirConfig if the file
// cannot be read or parsed.
func (ac *Config) readDirConfig(filename string) DirConfig {
	// Check mtime
	info, err := os.Stat(filename)
	if err != nil {
		return DirConfig{}
	}
	mtime := info.ModTime()

	// Return cached entry if mtime has not changed
	if ac.dirConfCache != nil {
		ac.dirConfCache.mu.RLock()
		if entry, ok := ac.dirConfCache.entries[filename]; ok && entry.mtime.Equal(mtime) {
			ac.dirConfCache.mu.RUnlock()
			return entry.config
		}
		ac.dirConfCache.mu.RUnlock()
	}

	// Read and parse
	var dirConf DirConfig
	block, err := ac.cache.Read(filename, ac.shouldCache(".algernon"))
	if err != nil {
		return dirConf
	}
	gcfg.ReadStringInto(&dirConf, string(block.Bytes()))

	// Store in cache
	if ac.dirConfCache == nil {
		ac.dirConfCache = &dirConfigCache{entries: make(map[string]dirConfigEntry)}
	}
	ac.dirConfCache.mu.Lock()
	ac.dirConfCache.entries[filename] = dirConfigEntry{config: dirConf, mtime: mtime}
	ac.dirConfCache.mu.Unlock()

	return dirConf
}
