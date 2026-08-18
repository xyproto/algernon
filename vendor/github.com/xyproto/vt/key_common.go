package vt

import (
	"strconv"
	"unicode"
	"unicode/utf8"
)

// Key codes returned by TTY.Key() and TTY.KeyCode() for special keys.
// Arrow keys and navigation keys are assigned codes above 127 to avoid
// collision with ASCII control characters and printable characters.
const (
	KeyPageDown = 250 // Page Down
	KeyPageUp   = 251 // Page Up
	KeyLeft     = 252 // Left Arrow
	KeyUp       = 253 // Up Arrow
	KeyRight    = 254 // Right Arrow
	KeyDown     = 255 // Down Arrow

	KeyShiftTab = 273 // Shift-Tab / Backtab
	KeyF1       = 274 // F1
	KeyF2       = 275 // F2
	KeyF3       = 276 // F3
	KeyF4       = 277 // F4
	KeyDelete   = 278 // Delete / Forward-Delete

	KeyF5  = 279 // F5
	KeyF6  = 280 // F6
	KeyF7  = 281 // F7
	KeyF8  = 282 // F8
	KeyF9  = 283 // F9
	KeyF10 = 284 // F10
	KeyF11 = 285 // F11
	KeyF12 = 286 // F12

	KeyInsert        = 256 // Insert
	KeyShiftInsert   = 257 // Shift-Insert
	KeyCtrlInsert    = 258 // Ctrl-Insert
	KeyAltUp         = 259 // Alt-Up
	KeyAltDown       = 260 // Alt-Down
	KeyAltRight      = 261 // Alt-Right
	KeyAltLeft       = 262 // Alt-Left
	KeyCtrlUp        = 263 // Ctrl-Up
	KeyCtrlDown      = 264 // Ctrl-Down
	KeyCtrlRight     = 265 // Ctrl-Right
	KeyCtrlLeft      = 266 // Ctrl-Left
	KeyShiftUp       = 267 // Shift-Up
	KeyShiftDown     = 268 // Shift-Down
	KeyShiftRight    = 269 // Shift-Right
	KeyShiftLeft     = 270 // Shift-Left
	KeyShiftHome     = 271 // Shift-Home
	KeyShiftEnd      = 272 // Shift-End
	KeyCtrlHome      = 287 // Ctrl-Home
	KeyCtrlEnd       = 288 // Ctrl-End
	KeyAltHome       = 289 // Alt-Home
	KeyAltEnd        = 290 // Alt-End
	KeyCtrlPageUp    = 291 // Ctrl-Page Up
	KeyCtrlPageDown  = 292 // Ctrl-Page Down
	KeyShiftPageUp   = 293 // Shift-Page Up
	KeyShiftPageDown = 294 // Shift-Page Down
	KeyCtrlDelete    = 295 // Ctrl-Delete
	KeyShiftDelete   = 296 // Shift-Delete
	KeyAltReturn     = 297 // Alt-Return / Alt-Enter
	KeyShiftReturn   = 298 // Shift-Return / Shift-Enter (only reported when the terminal supports the kitty keyboard protocol or xterm modifyOtherKeys=2)
)

// String representations returned by ReadKey for modifier+Return combos.
const (
	KeyAltReturnString   = "alt⏎"
	KeyShiftReturnString = "shift⏎"
)

// Terminal sequences that ask the terminal to start, and stop, reporting
// modified Return / Enter and similar key combinations. Writing these is
// harmless on terminals that don't understand them — the bytes are silently
// ignored.
//
//   - \x1b[>1u / \x1b[<u: kitty keyboard protocol (kitty, foot, wezterm, ghostty, recent alacritty)
//   - \x1b[>4;2m / \x1b[>4m: xterm modifyOtherKeys=2 (xterm, vte/gnome-terminal, konsole, urxvt, mintty)
const (
	EnableShiftReturnSeq  = "\x1b[>1u\x1b[>4;2m"
	DisableShiftReturnSeq = "\x1b[<u\x1b[>4m"
)

