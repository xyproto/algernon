package cache

type ModeSetting int

const (
	// Possible cache modes
	ModeUnset       = iota // cache mode has not been set
	ModeOn                 // cache everything
	ModeDevelopment        // cache everything, except Amber, Lua, GCSS and Markdown
	ModeProduction         // cache everything, except Amber and Lua
	ModeImages             // cache images (png, jpg, gif, svg)
	ModeSmall              // only cache small files (<=64KB) // 64 * 1024
	ModeOff                // cache nothing
)

const ModeDefault = ModeOn

var (
	// Table of cache mode setting names
	ModeNames = map[ModeSetting]string{
		ModeUnset:       "unset",
		ModeOn:          "On",
		ModeDevelopment: "Development",
		ModeProduction:  "Production",
		ModeImages:      "Images",
		ModeSmall:       "Small",
		ModeOff:         "Off",
	}
)

// newCacheModeSetting creates a CacheModeSetting based on a variety of string options, like "on" and "off".
func NewModeSetting(ModeString string) ModeSetting {
	switch ModeString {
	case "everything", "all", "on", "1", "enabled", "yes", "enable": // Cache everything.
		return ModeOn
	case "production", "prod": // Cache everything, except: Amber and Lua.
		return ModeProduction
	case "images", "image": // Cache images (png, jpg, gif, svg).
		return ModeImages
	case "small", "64k", "64KB": // Cache only small files (<=64KB), but not Amber and Lua
		return ModeSmall
	case "off", "disabled", "0", "no", "disable": // Disable caching entirely.
		return ModeOff
	case "dev", "default", "unset": // Cache everything, except: Amber, Lua, GCSS and Markdown.
		fallthrough
	default:
		return ModeDefault
	}
}

// Return the name of the cache mode setting, if set
func (cms ModeSetting) String() string {
	for k, v := range ModeNames {
		if k == cms {
			return v
		}
	}
	// Could not find the name
	return ModeNames[ModeUnset]
}
