//go:build !windows

package platformdep

// Per-directory filenames recognized by Algernon.
const (
	DirConfFilename = ".algernon"
	IgnoreFilename  = ".ignore"
	HistoryFilename = ".algernon_history"
)
