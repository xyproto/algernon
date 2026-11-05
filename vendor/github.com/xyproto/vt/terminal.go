package vt

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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
	saveCursorAttrs    = "\0337"
	restoreCursorAttrs = "\0338"
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
	beginSyncUpdate    = "\033[?2026h"
	endSyncUpdate      = "\033[?2026l"
)

// Terminal sequences for turning on and off the alternate screen, bracketed
// paste, focus events and mouse reporting. Writing these is harmless on
// terminals that don't understand them.
const (
	EnterAltScreenSeq        = "\033[?1049h"
	ExitAltScreenSeq         = "\033[?1049l"
	EnableBracketedPasteSeq  = "\033[?2004h"
	DisableBracketedPasteSeq = "\033[?2004l"
	EnableFocusEventsSeq     = "\033[?1004h"
	DisableFocusEventsSeq    = "\033[?1004l"
	EnableMouseSeq           = "\033[?1000h\033[?1002h\033[?1006h"
	DisableMouseSeq          = "\033[?1006l\033[?1003l\033[?1002l\033[?1000l"
)

// Key strings that ReadKey returns for bracketed paste and focus events
const (
	KeyPasteStartString = "\033[200~"
	KeyPasteEndString   = "\033[201~"
	KeyFocusInString    = "\033[I"
	KeyFocusOutString   = "\033[O"
)

// NoColor is the escape sequence for resetting all color attributes
const NoColor string = "\033[0m"

