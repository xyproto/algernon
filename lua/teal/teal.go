// Package teal supplies the Lua modules for Teal language support
// teal.lua and compat52.lua were modified as per https://github.com/yuin/gopher-lua/issues/314
package teal

import (
	"embed"
	"fmt"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	lua "github.com/xyproto/gopher-lua"
)

var (
	//go:embed *.lua
	fs embed.FS

	// Teal files that should be preloaded when Teal is loaded
	preloadTealFilenames = []string{"bit.lua", "bit32.lua", "compat52.lua", "tl.lua"}
)

const tealLoadScript = "require('compat52'); tl = require('tl'); tl.cache = {}"

// Load makes Teal available in the Lua VM
func Load(L *lua.LState) {
	for _, fname := range preloadTealFilenames {
		if err := preloadModuleFromFS(L, fname); err != nil {
			log.Errorf("Failed to load Teal: %v", err)
			return
		}
	}
	if err := L.DoString(tealLoadScript); err != nil {
		log.Errorf("Failed to set `tl` global variable: %v", err)
	}
}

// preloadModuleFromFS loads the given Lua filename from the embedded filesystem
func preloadModuleFromFS(L *lua.LState, fname string) error {
	pkgname := fname[:len(fname)-len(filepath.Ext(fname))]
	b, err := fs.ReadFile(fname)
	if err != nil {
		return fmt.Errorf("Unable to read %s: %w", fname, err)
	}
	mod, err := L.LoadString(string(b))
	if err != nil {
		return fmt.Errorf("Could not load %s: %w", fname, err)
	}
	pkg := L.GetField(L.Get(lua.EnvironIndex), "package")
	preload := L.GetField(pkg, "preload")
	L.SetField(preload, pkgname, mod)
	return nil
}
