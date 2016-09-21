package main

import (
	"github.com/bmizerany/assert"
	"github.com/yuin/gopher-lua"
	"html/template"
	"io/ioutil"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func pongoPageTest(n int, t *testing.T) {
	fs = NewFileStat(true, time.Minute*1)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	filename := "samples/pongo2/index.po2"
	luafilename := "samples/pongo2/data.lua"
	pongodata, err := ioutil.ReadFile(filename)
	assert.Equal(t, err, nil)
	cache := newFileCache(20000000, true, 64*KiB)

	luablock, err := cache.read(luafilename, shouldCache(".po2"))
	assert.Equal(t, err, nil)

	// luablock can be empty if there was an error or if the file was empty
	assert.Equal(t, luablock.HasData(), true)

	// Lua LState pool
	luapool := &lStatePool{saved: make([]*lua.LState, 0, 4)}
	defer luapool.Shutdown()

	// Pongo2+Lua mutex
	pongomutex := &sync.RWMutex{}

	// Make functions from the given Lua data available
	errChan := make(chan error)
	funcMapChan := make(chan template.FuncMap)
	go lua2funcMap(w, req, filename, luafilename, ".lua", nil, luapool, cache, pongomutex, errChan, funcMapChan)
	funcs := <-funcMapChan
	err = <-errChan
	assert.Equal(t, err, nil)

	// Trigger the error (now resolved)
	for i := 0; i < n; i++ {
		go pongoPage(w, req, filename, pongodata, funcs, cache)
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
