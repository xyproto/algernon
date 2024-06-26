// Package teal supplies the Lua modules for Teal language support
// teal.lua and compat52.lua were modified as per https://github.com/yuin/gopher-lua/issues/314
package teal

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	lua "github.com/xyproto/gopher-lua"
)

const tealLoadScript = "require('compat52'); tl = require('tl'); tl.cache = {}"

var (
	//go:embed *.lua
	fs embed.FS

	// Teal files that should be preloaded when Teal is loaded
	preloadTealFilenames = []string{"bit.lua", "bit32.lua", "compat52.lua", "tl.lua"}
)

// preloadModuleFromFS loads the given Lua filename from the embedded filesystem
func preloadModuleFromFS(L *lua.LState, fname string) error {
	// Derive package name by removing file extension
	pkgname := strings.TrimSuffix(fname, filepath.Ext(fname))

	// Read the file content
	b, err := fs.ReadFile(fname)
	if err != nil {
		return fmt.Errorf("unable to read %s: %w", fname, err)
	}

	// Load the Lua module
	mod, err := L.LoadString(string(b))
	if err != nil {
		return fmt.Errorf("could not load %s: %w", fname, err)
	}

	// Get the "package" table
	pkg := L.GetField(L.Get(lua.EnvironIndex), "package")
	if pkg == lua.LNil {
		return fmt.Errorf("unable to get 'package' table")
	}

	// Get the "preload" table inside "package"
	preload := L.GetField(pkg, "preload")
	if preload == lua.LNil {
		return fmt.Errorf("unable to get 'preload' table")
	}

	L.SetField(preload, pkgname, mod)
	return nil
}

// Load makes Teal available in the Lua VM
func Load(L *lua.LState) {
	for _, fname := range preloadTealFilenames {
		if err := preloadModuleFromFS(L, fname); err != nil {
			logrus.Errorf("Failed to load Teal: %v", err)
			return
		}
	}
	if err := L.DoString(tealLoadScript); err != nil {
		logrus.Errorf("Failed to set `tl` global variable: %v", err)
	}
}
