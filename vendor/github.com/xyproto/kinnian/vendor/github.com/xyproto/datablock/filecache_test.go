package datablock

import (
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/bmizerany/assert"
	"github.com/xyproto/cookie"
)

func TestID(t *testing.T) {
	cache := NewFileCache(1, false, 0, true)
	_ = cache.normalize("test.filename")
}

func TestHas(t *testing.T) {
	cache := NewFileCache(1, false, 0, true)
	cache.cacheWarningGiven = true // Silence warning when the cache is full
	readmeID := cache.normalize("README.md")
	if cache.hasFile(readmeID) {
		t.Error("Should not have any file in cache right now")
	}
}

func TestStore(t *testing.T) {
	cache := NewFileCache(100000, false, 0, true)
	data, err := ioutil.ReadFile("README.md")
	if err != nil {
		t.Error(err)
	}
	if _, err := cache.storeData("README.md", data); err != nil {
		t.Errorf("Could not store README.md in the cache: %s", err)
	}
	readmeID := cache.normalize("README.md")
	if !cache.hasFile(readmeID) {
		t.Error("Should have a file in cache right now")
	}
}

func TestLoad(t *testing.T) {
	cache := NewFileCache(100000, false, 0, true)
	readmeData, err := ioutil.ReadFile("README.md")
	if err != nil {
		t.Error(err)
	}
	_, err = cache.storeData("README.md", readmeData)
	if err != nil {
		t.Errorf("Could not store README.md in the cache: %s", err)
	}
	licenseData, err := ioutil.ReadFile("LICENSE")
	if err != nil {
		t.Error(err)
	}
	_, err = cache.storeData("LICENSE", licenseData)
	if err != nil {
		t.Errorf("Could not store LICENSE in the cache: %s", err)
	}
	readmeDataBlock2, err := cache.fetchAndCache("README.md")
	if err != nil {
		t.Errorf("Could not read file from cache: %s", err)
	}
	if len(readmeData) != readmeDataBlock2.Length() {
		t.Errorf("Different length of data in cache: %d vs %d", len(readmeData), readmeDataBlock2.Length())
	}
	readmeData2 := readmeDataBlock2.MustData()
	for i := range readmeData {
		if readmeData[i] != readmeData2[i] {
			t.Error("Data from cache differs!")
		}
	}
	licenseDataBlock2, err := cache.fetchAndCache("LICENSE")
	if err != nil {
		t.Errorf("Could not read file from cache: %s", err)
	}
	if len(licenseData) != licenseDataBlock2.Length() {
		t.Errorf("Different length of data in cache: %d vs %d", len(licenseData), licenseDataBlock2.Length())
	}
	licenseData2 := licenseDataBlock2.MustData()
	for i := range licenseData {
		if licenseData[i] != licenseData2[i] {
			t.Error("Data from cache differs!")
		}
	}
}

func TestOverflow(t *testing.T) {
	cache := NewFileCache(100000, false, 0, true)
	data, err := ioutil.ReadFile("README.md")
	assert.Equal(t, err, nil)
	// Repeatedly store a file until the cache is full
	for err == nil {
		_, err = cache.storeData("README.md", data)
	}
	if err == nil {
		t.Error("Cache should be full, but is not.")
	}
}

// Check if two byte slices differs.
// Uses the second argument for the length.
func differs(a, b []byte) bool {
	for i, x := range b {
		if x != a[i] {
			return true
		}
	}
	return false
}