// Key codes for 3-byte sequences (arrows, Home, End, F1-F4, Shift-Tab)
var keyCodeLookup = map[[3]byte]int{
	{27, 91, 65}:  KeyUp,       // Up Arrow
	{27, 91, 66}:  KeyDown,     // Down Arrow
	{27, 91, 67}:  KeyRight,    // Right Arrow
	{27, 91, 68}:  KeyLeft,     // Left Arrow
	{27, 79, 65}:  KeyUp,       // Up Arrow    (SS3, application cursor keys)
	{27, 79, 66}:  KeyDown,     // Down Arrow  (SS3)
	{27, 79, 67}:  KeyRight,    // Right Arrow (SS3)
	{27, 79, 68}:  KeyLeft,     // Left Arrow  (SS3)
	{27, 91, 'H'}: 1,           // Home (mapped to Ctrl-A)
	{27, 91, 'F'}: 5,           // End (mapped to Ctrl-E)
	{27, 79, 'H'}: 1,           // Home (SS3)
	{27, 79, 'F'}: 5,           // End (SS3)
	{27, 91, 90}:  KeyShiftTab, // Shift-Tab / Backtab (ESC [Z)
	{27, 79, 80}:  KeyF1,       // F1  (ESC O P)
	{27, 79, 81}:  KeyF2,       // F2  (ESC O Q)
	{27, 79, 82}:  KeyF3,       // F3  (ESC O R)
	{27, 79, 83}:  KeyF4,       // F4  (ESC O S)
	// rxvt / urxvt report modified arrows with lowercase finals
	{27, 91, 'a'}: KeyShiftUp,    // Shift-Up    (ESC [a)
	{27, 91, 'b'}: KeyShiftDown,  // Shift-Down  (ESC [b)
	{27, 91, 'c'}: KeyShiftRight, // Shift-Right (ESC [c)
	{27, 91, 'd'}: KeyShiftLeft,  // Shift-Left  (ESC [d)
	{27, 79, 'a'}: KeyCtrlUp,     // Ctrl-Up     (ESC Oa)
	{27, 79, 'b'}: KeyCtrlDown,   // Ctrl-Down   (ESC Ob)
	{27, 79, 'c'}: KeyCtrlRight,  // Ctrl-Right  (ESC Oc)
	{27, 79, 'd'}: KeyCtrlLeft,   // Ctrl-Left   (ESC Od)
}

// Key codes for 4-byte sequences (Page Up, Page Down, Home, End, Delete)
var pageNavLookup = map[[4]byte]int{
	{27, 91, 49, 126}: 1,           // Home (ESC [1~)
	{27, 91, 51, 126}: KeyDelete,   // Delete / Forward-Delete (ESC [3~)
	{27, 91, 52, 126}: 5,           // End (ESC [4~)
	{27, 91, 53, 126}: KeyPageUp,   // Page Up
	{27, 91, 54, 126}: KeyPageDown, // Page Down
	{27, 91, 55, 126}: 1,           // Home (ESC [7~)
	{27, 91, 56, 126}: 5,           // End (ESC [8~)
	{27, 91, 50, 126}: KeyInsert,   // Insert (ESC [2~)
	{27, 91, 91, 65}:  KeyF1,       // F1 (ESC [[A, Linux console)
	{27, 91, 91, 66}:  KeyF2,       // F2 (ESC [[B, Linux console)
	{27, 91, 91, 67}:  KeyF3,       // F3 (ESC [[C, Linux console)
	{27, 91, 91, 68}:  KeyF4,       // F4 (ESC [[D, Linux console)
	{27, 91, 91, 69}:  KeyF5,       // F5 (ESC [[E, Linux console)
	// rxvt / urxvt encode the modifier in the final byte: '$' is Shift, '^' is Ctrl
	{27, 91, 49, '$'}: KeyShiftHome,     // Shift-Home   (ESC [1$)
	{27, 91, 50, '$'}: KeyShiftInsert,   // Shift-Insert (ESC [2$)
	{27, 91, 51, '$'}: KeyShiftDelete,   // Shift-Delete (ESC [3$)
	{27, 91, 53, '$'}: KeyShiftPageUp,   // Shift-PgUp   (ESC [5$)
	{27, 91, 54, '$'}: KeyShiftPageDown, // Shift-PgDn   (ESC [6$)
	{27, 91, 55, '$'}: KeyShiftHome,     // Shift-Home   (ESC [7$)
	{27, 91, 56, '$'}: KeyShiftEnd,      // Shift-End    (ESC [8$)
	{27, 91, 49, '^'}: KeyCtrlHome,      // Ctrl-Home    (ESC [1^)
	{27, 91, 50, '^'}: KeyCtrlInsert,    // Ctrl-Insert  (ESC [2^)
	{27, 91, 51, '^'}: KeyCtrlDelete,    // Ctrl-Delete  (ESC [3^)
	{27, 91, 53, '^'}: KeyCtrlPageUp,    // Ctrl-PgUp    (ESC [5^)
	{27, 91, 54, '^'}: KeyCtrlPageDown,  // Ctrl-PgDn    (ESC [6^)
	{27, 91, 55, '^'}: KeyCtrlHome,      // Ctrl-Home    (ESC [7^)
	{27, 91, 56, '^'}: KeyCtrlEnd,       // Ctrl-End     (ESC [8^)
}

