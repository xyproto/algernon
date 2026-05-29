//go:build !windows

// Package platformdep contains platform-specific constants and helpers.
package platformdep

// Mingw reports whether the runtime looks like a Mingw/Cygwin terminal.
// Always false on Unix; only ever true on Windows.
const (
	Mingw        = false
	EnableColors = true
)