func TestRemovalAddition(t *testing.T) {
	cache := NewFileCache(8, false, 0, true)
	cache.cacheWarningGiven = true // Silence warning when the cache is full
	adata := []byte{1, 1, 1, 1}
	bdata := []byte{2, 2, 2, 2}
	cdata := []byte{3, 3}
	ddata := []byte{4}
	if cache.offset != 0 {
		t.Error("Cache offset is supposed to be 0, but is", cache.offset)
	}
	// Fill cache
	cache.storeData("a", adata)
	if differs(cache.blob, []byte{1, 1, 1, 1, 0, 0, 0, 0}) {
		t.Error("Cache is supposed to be 1 1 1 1 0 0 0 0, but is", cache.blob)
	}
	cache.storeData("b", bdata)
	if differs(cache.blob, []byte{1, 1, 1, 1, 2, 2, 2, 2}) {
		t.Error("Cache is supposed to be 1 1 1 1 2 2 2 2, but is", cache.blob)
	}
	if cache.offset != 8 {
		t.Error("Cache offset is supposed to be 8, but is", cache.offset)
	}
	// Make b the most popular data
	cache.fetchAndCache("b")
	// Remove a and store c at the end
	cache.storeData("c", cdata)
	if differs(cache.blob, []byte{2, 2, 2, 2, 3, 3}) {
		t.Error("Cache is supposed to be 2 2 2 2 3 3 x x, but is", cache.blob)
	}
	if cache.offset != 6 {
		t.Error("Cache offset is supposed to be 6, but is", cache.offset)
	}
	// Then two of d, to fill the cache
	cache.storeData("d", ddata)
	if cache.offset != 7 {
		t.Error("Cache offset is supposed to be 7, but is", cache.offset)
	}
	// Storing d again should not make a difference
	cache.storeData("d", ddata)
	if differs(cache.blob, []byte{2, 2, 2, 2, 3, 3, 4}) {
		t.Error("Cache is supposed to be 2 2 2 2 3 3 4 x, but is", cache.blob)
	}
	cache.remove(cache.normalize("b"))
	if differs(cache.blob, []byte{3, 3, 4}) {
		t.Error("Cache is supposed to be 3 3 4 x x x x x, but is", cache.blob)
	}
	cache.storeData("b", bdata)
	cache.storeData("a", adata)
	if differs(cache.blob, []byte{2, 2, 2, 2, 1, 1, 1, 1}) {
		t.Error("Cache is supposed to be 2, 2, 2, 2, 1, 1, 1, 1, but is", cache.blob)
	}
}

func TestRandomStoreGet(t *testing.T) {
	const cacheSize = 5
	cache := NewFileCache(5, false, 0, true)
	cache.cacheWarningGiven = true // Silence warning when the cache is full
	filenames := []string{"a", "b", "c"}
	datasets := [][]byte{{0, 1, 2}, {3, 4, 5, 6}, {7}}
	for i := 0; i < 100; i++ {
		switch rand.Intn(4) {
		case 0, 1: // Add data to the cache
			// Select one at random
			n := rand.Intn(3)
			filename := filenames[n]
			data := datasets[n]
			id := cache.normalize(filename)
			if !cache.hasFile(id) {
				//fmt.Printf("adding %s (%v)\n", filename, id)
				_, err := cache.storeData(filename, data)
				if err != nil {
					t.Errorf("Could not add %s: %s\n", filename, err)
					//} else {
					//	fmt.Printf("added %s (%v)\n", filename, id)
				}
				//fmt.Println(cache.Stats())
			}
		case 2: // Add, get and remove data
			filename := cookie.RandomHumanFriendlyString(rand.Intn(20))
			data := []byte(cookie.RandomString(rand.Intn(cacheSize + 1)))
			_, err := cache.storeData(filename, data)
			if err != nil {
				if err == ErrAlreadyStored {
					// If that filename is already stored, just continue
					continue
				}
				t.Fatal(err)
			}
			datablock2, err := cache.fetchAndCache(filename)
			if err != nil {
				t.Fatal(err)
			}
			id := cache.normalize(filename)
			err = cache.remove(id)
			if err != nil {
				t.Fatal(err)
			}
			if len(data) != datablock2.Length() {
				t.Fatal("WRONG LENGTH!")
			}
			data2 := datablock2.MustData()
			for i := 0; i < len(data); i++ {
				if data[i] != data2[i] {
					t.Fatal("WRONG BYTE!")
				}
			}
		default: // Read data from the cache
			// Select one at random
			n := rand.Intn(3)
			filename := filenames[n]
			data := datasets[n]
			//id := cache.normalize(filename)
			//fmt.Printf("retrieving %s (%v)\n", filename, id)
			retDataBlock, err := cache.fetchAndCache(filename)
			if err == nil {
				//fmt.Printf("retrieved %s (%v)\n", filename, id)
				//fmt.Println(cache.Stats())
				if retDataBlock.Length() != len(data) {
					t.Errorf("Wrong length of data: %d vs %d\n", retDataBlock.Length(), len(data))
				}
				retData := retDataBlock.MustData()
				for x := 0; x < len(data); x++ {
					if retData[x] != data[x] {
						t.Error("Wrong contents in cache!")
					}
				}
			}
		}
	}
}