// Key codes for 5-byte sequences (F5-F12)
var fKeyLookup = map[[5]byte]int{
	{27, 91, 49, 49, 126}: KeyF1,  // F1  (ESC [11~, rxvt/PuTTY)
	{27, 91, 49, 50, 126}: KeyF2,  // F2  (ESC [12~, rxvt/PuTTY)
	{27, 91, 49, 51, 126}: KeyF3,  // F3  (ESC [13~, rxvt/PuTTY)
	{27, 91, 49, 52, 126}: KeyF4,  // F4  (ESC [14~, rxvt/PuTTY)
	{27, 91, 49, 53, 126}: KeyF5,  // F5  (ESC [15~)
	{27, 91, 49, 55, 126}: KeyF6,  // F6  (ESC [17~)
	{27, 91, 49, 56, 126}: KeyF7,  // F7  (ESC [18~)
	{27, 91, 49, 57, 126}: KeyF8,  // F8  (ESC [19~)
	{27, 91, 50, 48, 126}: KeyF9,  // F9  (ESC [20~)
	{27, 91, 50, 49, 126}: KeyF10, // F10 (ESC [21~)
	{27, 91, 50, 51, 126}: KeyF11, // F11 (ESC [23~)
	{27, 91, 50, 52, 126}: KeyF12, // F12 (ESC [24~)
}

// Key codes for 6-byte modifier-key sequences (CSI with modifier parameter)
var modKeyLookup = map[[6]byte]int{
	{27, 91, 50, 59, 53, 126}: KeyCtrlInsert,    // Ctrl-Insert   (ESC [2;5~)
	{27, 91, 49, 59, 51, 65}:  KeyAltUp,         // Alt-Up        (ESC [1;3A)
	{27, 91, 49, 59, 51, 66}:  KeyAltDown,       // Alt-Down      (ESC [1;3B)
	{27, 91, 49, 59, 51, 67}:  KeyAltRight,      // Alt-Right     (ESC [1;3C)
	{27, 91, 49, 59, 51, 68}:  KeyAltLeft,       // Alt-Left      (ESC [1;3D)
	{27, 91, 49, 59, 53, 65}:  KeyCtrlUp,        // Ctrl-Up       (ESC [1;5A)
	{27, 91, 49, 59, 53, 66}:  KeyCtrlDown,      // Ctrl-Down     (ESC [1;5B)
	{27, 91, 49, 59, 53, 67}:  KeyCtrlRight,     // Ctrl-Right    (ESC [1;5C)
	{27, 91, 49, 59, 53, 68}:  KeyCtrlLeft,      // Ctrl-Left     (ESC [1;5D)
	{27, 91, 49, 59, 50, 65}:  KeyShiftUp,       // Shift-Up      (ESC [1;2A)
	{27, 91, 49, 59, 50, 66}:  KeyShiftDown,     // Shift-Down    (ESC [1;2B)
	{27, 91, 49, 59, 50, 67}:  KeyShiftRight,    // Shift-Right   (ESC [1;2C)
	{27, 91, 49, 59, 50, 68}:  KeyShiftLeft,     // Shift-Left    (ESC [1;2D)
	{27, 91, 49, 59, 50, 72}:  KeyShiftHome,     // Shift-Home    (ESC [1;2H)
	{27, 91, 49, 59, 50, 70}:  KeyShiftEnd,      // Shift-End     (ESC [1;2F)
	{27, 91, 49, 59, 53, 72}:  KeyCtrlHome,      // Ctrl-Home     (ESC [1;5H)
	{27, 91, 49, 59, 53, 70}:  KeyCtrlEnd,       // Ctrl-End      (ESC [1;5F)
	{27, 91, 49, 59, 51, 72}:  KeyAltHome,       // Alt-Home      (ESC [1;3H)
	{27, 91, 49, 59, 51, 70}:  KeyAltEnd,        // Alt-End       (ESC [1;3F)
	{27, 91, 53, 59, 53, 126}: KeyCtrlPageUp,    // Ctrl-PgUp     (ESC [5;5~)
	{27, 91, 54, 59, 53, 126}: KeyCtrlPageDown,  // Ctrl-PgDn     (ESC [6;5~)
	{27, 91, 53, 59, 50, 126}: KeyShiftPageUp,   // Shift-PgUp    (ESC [5;2~)
	{27, 91, 54, 59, 50, 126}: KeyShiftPageDown, // Shift-PgDn   (ESC [6;2~)
	{27, 91, 51, 59, 53, 126}: KeyCtrlDelete,    // Ctrl-Delete   (ESC [3;5~)
	{27, 91, 51, 59, 50, 126}: KeyShiftDelete,   // Shift-Delete  (ESC [3;2~)
	{27, 91, 50, 59, 50, 126}: KeyShiftInsert,   // Shift-Insert  (ESC [2;2~)
}

