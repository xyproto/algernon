package main

import (
	"bytes"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/yuin/gopher-lua"
	"io/ioutil"
	"strings"
	"sync"
)

// fileID identifies a filename
type fileID string

const emptyFileID = fileID("")

// FileCache manages a set of bytes as a cache
type FileCache struct {
	size              uint64            // Total size of the cache
	blob              []byte            // The cache storage
	index             map[fileID]uint64 // Overview of where the data is placed in the cache
	hits              map[fileID]uint64 // Keeping track of data popularity
	offset            uint64            // The current position in the cache storage (end of data)
	rw                *sync.RWMutex     // Used for avoiding data races and other issues
	cacheWarningGiven bool              // Used to only warn once if the cache is full
	compressed        bool
}

var (
	// ErrRemoval is used if a filename that does not exist is attempted to be removed
	ErrRemoval = errors.New("Can't remove a file ID that does not exist")

	// ErrNoData is used if no data is attempted to be stored in the cache
	ErrNoData = errors.New("No data")

	// ErrAlreadyStored is used if a given filename has already been stored in the cache
	ErrAlreadyStored = errors.New("That file ID is already stored")

	// ErrLargerThanCache is used if the given data is larger than the total cache size
	ErrLargerThanCache = errors.New("Data is larger than the the total cache size")
)

func newFileCache(cacheSize uint64, compressed bool) *FileCache {
	var cache FileCache
	cache.size = cacheSize
	cache.blob = make([]byte, cacheSize) // The cache storage
	cache.index = make(map[fileID]uint64)
	cache.hits = make(map[fileID]uint64)
	cache.rw = &sync.RWMutex{}
	cache.compressed = compressed
	return &cache
}

// Normalize the filename
func (cache *FileCache) normalize(filename string) fileID {
	// If the filename begins with "./", remove it
	if strings.HasPrefix(filename, "./") {
		return fileID(filename[2:])
	}
	return fileID(filename)
}

// Remove bytes from the cache blob
// i is the position, n is the number of bytes to remove
func (cache *FileCache) removeBytes(i, n uint64) {
	cache.blob = append(cache.blob[:i], cache.blob[i+n:]...)
	// Extend to the original capacity after removing bytes
	cache.blob = cache.blob[:cap(cache.blob)]
}

// Shuffle all indices that refers to position after removedpos, offset to the left
// Also moves the end of the data offset to the left
func (cache *FileCache) shuffleIndicesLeft(removedpos, offset uint64) {
	for id, pos := range cache.index {
		if pos > removedpos {
			cache.index[id] -= offset
		}
	}
}

// Remove a data index
func (cache *FileCache) removeIndex(id fileID) {
	delete(cache.index, id)
}

// Remove data from the cache and shuffle the rest of the data to the left
// Also adjusts all index pointers to indexes larger than the current position
// Also adjusts the cache.offset
func (cache *FileCache) remove(id fileID) error {
	if !cache.hasFile(id) {
		return ErrRemoval
	}

	// Find the position and size of the given id
	pos := cache.index[id]
	size := cache.dataSize(id)

	//if cache.offset < 0 || cache.offset > cache.size {
	//	panic(fmt.Sprintln("Offset out of bounds! end of data:", cache.offset, "size of cache:", cache.size))
	//}

	cache.removeIndex(id)
	cache.shuffleIndicesLeft(pos, size)
	cache.removeBytes(pos, size)
	cache.offset -= uint64(size)

	//if cache.offset < 0 || cache.offset > cache.size {
	//	panic(fmt.Sprintln("Offset out of bounds! end of data:", cache.offset, "size of cache:", cache.size))
	//}

	return nil
}

// Find the data with the least hits, that is currently in the cache
func (cache *FileCache) leastPopular() (fileID, error) {
	// If there is no data, return an error
	if len(cache.index) == 0 {
		return emptyFileID, ErrNoData
	}

	// Loop through all the data and return the first with no cache hits
	for id := range cache.index {
		// If there are no cache hits, return this id
		foundHit := false
		for hitID := range cache.hits {
			if hitID == id {
				foundHit = true
				break
			}
		}
		if !foundHit {
			return id, nil
		}
	}

	var (
		leastHits   uint64
		leastHitsID fileID
		firstRound  = true
	)

	// Loop through all the data and find the one with the least cache hits
	for id := range cache.index {
		// If there is a cache hit, check if it is the first round
		if firstRound {
			// First candidate
			leastHits = cache.hits[id]
			leastHitsID = id
			firstRound = false
			continue
		}
		// Not the first round, compare with the least popular ID so far
		if cache.hits[id] < leastHits {
			// Found a less popular ID
			leastHits = cache.hits[id]
			leastHitsID = id
		}
	}

	// Return the one with the fewest cache hits.
	return leastHitsID, nil
}

