package vt

// Key codes for 3-byte sequences (arrows, Home, End, F1-F4, Shift-Tab)
var keyCodeLookup = map[[3]byte]int{
	{27, 91, 65}:  253, // Up Arrow
	{27, 91, 66}:  255, // Down Arrow
	{27, 91, 67}:  254, // Right Arrow
	{27, 91, 68}:  252, // Left Arrow
	{27, 91, 'H'}: 1,   // Home (Ctrl-A)
	{27, 91, 'F'}: 5,   // End (Ctrl-E)
	{27, 91, 90}:  273, // Shift-Tab / Backtab (ESC [Z)
	{27, 79, 80}:  274, // F1  (ESC O P)
	{27, 79, 81}:  275, // F2  (ESC O Q)
	{27, 79, 82}:  276, // F3  (ESC O R)
	{27, 79, 83}:  277, // F4  (ESC O S)
}

// Key codes for 4-byte sequences (Page Up, Page Down, Home, End, Delete)
var pageNavLookup = map[[4]byte]int{
	{27, 91, 49, 126}: 1,   // Home (ESC [1~)
	{27, 91, 51, 126}: 278, // Delete / Forward-Delete (ESC [3~)
	{27, 91, 52, 126}: 5,   // End (ESC [4~)
	{27, 91, 53, 126}: 251, // Page Up (custom code)
	{27, 91, 54, 126}: 250, // Page Down (custom code)
	{27, 91, 55, 126}: 1,   // Home (ESC [7~)
	{27, 91, 56, 126}: 5,   // End (ESC [8~)
}

// Key codes for 5-byte sequences (F5-F12)
var fKeyLookup = map[[5]byte]int{
	{27, 91, 49, 53, 126}: 279, // F5  (ESC [15~)
	{27, 91, 49, 55, 126}: 280, // F6  (ESC [17~)
	{27, 91, 49, 56, 126}: 281, // F7  (ESC [18~)
	{27, 91, 49, 57, 126}: 282, // F8  (ESC [19~)
	{27, 91, 50, 48, 126}: 283, // F9  (ESC [20~)
	{27, 91, 50, 49, 126}: 284, // F10 (ESC [21~)
	{27, 91, 50, 51, 126}: 285, // F11 (ESC [23~)
	{27, 91, 50, 52, 126}: 286, // F12 (ESC [24~)
}

// Key codes for 6-byte modifier-key sequences (CSI with modifier parameter)
var modKeyLookup = map[[6]byte]int{
	{27, 91, 50, 59, 53, 126}: 258, // Ctrl-Insert   (ESC [2;5~)
	{27, 91, 49, 59, 51, 65}:  259, // Alt-Up        (ESC [1;3A)
	{27, 91, 49, 59, 51, 66}:  260, // Alt-Down      (ESC [1;3B)
	{27, 91, 49, 59, 51, 67}:  261, // Alt-Right     (ESC [1;3C)
	{27, 91, 49, 59, 51, 68}:  262, // Alt-Left      (ESC [1;3D)
	{27, 91, 49, 59, 53, 65}:  263, // Ctrl-Up       (ESC [1;5A)
	{27, 91, 49, 59, 53, 66}:  264, // Ctrl-Down     (ESC [1;5B)
	{27, 91, 49, 59, 53, 67}:  265, // Ctrl-Right    (ESC [1;5C)
	{27, 91, 49, 59, 53, 68}:  266, // Ctrl-Left     (ESC [1;5D)
	{27, 91, 49, 59, 50, 65}:  267, // Shift-Up      (ESC [1;2A)
	{27, 91, 49, 59, 50, 66}:  268, // Shift-Down    (ESC [1;2B)
	{27, 91, 49, 59, 50, 67}:  269, // Shift-Right   (ESC [1;2C)
	{27, 91, 49, 59, 50, 68}:  270, // Shift-Left    (ESC [1;2D)
	{27, 91, 49, 59, 50, 72}:  271, // Shift-Home    (ESC [1;2H)
	{27, 91, 49, 59, 50, 70}:  272, // Shift-End     (ESC [1;2F)
	{27, 91, 49, 59, 53, 72}:  287, // Ctrl-Home     (ESC [1;5H)
	{27, 91, 49, 59, 53, 70}:  288, // Ctrl-End      (ESC [1;5F)
	{27, 91, 49, 59, 51, 72}:  289, // Alt-Home      (ESC [1;3H)
	{27, 91, 49, 59, 51, 70}:  290, // Alt-End       (ESC [1;3F)
	{27, 91, 53, 59, 53, 126}: 291, // Ctrl-PgUp     (ESC [5;5~)
	{27, 91, 54, 59, 53, 126}: 292, // Ctrl-PgDn     (ESC [6;5~)
	{27, 91, 53, 59, 50, 126}: 293, // Shift-PgUp    (ESC [5;2~)
	{27, 91, 54, 59, 50, 126}: 294, // Shift-PgDn    (ESC [6;2~)
	{27, 91, 51, 59, 53, 126}: 295, // Ctrl-Delete   (ESC [3;5~)
	{27, 91, 51, 59, 50, 126}: 296, // Shift-Delete  (ESC [3;2~)
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