// String representations for 3-byte sequences
var keyStringLookup = map[[3]byte]string{
	{27, 91, 65}:  "↑",       // Up Arrow
	{27, 91, 66}:  "↓",       // Down Arrow
	{27, 91, 67}:  "→",       // Right Arrow
	{27, 91, 68}:  "←",       // Left Arrow
	{27, 79, 65}:  "↑",       // Up Arrow    (SS3, application cursor keys)
	{27, 79, 66}:  "↓",       // Down Arrow  (SS3)
	{27, 79, 67}:  "→",       // Right Arrow (SS3)
	{27, 79, 68}:  "←",       // Left Arrow  (SS3)
	{27, 91, 'H'}: "⇱",       // Home
	{27, 91, 'F'}: "⇲",       // End
	{27, 79, 'H'}: "⇱",       // Home (SS3 sequence)
	{27, 79, 'F'}: "⇲",       // End (SS3 sequence)
	{27, 91, 90}:  "backtab", // Shift-Tab / Backtab (ESC [Z)
	{27, 79, 80}:  "F1",      // F1  (ESC O P)
	{27, 79, 81}:  "F2",      // F2  (ESC O Q)
	{27, 79, 82}:  "F3",      // F3  (ESC O R)
	{27, 79, 83}:  "F4",      // F4  (ESC O S)
	// rxvt / urxvt report modified arrows with lowercase finals
	{27, 91, 'a'}: "shift↑", // Shift-Up    (ESC [a)
	{27, 91, 'b'}: "shift↓", // Shift-Down  (ESC [b)
	{27, 91, 'c'}: "shift→", // Shift-Right (ESC [c)
	{27, 91, 'd'}: "shift←", // Shift-Left  (ESC [d)
	{27, 79, 'a'}: "ctrl↑",  // Ctrl-Up     (ESC Oa)
	{27, 79, 'b'}: "ctrl↓",  // Ctrl-Down   (ESC Ob)
	{27, 79, 'c'}: "ctrl→",  // Ctrl-Right  (ESC Oc)
	{27, 79, 'd'}: "ctrl←",  // Ctrl-Left   (ESC Od)
}

