// Package cachemode provides ways to deal with different cache modes
package cachemode

// Setting represents a cache mode setting
type Setting int

// Possible cache modes
const (
	Unset       = iota // cache mode has not been set
	On                 // cache everything
	Development        // cache everything, except Amber, Lua, GCSS and Markdown
	Production         // cache everything, except Amber and Lua
	Images             // cache images (png, jpg, gif, svg)
	Small              // only cache small files (<=64KB) // 64 * 1024
	Off                // cache nothing
	Default     = On
)

// Names is a map of cache mode setting string representations
var Names = map[Setting]string{
	Unset:       "unset",
	On:          "On",
	Development: "Development",
	Production:  "Production",
	Images:      "Images",
	Small:       "Small",
	Off:         "Off",
}

// New creates a CacheModeSetting based on a variety of string options, like "on" and "off".
func New(mode string) Setting {
	switch mode {
	case "everything", "all", "on", "1", "enabled", "yes", "enable": // Cache everything.
		return On
	case "production", "prod": // Cache everything, except: Amber and Lua.
		return Production
	case "images", "image": // Cache images (png, jpg, gif, svg).
		return Images
	case "small", "64k", "64KB": // Cache only small files (<=64KB), but not Amber and Lua
		return Small
	case "off", "disabled", "0", "no", "disable": // Disable caching entirely.
		return Off
	case "dev", "default", "unset": // Cache everything, except: Amber, Lua, GCSS and Markdown.
		fallthrough
	default:
		return Default
	}
}

// String returns the name of the cache mode setting, if set
func (cms Setting) String() string {
	for k, v := range Names {
		if k == cms {
			return v
		}
	}
	// Could not find the name
	return Names[Unset]
}
