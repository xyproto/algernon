package vt

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
	{27, 91, 'H'}: 1,           // Home (mapped to Ctrl-A)
	{27, 91, 'F'}: 5,           // End (mapped to Ctrl-E)
	{27, 91, 90}:  KeyShiftTab, // Shift-Tab / Backtab (ESC [Z)
	{27, 79, 80}:  KeyF1,       // F1  (ESC O P)
	{27, 79, 81}:  KeyF2,       // F2  (ESC O Q)
	{27, 79, 82}:  KeyF3,       // F3  (ESC O R)
	{27, 79, 83}:  KeyF4,       // F4  (ESC O S)
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
}

// Key codes for 5-byte sequences (F5-F12)
var fKeyLookup = map[[5]byte]int{
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
}

// String representations for 3-byte sequences
var keyStringLookup = map[[3]byte]string{
	{27, 91, 65}:  "↑",       // Up Arrow
	{27, 91, 66}:  "↓",       // Down Arrow
	{27, 91, 67}:  "→",       // Right Arrow
	{27, 91, 68}:  "←",       // Left Arrow
	{27, 91, 'H'}: "⇱",       // Home
	{27, 91, 'F'}: "⇲",       // End
	{27, 79, 'H'}: "⇱",       // Home (SS3 sequence)
	{27, 79, 'F'}: "⇲",       // End (SS3 sequence)
	{27, 91, 90}:  "backtab", // Shift-Tab / Backtab (ESC [Z)
	{27, 79, 80}:  "F1",      // F1  (ESC O P)
	{27, 79, 81}:  "F2",      // F2  (ESC O Q)
	{27, 79, 82}:  "F3",      // F3  (ESC O R)
	{27, 79, 83}:  "F4",      // F4  (ESC O S)
}

// String representations for 4-byte sequences
var pageStringLookup = map[[4]byte]string{
	{27, 91, 49, 126}: "⇱", // Home
	{27, 91, 51, 126}: "⌦", // Delete / Forward-Delete
	{27, 91, 52, 126}: "⇲", // End
	{27, 91, 53, 126}: "⇞", // Page Up
	{27, 91, 54, 126}: "⇟", // Page Down
	{27, 91, 55, 126}: "⇱", // Home
	{27, 91, 56, 126}: "⇲", // End
}

// String representations for 5-byte sequences (F5-F12)
var fKeyStringLookup = map[[5]byte]string{
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
	{27, 91, 50, 59, 53, 126}: "⎘",      // Ctrl-Insert
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
}

// String representations for long CSI sequences (kitty keyboard protocol and xterm modifyOtherKeys=2)
var longCSILookup = map[string]string{
	"\x1b[13;2u":    "shift⏎", // Shift-Return (kitty CSI-u)
	"\x1b[13;3u":    "alt⏎",   // Alt-Return   (kitty CSI-u)
	"\x1b[27;2;13~": "shift⏎", // Shift-Return (xterm modifyOtherKeys=2)
	"\x1b[27;3;13~": "alt⏎",   // Alt-Return   (xterm modifyOtherKeys=2)
}
