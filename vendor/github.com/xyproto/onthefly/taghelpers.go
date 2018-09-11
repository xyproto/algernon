package onthefly

import (
	"strconv"
)

// SetMargin sets a margin given in em on a tag
func (tag *Tag) SetMargin(em int) {
	value := strconv.Itoa(em) + "em"
	tag.AddStyle("margin", value)
}

// SetRounded applies rounded corners to an HTML tag
func (tag *Tag) SetRounded(value string) {
	tag.AddStyle("border-radius", value)
	tag.AddStyle("-webkit-border-radius", value)
	tag.AddStyle("-moz-border-radius", value)
}

// SetRoundedEm applies rounded corners to an HTML tag
// em is the roundedness, given as "em"
func (tag *Tag) SetRoundedEm(em int) {
	value := strconv.Itoa(em) + "em"
	tag.AddStyle("border-radius", value)
	tag.AddStyle("-webkit-border-radius", value)
	tag.AddStyle("-moz-border-radius", value)
}

// SetColor changes the forground and background color CSS styles
func (tag *Tag) SetColor(fgColor, bgColor string) {
	tag.AddStyle("color", fgColor)
	tag.AddStyle("background-color", bgColor)
}

// AddBox adds a <div> box
func (tag *Tag) AddBox(id string, rounded bool, em, text, fgColor, bgColor, leftPadding string) *Tag {
	div := tag.AddNewTag("div")
	div.AddAttrib("id", id)
	div.AddContent(text)
	if rounded {
		div.SetRounded(em)
	}
	div.SetColor(fgColor, bgColor)
	div.AddStyle("padding-left", leftPadding)
	return div
}

// Add an <img> image
func (tag *Tag) AddImage(url string, width string) *Tag {
	img := tag.AddNewTag("img")
	img.AddAttrib("src", url)
	img.AddStyle("width", width)
	return img
}

// Repeat the background. repeat can be for instance "repeat-x"
func (tag *Tag) RepeatBackground(bgimageurl, repeat string) {
	tag.AddStyle("background-image", "url('"+bgimageurl+"')")
	tag.AddStyle("background-repeat", repeat)
}