// String representations for 4-byte sequences
var pageStringLookup = map[[4]byte]string{
	{27, 91, 49, 126}: "⇱",  // Home
	{27, 91, 51, 126}: "⌦",  // Delete / Forward-Delete
	{27, 91, 52, 126}: "⇲",  // End
	{27, 91, 53, 126}: "⇞",  // Page Up
	{27, 91, 54, 126}: "⇟",  // Page Down
	{27, 91, 55, 126}: "⇱",  // Home
	{27, 91, 56, 126}: "⇲",  // End
	{27, 91, 91, 65}:  "F1", // F1 (ESC [[A, Linux console)
	{27, 91, 91, 66}:  "F2", // F2 (ESC [[B, Linux console)
	{27, 91, 91, 67}:  "F3", // F3 (ESC [[C, Linux console)
	{27, 91, 91, 68}:  "F4", // F4 (ESC [[D, Linux console)
	{27, 91, 91, 69}:  "F5", // F5 (ESC [[E, Linux console)
	{27, 91, 50, 126}: "⎀",  // Insert (ESC [2~)
	// rxvt / urxvt encode the modifier in the final byte: '$' is Shift, '^' is Ctrl
	{27, 91, 49, '$'}: "shift⇱", // Shift-Home   (ESC [1$)
	{27, 91, 50, '$'}: "shift⎀", // Shift-Insert (ESC [2$)
	{27, 91, 51, '$'}: "shift⌦", // Shift-Delete (ESC [3$)
	{27, 91, 53, '$'}: "shift⇞", // Shift-PgUp   (ESC [5$)
	{27, 91, 54, '$'}: "shift⇟", // Shift-PgDn   (ESC [6$)
	{27, 91, 55, '$'}: "shift⇱", // Shift-Home   (ESC [7$)
	{27, 91, 56, '$'}: "shift⇲", // Shift-End    (ESC [8$)
	{27, 91, 49, '^'}: "ctrl⇱",  // Ctrl-Home    (ESC [1^)
	{27, 91, 50, '^'}: "ctrl⎀",  // Ctrl-Insert  (ESC [2^)
	{27, 91, 51, '^'}: "ctrl⌦",  // Ctrl-Delete  (ESC [3^)
	{27, 91, 53, '^'}: "ctrl⇞",  // Ctrl-PgUp    (ESC [5^)
	{27, 91, 54, '^'}: "ctrl⇟",  // Ctrl-PgDn    (ESC [6^)
	{27, 91, 55, '^'}: "ctrl⇱",  // Ctrl-Home    (ESC [7^)
	{27, 91, 56, '^'}: "ctrl⇲",  // Ctrl-End     (ESC [8^)
}

// String representations for 5-byte sequences (F5-F12)
var fKeyStringLookup = map[[5]byte]string{
	{27, 91, 49, 49, 126}: "F1",  // F1  (ESC [11~, rxvt/PuTTY)
	{27, 91, 49, 50, 126}: "F2",  // F2  (ESC [12~, rxvt/PuTTY)
	{27, 91, 49, 51, 126}: "F3",  // F3  (ESC [13~, rxvt/PuTTY)
	{27, 91, 49, 52, 126}: "F4",  // F4  (ESC [14~, rxvt/PuTTY)
	{27, 91, 49, 53, 126}: "F5",  // F5  (ESC [15~)
	{27, 91, 49, 55, 126}: "F6",  // F6  (ESC [17~)
	{27, 91, 49, 56, 126}: "F7",  // F7  (ESC [18~)
	{27, 91, 49, 57, 126}: "F8",  // F8  (ESC [19~)
	{27, 91, 50, 48, 126}: "F9",  // F9  (ESC [20~)
	{27, 91, 50, 49, 126}: "F10", // F10 (ESC [21~)
	{27, 91, 50, 51, 126}: "F11", // F11 (ESC [23~)
	{27, 91, 50, 52, 126}: "F12", // F12 (ESC [24~)
}

// String representations for 6-byte modifier-key sequences (CSI with modifier parameter)
var modKeyStringLookup = map[[6]byte]string{
	{27, 91, 50, 59, 53, 126}: "ctrl⎀",  // Ctrl-Insert
	{27, 91, 49, 59, 51, 65}:  "alt↑",   // Alt-Up
	{27, 91, 49, 59, 51, 66}:  "alt↓",   // Alt-Down
	{27, 91, 49, 59, 51, 67}:  "alt→",   // Alt-Right
	{27, 91, 49, 59, 51, 68}:  "alt←",   // Alt-Left
	{27, 91, 49, 59, 53, 65}:  "ctrl↑",  // Ctrl-Up
	{27, 91, 49, 59, 53, 66}:  "ctrl↓",  // Ctrl-Down
	{27, 91, 49, 59, 53, 67}:  "ctrl→",  // Ctrl-Right
	{27, 91, 49, 59, 53, 68}:  "ctrl←",  // Ctrl-Left
	{27, 91, 49, 59, 50, 65}:  "shift↑", // Shift-Up
	{27, 91, 49, 59, 50, 66}:  "shift↓", // Shift-Down
	{27, 91, 49, 59, 50, 67}:  "shift→", // Shift-Right
	{27, 91, 49, 59, 50, 68}:  "shift←", // Shift-Left
	{27, 91, 49, 59, 50, 72}:  "shift⇱", // Shift-Home
	{27, 91, 49, 59, 50, 70}:  "shift⇲", // Shift-End
	{27, 91, 49, 59, 53, 72}:  "ctrl⇱",  // Ctrl-Home
	{27, 91, 49, 59, 53, 70}:  "ctrl⇲",  // Ctrl-End
	{27, 91, 49, 59, 51, 72}:  "alt⇱",   // Alt-Home
	{27, 91, 49, 59, 51, 70}:  "alt⇲",   // Alt-End
	{27, 91, 53, 59, 53, 126}: "ctrl⇞",  // Ctrl-PgUp
	{27, 91, 54, 59, 53, 126}: "ctrl⇟",  // Ctrl-PgDn
	{27, 91, 53, 59, 50, 126}: "shift⇞", // Shift-PgUp
	{27, 91, 54, 59, 50, 126}: "shift⇟", // Shift-PgDn
	{27, 91, 51, 59, 53, 126}: "ctrl⌦",  // Ctrl-Delete
	{27, 91, 51, 59, 50, 126}: "shift⌦", // Shift-Delete
	{27, 91, 50, 59, 50, 126}: "shift⎀", // Shift-Insert
}