// Stop returns the escape sequence for resetting all color attributes,
// or "" when NO_COLOR is set.
func Stop() string {
	return envResetSeq
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

// underDvtm is true if running inside dvtm
var underDvtm = env.Has("DVTM")

// underAbduco is true if running inside abduco
var underAbduco = env.Has("ABDUCO")

// underZutty is true if running inside the Zutty terminal emulator
var underZutty = env.Has("ZUTTY_VERSION")

// syncUpdateSupported is false for terminal emulators that are known to not
// implement DEC private mode 2026 (synchronized output). Zutty logs a warning
// for every attempt to set or reset that mode.
var syncUpdateSupported = !underZutty

// multiplexed is true when running inside any known terminal multiplexer
var multiplexed = underTMUX || underScreen || underDvtm || underAbduco

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

// Zutty returns true when running inside the Zutty terminal emulator
func Zutty() bool {
	return underZutty
}

// SyncUpdateSupported returns true when the terminal emulator is expected to
// understand DEC private mode 2026 (synchronized output)
func SyncUpdateSupported() bool {
	return syncUpdateSupported
}

// beginSyncUpdateSeq returns the begin synchronized update escape sequence,
// or an empty string if the terminal emulator does not support it
func beginSyncUpdateSeq() string {
	if syncUpdateSupported {
		return beginSyncUpdate
	}
	return ""
}

// endSyncUpdateSeq returns the end synchronized update escape sequence,
// or an empty string if the terminal emulator does not support it
func endSyncUpdateSeq() string {
	if syncUpdateSupported {
		return endSyncUpdate
	}
	return ""
}

// xtermLike returns true when $TERM looks like an xterm-class emulator
func XtermLike() bool {
	return xtermLike
}

// HasAltArrows returns true when the terminal is expected to emit
// alt-arrow key sequences (ESC [1;3A/B/C/D for alt-up/down/right/left).
// These are only generated by xterm-class emulators.
func HasAltArrows() bool {
	return xtermLike
}

// HasCtrlArrows returns true when the terminal is expected to emit
// ctrl-arrow key sequences (ESC [1;5A/B/C/D for ctrl-up/down/right/left).
// These are only generated by xterm-class emulators.
func HasCtrlArrows() bool {
	return xtermLike
}

// HasShiftArrows returns true when the terminal is expected to emit
// shift-arrow and shift-home/end key sequences
// (ESC [1;2A/B/C/D/H/F for shift-up/down/right/left/home/end).
// These are only generated by xterm-class emulators.
func HasShiftArrows() bool {
	return xtermLike
}

// Has256Colors returns true when the terminal supports 256-color mode
// (i.e. $TERM contains "256color" or is "xterm-kitty").
func Has256Colors() bool {
	term := env.Str("TERM")
	return strings.Contains(term, "256color") || term == "xterm-kitty"
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

// EnterAltScreen switches to the alternate screen, so that the current
// contents of the terminal can be restored with ExitAltScreen
func EnterAltScreen() {
	fmt.Print(EnterAltScreenSeq)
}

// ExitAltScreen switches back from the alternate screen
func ExitAltScreen() {
	fmt.Print(ExitAltScreenSeq)
}

// EnableBracketedPaste makes the terminal surround pasted text with
// KeyPasteStartString and KeyPasteEndString. Use TTY.ReadPaste to read the
// pasted text after ReadKey has returned KeyPasteStartString.
func EnableBracketedPaste() {
	fmt.Print(EnableBracketedPasteSeq)
}

// DisableBracketedPaste turns off bracketed paste
func DisableBracketedPaste() {
	fmt.Print(DisableBracketedPasteSeq)
}

// EnableFocusEvents makes ReadKey return KeyFocusInString and
// KeyFocusOutString when the terminal gains and loses focus
func EnableFocusEvents() {
	fmt.Print(EnableFocusEventsSeq)
}

// DisableFocusEvents turns off focus events
func DisableFocusEvents() {
	fmt.Print(DisableFocusEventsSeq)
}

// EnableMouse turns on reporting of mouse button presses, releases, drags and
// the scroll wheel. Use ParseMouseEvent on the keys that ReadKey returns.
func EnableMouse() {
	fmt.Print(EnableMouseSeq)
}

// DisableMouse turns off mouse reporting
func DisableMouse() {
	fmt.Print(DisableMouseSeq)
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
// The terminal response format is "rgb:RRRR/GGGG/BBBB" where each component
// is a hex value of 1 to 4 digits.
func GetBackgroundColor(tty *TTY) (float64, float64, float64, error) {
	// First try the escape code used by e.g. alacritty
	result, err := queryTerminal(tty, "\033]11;?\a")
	if err != nil {
		return 0, 0, 0, err
	}
	if !strings.Contains(result, "]11;rgb:") {
		// Then try the escape code used by e.g. gnome-terminal
		more, err := queryTerminal(tty, "\033]10;?\a\033]11;?\a")
		if err != nil {
			return 0, 0, 0, err
		}
		result += more
	}
	if r, g, b, ok := parseOSCColor(result, "]11;rgb:"); ok {
		return r, g, b, nil
	}
	return 0, 0, 0, fmt.Errorf("could not read rgb value from terminal emulator, got: %q", result)
}

// queryTimeout is how long queryTerminal waits for a terminal that does not
// answer the primary device attributes request
const queryTimeout = 2 * time.Second

// queryTerminal sends the given query to the terminal, followed by a primary
// device attributes request, which practically all terminals answer. Since
// terminals answer in order, the reply to the query, if any, has arrived once
// the device attributes reply has arrived. This avoids both giving up too
// early on slow connections and leaving a late reply in the input.
func queryTerminal(tty *TTY, query string) (string, error) {
	if err := tty.WriteString(query + "\033[c"); err != nil {
		return "", err
	}
	var sb strings.Builder
	deadline := time.Now().Add(queryTimeout)
	for time.Now().Before(deadline) {
		s, err := tty.ReadStringKeepTiming()
		if err != nil {
			if tty.reader != nil {
				break // the input of a mock TTY has ended
			}
			continue
		}
		sb.WriteString(s)
		if hasDeviceAttributesReply(sb.String()) {
			break
		}
	}
	return sb.String(), nil
}

// hasDeviceAttributesReply returns true if s contains a reply to a primary
// device attributes request, ESC [ ? digits and semicolons c
func hasDeviceAttributesReply(s string) bool {
	for {
		_, after, found := strings.Cut(s, "\033[?")
		if !found {
			return false
		}
		i := 0
		for i < len(after) && (after[i] >= '0' && after[i] <= '9' || after[i] == ';') {
			i++
		}
		if i < len(after) && after[i] == 'c' {
			return true
		}
		s = after
	}
}

// parseOSCColor finds the color that follows the given prefix in an OSC color
// reply such as "\033]11;rgb:RRRR/GGGG/BBBB\a", and returns its components
// as values in [0.0, 1.0]
func parseOSCColor(reply, prefix string) (float64, float64, float64, bool) {
	_, after, ok := strings.Cut(reply, prefix)
	if !ok {
		return 0, 0, 0, false
	}
	// Trim anything after the three hex components (e.g. trailing BEL, ST, newline)
	if i := strings.IndexFunc(after, func(r rune) bool {
		return r < 0x20 || r == '\033' || r == '\a' || r == '\\'
	}); i >= 0 {
		after = after[:i]
	}
	parts := strings.Split(after, "/")
	if len(parts) != 3 {
		return 0, 0, 0, false
	}
	var values [3]float64
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) < 1 || len(part) > 4 {
			return 0, 0, 0, false
		}
		v, err := strconv.ParseUint(part, 16, 16)
		if err != nil {
			return 0, 0, 0, false
		}
		values[i] = float64(v) / float64(uint64(1)<<(4*len(part))-1)
	}
	return values[0], values[1], values[2], true
}

// BeginSyncUpdate sends the terminal's begin synchronized update escape sequence
func BeginSyncUpdate() {
	fmt.Print(beginSyncUpdateSeq())
}

// EndSyncUpdate sends the terminal's end synchronized update escape sequence
func EndSyncUpdate() {
	fmt.Print(endSyncUpdateSeq())
}
