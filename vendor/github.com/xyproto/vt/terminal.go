package vt

import (
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/xyproto/env/v2"
)

const (
	cursorHome         = "\033[H"
	cursorHomeTemplate = "\033[%d;%dH"
	cursorUp           = "\033[A"
	cursorDown         = "\033[B"
	cursorForward      = "\033[C"
	cursorBackward     = "\033[D"
	saveCursor         = "\033[s"
	restoreCursor      = "\033[u"
	saveCursorAttrs    = "\033[7"
	restoreCursorAttrs = "\033[8"
	resetDevice        = "\033c"
	eraseScreen        = "\033[2J"
	eraseEndOfLine     = "\033[K"
	eraseStartOfLine   = "\033[1K"
	eraseLine          = "\033[2K"
	eraseDown          = "\033[J"
	eraseUp            = "\033[1J"
	enableLineWrap     = "\033[?7h"
	disableLineWrap    = "\033[?7l"
	showCursor         = "\033[?25h"
	hideCursor         = "\033[?25l"
	echoOff            = "\033[12h"
	attributeTemplate  = "\033[%sm"
)

// NoColor is the escape sequence for resetting all color attributes
const NoColor string = "\033[0m"

var maybeNoColor *string

// Stop returns the escape sequence for resetting all color attributes
func Stop() string {
	if maybeNoColor != nil {
		return *maybeNoColor
	}
	s := NoColor
	maybeNoColor = &s
	return s
}

// writeAllToStdout writes the given byte slice to stdout, retrying on partial writes
func writeAllToStdout(data []byte) bool {
	for len(data) > 0 {
		n, err := os.Stdout.Write(data)
		if err != nil || n <= 0 {
			return false
		}
		data = data[n:]
	}
	return true
}

// SetXY moves the cursor to the given position (0,0 is top left)
func SetXY(x, y uint) {
	fmt.Printf(cursorHomeTemplate, y+1, x+1)
}

// Home moves the cursor to the top-left corner
func Home() {
	fmt.Print(cursorHome)
}

// Reset sends the terminal reset sequence
func Reset() {
	fmt.Print(resetDevice)
}

// Clear erases the entire screen
func Clear() {
	fmt.Print(eraseScreen)
}

// SetNoColor resets all color attributes
func SetNoColor() {
	fmt.Print(NoColor)
}

// underTMUX is true if running inside TMUX
var underTMUX = env.Has("TMUX")

// underScreen is true if running inside GNU Screen
var underScreen = env.Has("STY")

// underZellij is true if running inside Zellij
var underZellij = env.Has("ZELLIJ")

// underDvtm is true if running inside dvtm
var underDvtm = env.Has("DVTM")

// underAbduco is true if running inside abduco
var underAbduco = env.Has("ABDUCO")

// multiplexed is true when running inside any known terminal multiplexer
var multiplexed = underTMUX || underScreen || underZellij || underDvtm || underAbduco

// xtermLike is true when $TERM looks like an xterm-class emulator
var xtermLike = strings.HasPrefix(env.Str("TERM"), "xterm")

// safeReset is true when it is safe to send \033c (RIS) and \033[12h (SRM).
// These are skipped under multiplexers, on the Linux console, and on
// non-xterm consoles where the behaviour is undefined or destructive.
var safeReset = xtermLike && !multiplexed

// Multiplexed returns true when running inside a terminal multiplexer
func Multiplexed() bool {
	return multiplexed
}

// xtermLike returns true when $TERM looks like an xterm-class emulator
func XtermLike() bool {
	return xtermLike
}

// Init initializes the terminal for full-screen canvas use
func Init() {
	initTerminal()
	if safeReset {
		Reset()   // \033c (RIS): only safe on xterm-class terminals outside multiplexers
		EchoOff() // \033[12h (SRM): ditto
	}
	Clear()
	ShowCursor(false)
	SetLineWrap(false)
}

// Close restores the terminal and clears the screen.
// Use CloseKeepContent to keep the canvas content visible.
func Close() {
	SetLineWrap(true)
	ShowCursor(true)
	Clear()
	Home()
}

// CloseKeepContent restores the terminal but leaves the canvas content visible
func CloseKeepContent() {
	SetLineWrap(true)
	ShowCursor(true)
	Home()
}

// EchoOff disables terminal echo
func EchoOff() {
	if echoOffHelper() {
		fmt.Print(echoOff)
	}
}

// SetLineWrap enables or disables line wrapping
func SetLineWrap(enable bool) {
	if enable {
		fmt.Print(enableLineWrap)
	} else {
		fmt.Print(disableLineWrap)
	}
}

// ShowCursor shows or hides the terminal cursor
func ShowCursor(enable bool) {
	showCursorHelper(enable)
	if enable {
		fmt.Print(showCursor)
	} else {
		fmt.Print(hideCursor)
	}
}

// GetBackgroundColor queries the terminal for its background color.
// Returns normalized RGB values in [0.0, 1.0], or an error.
func GetBackgroundColor(tty *TTY) (float64, float64, float64, error) {
	// First try the escape code used by ie. alacritty
	if err := tty.WriteString("\033]11;?\a"); err != nil {
		return 0, 0, 0, err
	}
	result, err := tty.ReadString()
	if err != nil {
		return 0, 0, 0, err
	}
	if !strings.Contains(result, "rgb:") {
		// Then try the escape code used by ie. gnome-terminal
		if err := tty.WriteString("\033]10;?\a\033]11;?\a"); err != nil {
			return 0, 0, 0, err
		}
		result, err = tty.ReadString()
		if err != nil {
			return 0, 0, 0, err
		}
	}
	if _, after, ok := strings.Cut(result, "rgb:"); ok {
		rgb := after
		if strings.Count(rgb, "/") == 2 {
			parts := strings.SplitN(rgb, "/", 3)
			if len(parts) == 3 {
				r := new(big.Int)
				r.SetString(parts[0], 16)
				g := new(big.Int)
				g.SetString(parts[1], 16)
				b := new(big.Int)
				b.SetString(parts[2], 16)
				return float64(r.Int64() / 65535.0), float64(g.Int64() / 65535.0), float64(b.Int64() / 65535.0), nil
			}
		}
	}

	return 0, 0, 0, fmt.Errorf("could not read rgb value from terminal emulator, got: %q", result)
}
