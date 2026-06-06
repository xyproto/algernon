package engine

import (
	"path/filepath"
	"runtime"
	"testing"
)

// Resolution of the optional globals.lua location, see issue #103.
func TestGlobalsLuaPath(t *testing.T) {
	cases := []struct {
		name    string
		dirOrFn string
		single  bool
		want    string
	}{
		{"directory mode", "/srv/app", false, filepath.Join("/srv/app", "globals.lua")},
		{"single file mode", "/srv/app/index.lua", true, filepath.Join("/srv/app", "globals.lua")},
		{"empty input", "", false, ""},
	}
	for _, c := range cases {
		if runtime.GOOS == "windows" {
			t.Skip("paths in this test assume POSIX separators")
		}
		if got := globalsLuaPath(c.dirOrFn, c.single); got != c.want {
			t.Errorf("%s: globalsLuaPath(%q, %v) = %q, want %q", c.name, c.dirOrFn, c.single, got, c.want)
		}
	}
}
