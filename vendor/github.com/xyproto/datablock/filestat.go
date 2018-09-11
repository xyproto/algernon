package datablock

import (
	"os"
	"sync"
	"time"
)

// FileStat can cache calls to os.Stat. This requires that the user wants to
// assume that no files are removed from the server directory while the server
// is running, to gain some additional speed (and a tiny bit of memory use for
// the cache).
type FileStat struct {
	// If cache + mut are enabled
	useCache bool

	// Cache for checking if directories exists, if "everFile" is enabled
	dirCache map[string]bool
	dirMut   *sync.RWMutex

	// Cache for checking if files exists, if "everFile" is enabled
	exCache map[string]bool
	exMut   *sync.RWMutex

	// How often the stat cache should be cleared
	clearStatCacheDelay time.Duration
}

// NewFileStat creates a new FileStat struct, with optional caching.
// Only use the caching if it is not critical that os.Stat is always correct.
func NewFileStat(useCache bool, clearStatCacheDelay time.Duration) *FileStat {
	if !useCache {
		return &FileStat{false, nil, nil, nil, nil, clearStatCacheDelay}
	}

	dirCache := make(map[string]bool)
	dirMut := new(sync.RWMutex)

	exCache := make(map[string]bool)
	exMut := new(sync.RWMutex)

	fs := &FileStat{true, dirCache, dirMut, exCache, exMut, clearStatCacheDelay}

	// Clear the file stat cache every N seconds
	go func() {
		for {
			fs.Sleep(0)

			fs.dirMut.Lock()
			fs.dirCache = make(map[string]bool)
			fs.dirMut.Unlock()

			fs.exMut.Lock()
			fs.exCache = make(map[string]bool)
			fs.exMut.Unlock()
		}
	}()

	return fs
}

// Sleep for an entire stat cache clear cycle + optional extra sleep time
func (fs *FileStat) Sleep(extraSleep time.Duration) {
	time.Sleep(fs.clearStatCacheDelay)
}

// Normalize a filename by removing the precedeing "./".
// Useful when caching, to avoid duplicate entries.
func normalize(filename string) string {
	if filename == "" {
		return ""
	}
	// Slight optimization:
	// Avoid taking the length of the string until we know it is needed
	if filename[0] == '.' {
		if len(filename) > 2 { // Don't remove "./" if that is all there is
			if filename[1] == '/' {
				return filename[2:]
			}
		}
	}
	return filename
}

// IsDir checks if a given path is a directory
func (fs *FileStat) IsDir(path string) bool {
	if fs.useCache {
		path = normalize(path)
		// Assume this to be true
		if path == "." {
			return true
		}
		// Use the read mutex
		fs.dirMut.RLock()
		// Check the cache
		val, ok := fs.dirCache[path]
		if ok {
			fs.dirMut.RUnlock()
			return val
		}
		fs.dirMut.RUnlock()
		// Use the write mutex
		fs.dirMut.Lock()
		defer fs.dirMut.Unlock()
		// Check the filesystem
		fileInfo, err := os.Stat(path)
		if err != nil {
			// Save to cache and return
			fs.dirCache[path] = false
			return false
		}
		okDir := fileInfo.IsDir()
		// Save to cache and return
		fs.dirCache[path] = okDir
		return okDir
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileInfo.IsDir()
}

// Exists check if a given path exists
func (fs *FileStat) Exists(path string) bool {
	if fs.useCache {
		path = normalize(path)
		// Use the read mutex
		fs.exMut.RLock()
		// Check the cache
		val, ok := fs.exCache[path]
		if ok {
			fs.exMut.RUnlock()
			return val
		}
		fs.exMut.RUnlock()
		// Use the write mutex
		fs.exMut.Lock()
		defer fs.exMut.Unlock()
		// Check the filesystem
		_, err := os.Stat(path)
		// Save to cache and return
		fs.exCache[path] = err == nil
		return err == nil
	}
	_, err := os.Stat(path)
	return err == nil
}