// Store a file in the cache
// Returns true when the cache has reached the maximum (and also removed data to make space)
func (cache *FileCache) storeData(filename string, data []byte) error {
	var (
		fileSize = uint64(len(data))
		id       = cache.normalize(filename)
	)

	if cache.hasFile(id) {
		return ErrAlreadyStored
	}

	if fileSize > cache.size {
		return ErrLargerThanCache
	}

	// Warn once that the cache is now full
	if !cache.cacheWarningGiven && fileSize > cache.freeSpace() {
		log.Warn("Cache is full. You may want to increase the cache size.")
		cache.cacheWarningGiven = true
	}

	// While there is not enough space, remove the least popular data
	for fileSize > cache.freeSpace() {

		// Find the least popular data
		removeID, err := cache.leastPopular()
		if err != nil {
			return err
		}

		// Remove it
		if verboseMode {
			log.Info(fmt.Sprintf("Removing the unpopular %v from the cache (%d bytes)", removeID, cache.dataSize(removeID)))
		}

		spaceBefore := cache.freeSpace()
		err = cache.remove(removeID)
		if err != nil {
			return err
		}
		spaceAfter := cache.freeSpace()

		// Panic if there is no more free cache space after removing data
		if spaceBefore == spaceAfter {
			panic(fmt.Sprintf("Removed %v, but the free space is the same! Still %d bytes.", removeID, spaceAfter))
		}
	}

	//if cache.offset < 0 || cache.offset > cache.size {
	//	panic(fmt.Sprintln("Offset out of bounds! end of data:", cache.offset, "size of cache:", cache.size))
	//}

	if verboseMode {
		log.Info(fmt.Sprintf("Storing in cache: %v", id))
	}

	// Register the position in the data index
	cache.index[id] = cache.offset

	// Copy the contents to the cache
	var i uint64
	for i = 0; i < fileSize; i++ {
		cache.blob[cache.offset+i] = data[i]
	}

	// Move the offset to the end of the data (the next free location)
	cache.offset += uint64(fileSize)

	return nil
}

// Check if the given filename exists in the cache
func (cache *FileCache) hasFile(id fileID) bool {
	for key := range cache.index {
		if key == id {
			return true
		}
	}
	return false
}

// Find the next start of a data block, given a position
// Returns a position and true if a next position was found
func (cache *FileCache) nextData(startpos uint64) (uint64, bool) {
	// Find the end of the data (larger than startpos, but smallest diff)
	datasize := cache.size - startpos // Largest possible data size
	endpos := startpos                // Initial end of data
	found := false
	for _, pos := range cache.index {
		if pos > startpos && (pos-startpos) < datasize {
			datasize = (pos - startpos)
			endpos = pos
			found = true
		}
	}
	// endpos is the next start of a data block
	return endpos, found
}

// Find the size of a cached data block. id must exist.
func (cache *FileCache) dataSize(id fileID) uint64 {
	startpos := cache.index[id]

	// Find the next data block
	endpos, foundNext := cache.nextData(startpos)

	// Use the end of data as the end position, if no next position is found
	if !foundNext {
		endpos = cache.offset
	}

	return endpos - startpos
}

// Store a file in the cache
// Returns true if we got the data from the file, regardless of cache errors.
func (cache *FileCache) storeFile(filename string) ([]byte, bool, error) {
	// Read the file
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, false, err
	}
	// Store in cache, log a warning if the cache has filled up and needs to make space every time
	return data, true, cache.storeData(filename, data)
}