// String representations for long CSI sequences (kitty keyboard protocol and xterm modifyOtherKeys=2)
var longCSILookup = map[string]string{
	"\x1b[13;2u":    "shift⏎", // Shift-Return (kitty CSI-u)
	"\x1b[13;3u":    "alt⏎",   // Alt-Return   (kitty CSI-u)
	"\x1b[27;2;13~": "shift⏎", // Shift-Return (xterm modifyOtherKeys=2)
	"\x1b[27;3;13~": "alt⏎",   // Alt-Return   (xterm modifyOtherKeys=2)
}

// parseFirstKey parses the first key sequence from buf and returns its string
// representation plus the number of bytes consumed. When the buffer starts
// with an incomplete sequence (e.g. only ESC), consumed == 0 signals the
// caller to try reading more bytes before classifying. A return of
// (key, consumed) with consumed > 0 means a complete key has been recognised.
func parseFirstKey(buf []byte) (string, int) {
	n := len(buf)
	if n == 0 {
		return "", 0
	}
	// Non-ESC single byte: plain character or control code.
	if buf[0] != 27 {
		r, size := utf8.DecodeRune(buf)
		if r == utf8.RuneError && size <= 1 {
			return "c:" + strconv.Itoa(int(buf[0])), 1
		}
		if unicode.IsPrint(r) {
			return string(r), size
		}
		return "c:" + strconv.Itoa(int(buf[0])), 1
	}
	// ESC alone: need more bytes to decide (might be start of CSI/SS3).
	if n < 2 {
		return "", 0
	}
	// Lone ESC followed by something that's not [ or O: it's the Escape key
	// (or Alt+key) — for orbiton's purposes return it as c:27 and keep the
	// next byte for the following call.
	if buf[1] != '[' && buf[1] != 'O' {
		// Alt-Return is reported as ESC + CR (or ESC + LF) on most terminals.
		// When both bytes have already arrived in the same buffer the user
		// pressed them together — a real Escape would have been consumed
		// before the next key arrived — so treat the pair as a single key.
		if buf[1] == 0x0D || buf[1] == 0x0A {
			return "alt⏎", 2
		}
		return "c:27", 1
	}
	// 3-byte sequences: ESC [ X   or   ESC O X
	if n >= 3 {
		seq3 := [3]byte{buf[0], buf[1], buf[2]}
		if str, ok := keyStringLookup[seq3]; ok {
			return str, 3
		}
	}
	// 4-byte sequences: ESC [ N ~
	if n >= 4 {
		seq4 := [4]byte{buf[0], buf[1], buf[2], buf[3]}
		if str, ok := pageStringLookup[seq4]; ok {
			return str, 4
		}
	}
	// 5-byte sequences: ESC [ N N ~
	if n >= 5 {
		seq5 := [5]byte{buf[0], buf[1], buf[2], buf[3], buf[4]}
		if str, ok := fKeyStringLookup[seq5]; ok {
			return str, 5
		}
	}
	// 6-byte modifier sequences: ESC [ 1 ; M X
	if n >= 6 {
		seq6 := [6]byte{buf[0], buf[1], buf[2], buf[3], buf[4], buf[5]}
		if str, ok := modKeyStringLookup[seq6]; ok {
			return str, 6
		}
	}
	// ESC [ [ X is the Linux console form of F1-F5, looked up above. The second
	// '[' is not a CSI final byte, so wait for the fourth byte to arrive rather
	// than letting the scan below consume an incomplete sequence.
	if n == 3 && buf[1] == '[' && buf[2] == '[' {
		return "", 0
	}
	// Unknown CSI sequence. Consume up to the terminator so stray bytes don't
	// get re-emitted as literal "^[[..." text. A CSI/SS3 final byte is in the
	// range 0x40-0x7E (or '~' for page-type sequences). rxvt also terminates
	// its Shift-modified sequences with '$', which is outside that range.
	if buf[1] == '[' || buf[1] == 'O' {
		for i := 2; i < n; i++ {
			b := buf[i]
			if (b >= 0x40 && b <= 0x7E) || b == '~' || b == '$' {
				seq := string(buf[:i+1])
				// Recognise long CSI sequences (kitty CSI-u, xterm
				// modifyOtherKeys=2) that report modified keys not
				// covered by the fixed-size lookups above.
				if str, ok := longCSILookup[seq]; ok {
					return str, i + 1
				}
				return seq, i + 1
			}
		}
		// Terminator not yet in buffer — wait for more bytes.
		return "", 0
	}
	// Fallback: consume one byte.
	return string(buf[:1]), 1
}

