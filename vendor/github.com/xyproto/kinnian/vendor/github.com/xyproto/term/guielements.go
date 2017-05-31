package term

// Draw the background color. Clear the screen.
func DrawBackground() {
	Clear()
}

// Draw a box using ASCII graphics.
// The given Box struct defines the size and placement.
// If extrude is True, the box looks a bit more like it's sticking out.
func DrawBox(r *Box, extrude bool) *Rect {
	x := r.frame.X
	y := r.frame.Y
	width := r.frame.W
	height := r.frame.H
	FG1 := BOXLIGHT
	FG2 := BOXDARK
	if !extrude {
		FG1 = BOXDARK
		FG2 = BOXLIGHT
	}
	Write(x, y, TLCHAR, FG1, BOXBG)
	Write(x+1, y, Repeat(HCHAR, width-2), FG1, BOXBG)
	Write(x+width-1, y, TRCHAR, FG1, BOXBG)
	for i := y + 1; i < y+height; i++ {
		Write(x, i, VCHAR, FG1, BOXBG)
		Write(x+1, i, Repeat(" ", width-2), FG1, BOXBG)
		Write(x+width-1, i, VCHAR2, FG2, BOXBG)
	}
	Write(x, y+height-1, BLCHAR, FG1, BOXBG)
	Write(x+1, y+height-1, Repeat(HCHAR2, width-2), FG2, BOXBG)
	Write(x+width-1, y+height-1, BRCHAR, FG2, BOXBG)
	return &Rect{x, y, width, height}
}

// Draw a list widget. Takes a Box struct for the size and position.
// Takes a list of strings to be listed and an int that represents
// which item is currently selected. Does not scroll or wrap.
func DrawList(r *Box, items []string, selected int) {
	for i, s := range items {
		color := LISTTEXT
		if i == selected {
			color = LISTFOCUS
		}
		Write(r.frame.X, r.frame.Y+i, s, color, BOXBG)
	}
}

// Draws a button widget at the given placement,
// with the given text. If active is False,
// it will look more "grayed out".
func DrawButton(x, y int, text string, active bool) {
	color := BUTTONTEXT
	if active {
		color = BUTTONFOCUS
	}
	Write(x, y, "<  ", color, BG)
	Write(x+3, y, text, color, BG)
	Write(x+3+len(text), y, "  >", color, BG)
}

// Outputs a multiline string at the given coordinates.
// Uses the box background color.
// Returns the final y coordinate after drawing.
func DrawAsciiArt(x, y int, text string) int {
	var i int
	for i, line := range Splitlines(text) {
		Write(x, y+i, line, TEXTCOLOR, BOXBG)
	}
	return y + i
}

// Outputs a multiline string at the given coordinates.
// Uses the default background color.
// Returns the final y coordinate after drawing.
func DrawRaw(x, y int, text string) int {
	var i int
	for i, line := range Splitlines(text) {
		Write(x, y+i, line, TEXTCOLOR, BG)
	}
	return y + i
}
