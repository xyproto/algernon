package onthefly

import (
	"strconv"
)

// StandaloneTag creates an empty page the only contains a tag with the given
// name. Returns both the Page and the Tag.
func StandaloneTag(tagname string) (*Page, *Tag) {
	page := NewPage("blank", tagname)
	tag, _ := page.GetTag(tagname)
	return page, tag
}

// TagString creates a new page with the given tag (name).
// The page is then returned as a string.
func TagString(tagname string) string {
	return NewPage("blank", tagname).String()
}

// SetPixelPosition sets the absolute CSS position of a HTML tag
func SetPixelPosition(tag *Tag, xpx, ypx int) {
	tag.AddStyle("position", "absolute")
	xpxs := strconv.Itoa(xpx) + "px"
	ypxs := strconv.Itoa(ypx) + "px"
	tag.AddStyle("top", xpxs)
	tag.AddStyle("left", ypxs)
}

// SetRelativePosition sets the relative CSS position of a HTML tag
func SetRelativePosition(tag *Tag, x, y string) {
	tag.AddStyle("position", "relative")
	tag.AddStyle("top", x)
	tag.AddStyle("left", y)
}

// SetWidthAndSide sets the "width" and also "float: left" or "float: right"
func SetWidthAndSide(tag *Tag, width string, leftSide bool) {
	side := "right"
	if leftSide {
		side = "left"
	}
	tag.AddStyle("float", side)
	tag.AddStyle("width", width)
}
