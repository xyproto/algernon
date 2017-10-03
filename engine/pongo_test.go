package engine

import (
	"html/template"
	"io/ioutil"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bmizerany/assert"
	"github.com/xyproto/algernon/lua/pool"
	"github.com/xyproto/algernon/utils"
	"github.com/xyproto/datablock"
)

func pongoPageTest(n int, t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	filename := "testdata/index.po2"
	luafilename := "testdata/data.lua"
	pongodata, err := ioutil.ReadFile(filename)
	assert.Equal(t, err, nil)

	ac, err := New("Algernon 123", "Just a test")
	assert.Equal(t, err, nil)

	// Use a FileStat cache with different settings
	ac.SetFileStatCache(datablock.NewFileStat(true, time.Minute*1))

	ac.cache = datablock.NewFileCache(20000000, true, 64*utils.KiB, true, 0)

	luablock, err := ac.cache.Read(luafilename, ac.shouldCache(".po2"))
	assert.Equal(t, err, nil)

	// luablock can be empty if there was an error or if the file was empty
	assert.Equal(t, luablock.HasData(), true)

	// Lua LState pool
	ac.luapool = pool.New()
	defer ac.luapool.Shutdown()

	// Make functions from the given Lua data available
	errChan := make(chan error)
	funcMapChan := make(chan template.FuncMap)
	go ac.Lua2funcMap(w, req, filename, luafilename, ".lua", errChan, funcMapChan)
	funcs := <-funcMapChan
	err = <-errChan
	assert.Equal(t, err, nil)

	// Trigger the error (now resolved)
	for i := 0; i < n; i++ {
		go ac.PongoPage(w, req, filename, pongodata, funcs)
	}
}

func TestPongoPage(t *testing.T) {
	pongoPageTest(1, t)
}

//func TestConcurrentPongoPage1(t *testing.T) {
//	pongoPageTest(10, t)
//}
//
//func TestConcurrentPongoPage2(t *testing.T) {
//	for i := 0; i < 10; i++ {
//		go pongoPageTest(1, t)
//	}
//}
//
//func TestConcurrentPongoPage3(t *testing.T) {
//	for i := 0; i < 10; i++ {
//		go pongoPageTest(10, t)
//	}
//}
//
//func TestConcurrentPongoPage4(t *testing.T) {
//	for i := 0; i < 1000; i++ {
//		go pongoPageTest(1000, t)
//	}
//}
