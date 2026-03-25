package vt

// Key codes for 3-byte sequences (arrows, Home, End)
var keyCodeLookup = map[[3]byte]int{
	{27, 91, 65}:  253, // Up Arrow
	{27, 91, 66}:  255, // Down Arrow
	{27, 91, 67}:  254, // Right Arrow
	{27, 91, 68}:  252, // Left Arrow
	{27, 91, 'H'}: 1,   // Home (Ctrl-A)
	{27, 91, 'F'}: 5,   // End (Ctrl-E)
}

// Key codes for 4-byte sequences (Page Up, Page Down, Home, End)
var pageNavLookup = map[[4]byte]int{
	{27, 91, 49, 126}: 1,   // Home (ESC [1~)
	{27, 91, 52, 126}: 5,   // End (ESC [4~)
	{27, 91, 53, 126}: 251, // Page Up (custom code)
	{27, 91, 54, 126}: 250, // Page Down (custom code)
	{27, 91, 55, 126}: 1,   // Home (ESC [7~)
	{27, 91, 56, 126}: 5,   // End (ESC [8~)
}

// Key codes for 6-byte sequences (Ctrl-Insert)
var ctrlInsertLookup = map[[6]byte]int{
	{27, 91, 50, 59, 53, 126}: 258, // Ctrl-Insert (ESC [2;5~)
}

// String representations for 3-byte sequences
var keyStringLookup = map[[3]byte]string{
	{27, 91, 65}:  "↑", // Up Arrow
	{27, 91, 66}:  "↓", // Down Arrow
	{27, 91, 67}:  "→", // Right Arrow
	{27, 91, 68}:  "←", // Left Arrow
	{27, 91, 'H'}: "⇱", // Home
	{27, 91, 'F'}: "⇲", // End
	{27, 79, 'H'}: "⇱", // Home (SS3 sequence)
	{27, 79, 'F'}: "⇲", // End (SS3 sequence)
}

// String representations for 4-byte sequences
var pageStringLookup = map[[4]byte]string{
	{27, 91, 49, 126}: "⇱", // Home
	{27, 91, 52, 126}: "⇲", // End
	{27, 91, 53, 126}: "⇞", // Page Up
	{27, 91, 54, 126}: "⇟", // Page Down
	{27, 91, 55, 126}: "⇱", // Home
	{27, 91, 56, 126}: "⇲", // End
}

// String representations for 6-byte sequences (Ctrl-Insert)
var ctrlInsertStringLookup = map[[6]byte]string{
	{27, 91, 50, 59, 53, 126}: "⎘", // Ctrl-Insert (Copy)
}
