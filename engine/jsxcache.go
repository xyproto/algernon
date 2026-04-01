package engine

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/cachemode"
)

// bundleCacheEntry holds bundled output alongside the source file's modification time.
type bundleCacheEntry struct {
	modTime time.Time
	data    []byte
}

// bundleCache is an in-memory cache for esbuild-bundled JS/JSX files,
// keyed by absolute file path. Results are invalidated when the file's
// modification time changes.
type bundleCache struct {
	mu      sync.RWMutex
	entries map[string]bundleCacheEntry
	hits    map[string]uint64
}

func newBundleCache() *bundleCache {
	return &bundleCache{entries: make(map[string]bundleCacheEntry), hits: make(map[string]uint64)}
}

// Clear clears all entries from the bundle cache.
func (bc *bundleCache) Clear() {
	bc.mu.Lock()
	bc.entries = make(map[string]bundleCacheEntry)
	bc.hits = make(map[string]uint64)
	bc.mu.Unlock()
}

// needsBundling reports whether the JS/JSX source requires full bundling,
// i.e. it contains ES module import statements or CommonJS require() calls.
func needsBundling(data []byte) bool {
	return bytes.Contains(data, []byte("import ")) ||
		bytes.Contains(data, []byte("import(")) ||
		bytes.Contains(data, []byte("require("))
}

// minified) and caches the result in memory. Subsequent calls return the cached
// output as long as the file's modification time has not changed.
//
// If srcData is non-nil it is passed directly to esbuild via stdin, avoiding a
// second disk read when the caller has already loaded the source. The working
// directory is always set to the file's parent directory so that esbuild can
// resolve node_modules relative to the source file.
func (ac *Config) bundleFile(filename string, srcData []byte) ([]byte, error) {
	info, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}
	modTime := info.ModTime()

	useCache := !ac.noCache && ac.cacheMode != cachemode.Off

	bc := ac.bundleCache
	if useCache {
		bc.mu.RLock()
		entry, ok := bc.entries[filename]
		bc.mu.RUnlock()
		if ok && entry.modTime.Equal(modTime) {
			bc.mu.Lock()
			bc.hits[filename]++
			bc.mu.Unlock()
			return entry.data, nil
		}
	}

	dir := filepath.Dir(filename)
	opts := api.BuildOptions{
		Bundle:            true,
		Platform:          api.PlatformBrowser,
		Format:            api.FormatIIFE,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		Charset:           api.CharsetUTF8,
		Write:             false,
		AbsWorkingDir:     dir,
		LogLevel:          api.LogLevelSilent,
	}
	if srcData != nil {
		// Choose the loader based on file extension so JSX syntax is handled.
		loader := api.LoaderJS
		if strings.HasSuffix(strings.ToLower(filename), ".jsx") {
			loader = api.LoaderJSX
		}
		contents := string(srcData)
		if ac.autoRefresh {
			contents = injectRefreshRegistrations(contents, filepath.Base(filename))
		}
		opts.Stdin = &api.StdinOptions{
			Contents:   contents,
			ResolveDir: dir,
			Sourcefile: filename,
			Loader:     loader,
		}
	} else {
		opts.EntryPoints = []string{filename}
	}
	if ac.autoRefresh {
		opts.Plugins = []api.Plugin{reactRefreshPlugin()}
	}

	result := api.Build(opts)

	if len(result.Errors) > 0 {
		msgs := make([]string, len(result.Errors))
		for i, e := range result.Errors {
			msgs[i] = e.Text
		}
		return nil, fmt.Errorf("bundle %s: %s", filepath.Base(filename), strings.Join(msgs, "; "))
	}

	if len(result.OutputFiles) == 0 {
		return nil, fmt.Errorf("bundle %s: no output produced", filepath.Base(filename))
	}

	data := result.OutputFiles[0].Contents

	if useCache {
		bc.mu.Lock()
		if ac.cacheMaxEntitySize == 0 || uint64(len(data)) <= ac.cacheMaxEntitySize {
			if ac.bundleCacheMaxMemory == 0 || bc.BytesUsed()+uint64(len(data)) <= ac.bundleCacheMaxMemory {
				bc.entries[filename] = bundleCacheEntry{modTime: modTime, data: data}
				bc.hits[filename] = 0
				logrus.Debugf("bundled and cached %s (%d bytes)", filepath.Base(filename), len(data))
			} else {
				for bc.BytesUsed()+uint64(len(data)) > ac.bundleCacheMaxMemory && bc.evictLocked() {
				}
				if bc.BytesUsed()+uint64(len(data)) <= ac.bundleCacheMaxMemory {
					bc.entries[filename] = bundleCacheEntry{modTime: modTime, data: data}
					bc.hits[filename] = 0
					logrus.Debugf("bundled and cached %s (%d bytes)", filepath.Base(filename), len(data))
				}
			}
		}
		bc.mu.Unlock()
	}

	return data, nil
}

// BytesUsed returns the total bytes used by all entries in the bundle cache.
func (bc *bundleCache) BytesUsed() uint64 {
	var total uint64
	for _, entry := range bc.entries {
		total += uint64(len(entry.data))
	}
	return total
}

// evictLocked removes the least-popular entry. Must be called with bc.mu held.
func (bc *bundleCache) evictLocked() bool {
	if len(bc.entries) == 0 {
		return false
	}

	var targetKey string
	var minHits uint64 = ^uint64(0)
	var maxSize uint64

	for key, entry := range bc.entries {
		hits := bc.hits[key]
		size := uint64(len(entry.data))
		if hits < minHits || (hits == minHits && size > maxSize) {
			targetKey = key
			minHits = hits
			maxSize = size
		}
	}

	if targetKey != "" {
		delete(bc.entries, targetKey)
		delete(bc.hits, targetKey)
		return true
	}
	return false
}

// EvictLargest removes the least-popular entry from the bundle cache.
func (bc *bundleCache) EvictLargest() bool {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	return bc.evictLocked()
}