// Retrieve a file from the cache, or from disk
func (cache *FileCache) fetchData(filename string) ([]byte, error) {
	// RWMutex locks
	cache.rw.Lock()
	defer cache.rw.Unlock()

	id := cache.normalize(filename)

	// Check if the file needs to be read from disk
	fileCached := cache.hasFile(id)

	if !fileCached {
		if verboseMode {
			log.Info("Reading from disk: " + string(id))
			log.Info("Storing in cache: " + string(id))
		}
		data, gotTheData, err := cache.storeFile(string(id))
		// Cache errors are logged as warnings, and not being returned
		if err != nil {
			// Log cache errors as warnings (could be that the file is too large)
			if verboseMode {
				log.Warn(err)
			}
		}
		// Only return an error here if we were not able to read the file from disk
		if !gotTheData {
			return nil, err
		}
		// Return the data, with no errors to report
		return data, nil
	}

	// Find the start of the data
	startpos := cache.index[id]

	// Find the size of the data
	size := cache.dataSize(id)

	// Copy the data from the cache
	data := make([]byte, size)
	var i uint64
	for i = 0; i < size; i++ {
		data[i] = cache.blob[startpos+i]
	}

	// Mark a cache hit
	cache.hits[id]++

	if verboseMode {
		log.Info("Retrieving from cache: " + string(id))
	}

	// Return the data
	return data, nil
}

func (cache *FileCache) freeSpace() uint64 {
	return cache.size - cache.offset
}

// Return formatted cache statstics
func (cache *FileCache) stats() string {
	cache.rw.Lock()
	defer cache.rw.Unlock()

	var buf bytes.Buffer
	buf.WriteString("Cache stats:\n")
	buf.WriteString(fmt.Sprintf("\tTotal cache\t%d bytes\n", cache.size))
	buf.WriteString(fmt.Sprintf("\tFree cache:\t%d bytes\n", cache.freeSpace()))
	buf.WriteString(fmt.Sprintf("\tEnd of data\t%d\n", cache.offset))
	if len(cache.index) > 0 {
		buf.WriteString("\tData in cache:\n")
		for id, pos := range cache.index {
			buf.WriteString(fmt.Sprintf("\t\tid=%v\tpos=%d\tsize=%d\n", id, pos, cache.dataSize(id)))
		}
	}
	var totalHits uint64
	if len(cache.hits) > 0 {
		buf.WriteString("\tCache hits:\n")
		for id, hits := range cache.hits {
			buf.WriteString(fmt.Sprintf("\t\tid=%v\thits=%d\n", id, hits))
			totalHits += hits
		}
		buf.WriteString(fmt.Sprintf("\tTotal cache hits:\t%d", totalHits))
	}
	return buf.String()
}

// Clear the entire cache
func (cache *FileCache) clear() {
	cache.rw.Lock()
	defer cache.rw.Unlock()

	cache = newFileCache(cache.size, cache.compressed)

	// Allow one warning if the cache should fill up
	cache.cacheWarningGiven = false
}

// Export functions related to the cache. cache can be nil.
func exportCacheFunctions(L *lua.LState, cache *FileCache) {

	const disabledMessage = "Caching is disabled"
	const clearedMessage = "Cache cleared"

	// Return information about the cache use
	L.SetGlobal("CacheStats", L.NewFunction(func(L *lua.LState) int {
		if cache == nil {
			L.Push(lua.LString(disabledMessage))
			return 1 // number of results
		}
		info := cache.stats()
		// Return the string, but drop the final newline
		L.Push(lua.LString(info[:len(info)-1]))
		return 1 // number of results
	}))

	// Clear the cache
	L.SetGlobal("ClearCache", L.NewFunction(func(L *lua.LState) int {
		if cache == nil {
			L.Push(lua.LString(disabledMessage))
			return 1 // number of results
		}
		cache.clear()
		L.Push(lua.LString(clearedMessage))
		return 1 // number of results
	}))

}

// For reading files, with optional caching
func (cache *FileCache) read(filename string, cacheEnabled bool) ([]byte, error) {
	if cacheEnabled {
		// Read the file from cache (or disk, if not cached)
		return cache.fetchData(filename)
	}
	// Normalize the filename
	filename = string(cache.normalize(filename))
	if verboseMode {
		log.Info("Reading from disk: " + filename)
	}
	// RWMutex locks
	//cache.rw.Lock()
	//defer cache.rw.Unlock()
	// Read the file
	return ioutil.ReadFile(filename)
}
