//go:build windows

package platformdep

import (
	"strings"

	"github.com/xyproto/env/v2"
)

var (
	// Probably using Mingw, or something like it
	Mingw = strings.HasPrefix(env.Str("TERM"), "xterm")

	EnableColors = Mingw
)