// ReadKey reads a key sequence (or printable character) from the TTY.
// When multiple key sequences arrive in one read (for example a held-down
// arrow key during a slow redraw), they are returned one by one on
// successive calls via a pending byte buffer — this prevents queued arrow
// escapes from leaking into the document as literal "^[[..." text.
func (tty *TTY) ReadKey() string {
	// Try to return a key already sitting in the pending buffer first. This is
	// done before touching the terminal: RawMode below performs two ioctl
	// syscalls, and calling it once per key while draining a large burst of
	// buffered input (such as a paste) makes pasting noticeably slow,
	// especially on macOS where those ioctls are expensive. Parsing from the
	// pending buffer does not read from the file descriptor, so the terminal
	// mode does not need to be re-applied here.
	if key, consumed := parseFirstKey(tty.pending); consumed > 0 {
		tty.pending = tty.pending[consumed:]
		return key
	}

	// Note: we deliberately do NOT restore the original terminal state or
	// flush the input queue on exit. Restoring would re-enable echo between
	// keystrokes (causing raw escape sequences like "\x1b[A" to be echoed
	// onto the screen while the editor is busy redrawing — visible as
	// literal "^[[A" and, in graphical book mode, as flicker/jumping).
	// Flushing would discard keystrokes the user typed while a redraw was
	// in progress. The outer editor loop restores the terminal on exit.
	tty.RawMode()

	// Need more bytes. Use a generous read buffer so bursts of queued input
	// (e.g. every \x1b[C from a held Right-arrow) are not split across reads.
	// Block until at least one byte arrives.
	savedTimeout, err := tty.SetTimeout(0)
	if err != nil {
		return ""
	}
	defer tty.SetTimeout(savedTimeout)

	readBuf := make([]byte, 256)
	numRead, err := tty.readBytes(readBuf)
	if numRead < 0 {
		numRead = 0
	}
	if err != nil && numRead == 0 {
		return ""
	}
	tty.pending = append(tty.pending, readBuf[:numRead]...)

	// If the pending buffer currently holds only an incomplete escape
	// sequence (e.g. lone ESC or ESC [ without a terminator), do one short
	// follow-up read to let the rest arrive before classifying.
	if key, consumed := parseFirstKey(tty.pending); consumed > 0 {
		tty.pending = tty.pending[consumed:]
		return key
	}
	// Incomplete: wait briefly for the tail of the escape sequence.
	tty.SetTimeoutNoSave(defaultTimeout)
	numRead2, _ := tty.readBytes(readBuf)
	if numRead2 > 0 {
		tty.pending = append(tty.pending, readBuf[:numRead2]...)
	}
	if key, consumed := parseFirstKey(tty.pending); consumed > 0 {
		tty.pending = tty.pending[consumed:]
		return key
	}
	// Still nothing parseable (shouldn't normally happen); flush the pending
	// bytes as-is so we don't deadlock on them. A lone ESC byte that never
	// got a continuation is the Escape key itself — return it as "c:27" so
	// callers that compare against the canonical key string (e.g. menu
	// dismissal) continue to work.
	if len(tty.pending) == 1 && tty.pending[0] == 27 {
		tty.pending = tty.pending[:0]
		return "c:27"
	}
	s := string(tty.pending)
	tty.pending = tty.pending[:0]
	return s
}
