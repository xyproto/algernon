package term

import (
	"github.com/nsf/termbox-go"
)

// Only a few selected colors are included here, for providing
// a coherent visual appearance for command line applications.
// For more colors, import and use termbox directly.
const (
	Black     = termbox.ColorBlack
	Red       = termbox.ColorRed
	Green     = termbox.ColorGreen
	Yellow    = termbox.ColorYellow
	Blue      = termbox.ColorBlue
	Magenta   = termbox.ColorMagenta
	Cyan      = termbox.ColorCyan
	White     = termbox.ColorWhite
	Bold      = termbox.AttrBold
	Underline = termbox.AttrUnderline
	Reverse   = termbox.AttrReverse
)
