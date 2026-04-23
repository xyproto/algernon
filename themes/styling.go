package themes

import (
	"embed"
	"io/fs"
	"path"
	"strings"
)

// assetsFS holds the built-in theme stylesheets. Every assets/<name>.css becomes a key in builtinThemes.
//
//go:embed assets/*.css
var assetsFS embed.FS

var (
	// builtinCodeStyles is a map of the themes names corresponding to chroma styles
	// See the Chroma Style Gallery for more styles: https://xyproto.github.io/splash/docs/
	// "default" currently points to "material"
	builtinCodeStyles = map[string]string{"material": "lovelace", "gray": "manni", "dark": "dracula", "redbox": "fruity", "bw": "bw", "wing": "tango", "neon": "api", "light": "monokailight", "werc": "dracula", "setconf": "monokailight"}

	// builtinThemes is a map over the available built-in CSS themes. Corresponds with the font themes below.
	// "default" and "gray" are equal. "default" should never be used directly, but is here as a safeguard.
	builtinThemes = loadBuiltinThemes()
)

// loadBuiltinThemes reads every assets/*.css entry from the embedded FS into a name -> body map.
// A missing or unreadable entry panics at init time.
func loadBuiltinThemes() map[string][]byte {
	themes := make(map[string][]byte)
	entries, err := fs.ReadDir(assetsFS, "assets")
	if err != nil {
		panic("themes: could not read embedded assets: " + err.Error())
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".css") || name == "mui.css" {
			// mui.css is only used via MaterialHead(), not as a theme
			continue
		}
		body, err := fs.ReadFile(assetsFS, path.Join("assets", name))
		if err != nil {
			panic("themes: could not read " + name + ": " + err.Error())
		}
		themes[strings.TrimSuffix(name, ".css")] = body
	}
	return themes
}
