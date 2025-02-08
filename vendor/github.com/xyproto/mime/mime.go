// Package mime helps retrieving mimetypes given extensions.
// This is an alternative to the "mime" package, and has fallbacks for the most common types.
package mime

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
)

var fallback = map[string]string{
	"7z":      "application/octet-stream",
	"arj":     "application/octet-stream",
	"avi":     "video/x-msvideo",
	"bmp":     "image/bmp",
	"bz2":     "application/x-bzip2",
	"com":     "application/octet-stream",
	"css":     "text/css",
	"csv":     "text/csv",
	"doc":     "application/msword",
	"docx":    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"elf":     "application/octet-stream",
	"eot":     "application/vnd.ms-fontobject",
	"exe":     "application/vnd.microsoft.portable-executable",
	"flac":    "audio/flac",
	"flv":     "video/x-flv",
	"gif":     "image/gif",
	"gz":      "application/x-gzip",
	"html":    "text/html",
	"ico":     "image/x-icon",
	"jpeg":    "image/jpg",
	"jpg":     "image/jpg",
	"js":      "application/javascript",
	"json":    "application/javascript",
	"lz":      "application/octet-stream",
	"md":      "text/markdown",
	"mkv":     "video/x-matroska",
	"mov":     "video/quicktime",
	"mp3":     "audio/mpeg",
	"mp4":     "video/mp4",
	"ogg":     "video/ogg",
	"otf":     "font/otf",
	"pdf":     "application/pdf",
	"png":     "image/png",
	"ppt":     "application/vnd.ms-powerpoint",
	"pptx":    "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	"rar":     "application/octet-stream",
	"rss":     "application/rss+xml",
	"svg":     "image/svg+xml",
	"tar":     "application/x-tar",
	"tar.bz":  "application/x-bzip-compressed-tar",
	"tar.bz2": "application/x-bzip-compressed-tar",
	"tar.gz":  "application/x-gzip-compressed-tar",
	"tar.xz":  "application/x-xz-compressed-tar",
	"tbz":     "application/x-bzip-compressed-tar",
	"tbz2":    "application/x-bzip-compressed-tar",
	"tgz":     "application/x-gzip-compressed-tar",
	"torrent": "application/x-bittorrent",
	"ttf":     "font/ttf",
	"txt":     "text/plain",
	"txz":     "application/x-xz-compressed-tar",
	"wasm":    "application/wasm",
	"wav":     "audio/wav",
	"webm":    "video/webm",
	"webp":    "image/webp",
	"woff":    "font/woff",
	"woff2":   "font/woff2",
	"xls":     "application/vnd.ms-excel",
	"xlsx":    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"xml":     "text/xml",
	"xz":      "application/x-xz",
	"zip":     "application/zip",
}

// Reader caches the contents of a mime info text file
type Reader struct {
	filename  string
	utf8      bool
	mimetypes map[string]string
	mu        sync.RWMutex
}

// New creates a new Reader. The filename is a list of mimetypes and extensions.
// If utf8 is true, "; charset=utf-8" will be added when setting http headers.
func New(filename string, utf8 bool) *Reader {
	return &Reader{filename, utf8, nil, sync.RWMutex{}}
}

// Read a mimetype text file. Return a hash map from ext to mimetype.
func readMimetypes(filename string) (map[string]string, error) {
	mimetypes := make(map[string]string)
	// Read the mimetype file
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	// For each line, store extensions and mimetypes in the hash map
	for _, line := range bytes.Split(data, []byte("\n")) {
		fields := bytes.Fields(line)
		if len(fields) > 1 {
			for _, ext := range fields[1:] {
				mimetypes[string(ext)] = string(fields[0])
			}
		}
	}
	return mimetypes, nil
}

// Get returns the mimetype, or an empty string if no mimetype or mimetype source is found
func (mr *Reader) Get(ext string) string {
	if len(ext) == 0 {
		return ""
	}
	if ext[0] == '.' {
		ext = ext[1:]
	}

	// Try to lookup the extension in the map
	mr.mu.RLock()
	if mr.mimetypes != nil {
		if mime, ok := mr.mimetypes[ext]; ok {
			mr.mu.RUnlock()
			return mime
		}
	}
	mr.mu.RUnlock()

	// If mimetypes is nil or no value was found, initialize the map
	mr.mu.Lock()
	defer mr.mu.Unlock()
	if mr.mimetypes == nil {
		var err error
		mr.mimetypes, err = readMimetypes(mr.filename)
		if err != nil {
			if mime, ok := fallback[ext]; ok {
				return mime
			}
			return ""
		}
	}

	// Find and return the mimetype, if possible

	if mime, ok := mr.mimetypes[ext]; ok {
		return mime
	}
	if mime, ok := fallback[ext]; ok {
		return mime
	}
	return ""
}

// SetHeader sets the Content-Type for a given ResponseWriter and filename extension
func (mr *Reader) SetHeader(w http.ResponseWriter, ext string) {
	mimestring := mr.Get(ext)
	if mimestring == "" {
		// Default mime type
		mimestring = "application/octet-stream"
	}
	if mr.utf8 && !strings.Contains(mimestring, "wasm") && !strings.Contains(mimestring, "image") {
		mimestring += "; charset=utf-8"
	}
	w.Header().Set("Content-Type", mimestring)
}