func TestCompression(t *testing.T) {
	cache := NewFileCache(100000, true, 0, true)
	assert.Equal(t, true, cache.IsEmpty())
	readmeData, err := ioutil.ReadFile("README.md")
	assert.Equal(t, err, nil)
	compressedREADMEblock, err := cache.storeData("README.md", readmeData)
	if err != nil {
		t.Errorf("Could not store README.md in the cache: %s", err)
	}
	assert.NotEqual(t, 0, compressedREADMEblock.Length())
	licenseData, err := ioutil.ReadFile("LICENSE")
	assert.Equal(t, err, nil)
	compressedLICENSEblock, err := cache.storeData("LICENSE", licenseData)
	if err != nil {
		t.Errorf("Could not store LICENSE in the cache: %s", err)
	}
	assert.NotEqual(t, 0, compressedLICENSEblock.Length())
	readmeDataBlock2, err := cache.fetchAndCache("README.md")
	if err != nil {
		t.Errorf("Could not read file from cache: %s", err)
	}
	if len(readmeData) != len(readmeDataBlock2.MustData()) {
		t.Errorf("Different length of data in cache: %d vs %d", len(readmeData), len(readmeDataBlock2.MustData()))
	}
	readmeData2 := readmeDataBlock2.MustData()
	for i := range readmeData {
		if readmeData[i] != readmeData2[i] {
			t.Error("Data from cache differs!")
		}
	}
	licenseDataBlock2, err := cache.fetchAndCache("LICENSE")
	if err != nil {
		t.Errorf("Could not read file from cache: %s", err)
	}
	if len(licenseData) != len(licenseDataBlock2.MustData()) {
		t.Errorf("Different length of data in cache: %d vs %d", len(licenseData), len(licenseDataBlock2.MustData()))
	}
	licenseData2 := licenseDataBlock2.MustData()
	for i := range licenseData {
		if licenseData[i] != licenseData2[i] {
			t.Error("Data from cache differs!")
		}
	}
}

func TestClear(t *testing.T) {
	cache := NewFileCache(1000, true, 0, true)
	assert.Equal(t, cache.size, cache.freeSpace())
	readmeData, err := ioutil.ReadFile("README.md")
	assert.Equal(t, err, nil)
	compressedREADMEblock, err := cache.storeData("README.md", readmeData)
	if err != nil {
		t.Errorf("Could not store README.md in the cache: %s", err)
	}
	assert.NotEqual(t, 0, compressedREADMEblock.Length())
	assert.NotEqual(t, cache.size, cache.freeSpace())
	cache.Clear()
	assert.Equal(t, cache.size, cache.freeSpace())
}

func TestStats(t *testing.T) {
	cache := NewFileCache(123, true, 0, true)
	assert.Equal(t, strings.Contains(cache.Stats(), "123 bytes"), true)
}

func TestRead(t *testing.T) {
	// New file cache
	cache := NewFileCache(2000, true, 0, true)
	// Open a tempfile
	tmpfile, err := ioutil.TempFile("", "example")
	assert.Equal(t, err, nil)
	// Write data to tempfile
	content := []byte("SHOW ME WHAT YOU GOT")
	_, err = tmpfile.Write(content)
	assert.Equal(t, err, nil)
	// Close tempfile
	err = tmpfile.Close()
	assert.Equal(t, err, nil)
	// Read tempfile into cache, from disk
	readmeData, err := cache.Read(tmpfile.Name(), true)
	assert.Equal(t, err, nil)
	// Check if data is equal
	assert.Equal(t, content, readmeData.MustData())
	// Remove tempfile
	os.Remove(tmpfile.Name())
	// Read tempfile from cache, it's gone from disk
	readmeData2, err := cache.Read(tmpfile.Name(), true)
	assert.Equal(t, err, nil)
	// Check if strings are equal
	assert.Equal(t, string(content), readmeData2.String())
	// Try (and fail) to read data from disk, not from cache
	_, err = cache.Read(tmpfile.Name(), false)
	assert.NotEqual(t, err, nil) // Supposed to be an error
}
