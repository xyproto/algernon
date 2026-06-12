package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/xyproto/unzip"
)

// algCacheEntry holds the result of one .alg extraction. ready is closed once
// the extraction finishes, so concurrent requests for the same file wait for
// the in-flight extraction instead of redoing it.
//
// targetDir is the physical extraction directory (the one to remove on
// cleanup). serveDir is what the handler hands to DirPage; it may equal
// targetDir or a single descended top-level subdirectory.
//
// refs counts in-flight callers still using serveDir. evicted is set when the
// entry is removed from the map (mtime/size mismatch); when refs reaches 0
// after eviction, targetDir is removed from disk.
type algCacheEntry struct {
	serveDir  string
	targetDir string
	mtime     time.Time
	size      int64
	err       error
	ready     chan struct{}
	refs      int
	evicted   bool
}

// algExtractionCache memoizes extracted .alg web applications by absolute
// filename. Entries are invalidated when the source .alg's mtime or size
// changes; the cached on-disk directory is reference-counted so in-flight
// requests keep serving from it until they release.
type algExtractionCache struct {
	mu       sync.Mutex
	entries  map[string]*algCacheEntry
	rootDir  string
	rootOnce sync.Once
	rootErr  error
}

func newAlgExtractionCache() *algExtractionCache {
	return &algExtractionCache{entries: make(map[string]*algCacheEntry)}
}

// ensureExtractionRoot picks /dev/shm if writable, otherwise serverTempDir,
// creates a dedicated subdirectory under it, and registers a shutdown hook to
// remove it.
func (ac *Config) ensureExtractionRoot(cache *algExtractionCache) (string, error) {
	cache.rootOnce.Do(func() {
		baseRoot := "/dev/shm"
		canary, err := os.CreateTemp(baseRoot, "algernon-alg-canary-")
		if err != nil {
			baseRoot = ac.serverTempDir
		} else {
			canary.Close()
			os.Remove(canary.Name())
		}
		root, err := os.MkdirTemp(baseRoot, "algernon-alg-")
		if err != nil {
			cache.rootErr = err
			return
		}
		cache.rootDir = root
		AtShutdown(func() {
			os.RemoveAll(root)
		})
	})
	return cache.rootDir, cache.rootErr
}

// extractAlg returns the directory containing the extracted contents of the
// given .alg archive, reusing a previous extraction when the source has not
// changed and serializing concurrent extractions of the same file. The caller
// must invoke the returned release function once the serveDir is no longer in
// use; that releases the cache's reference on the underlying directory so
// evicted extractions can be cleaned up.
func (ac *Config) extractAlg(algFilename string) (string, func(), error) {
	noop := func() {}

	absFile, err := filepath.Abs(algFilename)
	if err != nil {
		absFile = algFilename
	}
	info, err := os.Stat(absFile)
	if err != nil {
		return "", noop, err
	}
	mtime := info.ModTime()
	size := info.Size()

	cache := ac.algCache

	for {
		cache.mu.Lock()
		entry, ok := cache.entries[absFile]
		if !ok {
			pending := &algCacheEntry{
				ready: make(chan struct{}),
				mtime: mtime,
				size:  size,
			}
			cache.entries[absFile] = pending
			cache.mu.Unlock()

			serveDir, targetDir, extractErr := ac.performAlgExtraction(absFile)

			cache.mu.Lock()
			pending.serveDir = serveDir
			pending.targetDir = targetDir
			pending.err = extractErr
			if extractErr == nil {
				pending.refs = 1
			} else if cache.entries[absFile] == pending {
				// Don't keep a poisoned entry in the map; let later
				// callers try again.
				delete(cache.entries, absFile)
			}
			close(pending.ready)
			cache.mu.Unlock()

			if extractErr != nil {
				return "", noop, extractErr
			}
			return serveDir, ac.makeRelease(pending), nil
		}
		cache.mu.Unlock()

		<-entry.ready

		cache.mu.Lock()
		// Re-validate while holding the lock: entry could have been evicted
		// after ready fired, and we want to avoid TOCTOU between the equality
		// checks and the refs++.
		if entry.err == nil && !entry.evicted &&
			entry.mtime.Equal(mtime) && entry.size == size {
			entry.refs++
			cache.mu.Unlock()
			return entry.serveDir, ac.makeRelease(entry), nil
		}
		// Stale, errored, or evicted. Evict (if still current) and possibly
		// schedule cleanup of the now-unreferenced target directory, then
		// loop to either pick up a peer's new pending entry or install our
		// own.
		if cache.entries[absFile] == entry {
			delete(cache.entries, absFile)
		}
		toRemove := evictLocked(entry)
		cache.mu.Unlock()
		if toRemove != "" {
			os.RemoveAll(toRemove)
		}
	}
}

// evictLocked marks the entry as evicted and, if no callers currently hold a
// reference, hands the targetDir back to the caller for removal. Must be
// called with cache.mu held.
func evictLocked(entry *algCacheEntry) string {
	entry.evicted = true
	if entry.refs > 0 || entry.targetDir == "" {
		return ""
	}
	target := entry.targetDir
	entry.targetDir = ""
	return target
}

// makeRelease returns a release function tied to the given cache entry.
// Calling it decrements the entry's reference count; if the entry has been
// evicted and the count reaches zero, the on-disk extraction directory is
// removed.
func (ac *Config) makeRelease(entry *algCacheEntry) func() {
	cache := ac.algCache
	var once sync.Once
	return func() {
		once.Do(func() {
			cache.mu.Lock()
			entry.refs--
			var toRemove string
			if entry.refs == 0 && entry.evicted && entry.targetDir != "" {
				toRemove = entry.targetDir
				entry.targetDir = ""
			}
			cache.mu.Unlock()
			if toRemove != "" {
				os.RemoveAll(toRemove)
			}
		})
	}
}

func (ac *Config) performAlgExtraction(absFile string) (serveDir, targetDir string, err error) {
	root, err := ac.ensureExtractionRoot(ac.algCache)
	if err != nil {
		return "", "", err
	}
	target, err := os.MkdirTemp(root, fmt.Sprintf("%s-", filepath.Base(absFile)))
	if err != nil {
		return "", "", err
	}
	if err := unzip.Extract(absFile, target); err != nil {
		os.RemoveAll(target)
		return "", "", err
	}
	// If the archive contained a single top-level directory, descend into it
	// so the served root matches the original inline behaviour.
	if entries, readErr := os.ReadDir(target); readErr == nil && len(entries) == 1 && entries[0].IsDir() {
		return filepath.Join(target, entries[0].Name()), target, nil
	}
	return target, target, nil
}
