// Package teal supplies the Lua modules for Teal language support
// teal.lua and compat52.lua were modified as per https://github.com/yuin/gopher-lua/issues/314
package teal

import (
	"embed"
	"fmt"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/yuin/gopher-lua"
)

//go:embed *.lua
var fs embed.FS

// Load makes Teal available in the Lua VM.
func Load(L *lua.LState) {
	if err := preloadModuleFromFS(L, "bit.lua"); err != nil {
		log.Errorf("Failed to load Teal: %v", err)
		return
	}
	if err := preloadModuleFromFS(L, "bit32.lua"); err != nil {
		log.Errorf("Failed to load Teal: %v", err)
		return
	}
	if err := preloadModuleFromFS(L, "compat52.lua"); err != nil {
		log.Errorf("Failed to load Teal: %v", err)
		return
	}
	if err := preloadModuleFromFS(L, "tl.lua"); err != nil {
		log.Errorf("Failed to load Teal: %v", err)
		return
	}
	if err := L.DoString("require('compat52'); tl = require('tl'); tl.cache = {}"); err != nil {
		log.Errorf("Failed to set `tl` global variable: %v", err)
	}
}

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
