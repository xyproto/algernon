package env

import "strings"

// WaylandSession returns true of XDG_SESSION_TYPE is "wayland" or if
// DESKTOP_SESSION contains "wayland".
func WaylandSession() bool {
	return Str("XDG_SESSION_TYPE") == "wayland" || Has("WAYLAND_DISPLAY") || strings.Contains(Str("DESKTOP_SESSION"), "wayland")
}

// XSession returns true if DISPLAY is set.
// X being available is not neccessarily in opposition to Wayland running.
func XSession() bool {
	return Has("DISPLAY")
}

// OnlyXSession returns true if DISPLAY is set and WaylandSession() is false.
func OnlyXSession() bool {
	return Has("DISPLAY") && !WaylandSession()
}

// XOrWaylandSession returns true if DISPLAY is set or WaylandSession() returns true.
func XOrWaylandSession() bool {
	return Has("DISPLAY") || WaylandSession()
}
