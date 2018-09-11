package onthefly

// Various "hardcoded" stylistic choices

// RoundedBox styles the tag as a box with a rounded border
func (tag *Tag) RoundedBox() {
	tag.AddStyle("border", "solid 1px #b4b4b4")
	tag.AddStyle("border-radius", "10px")
	tag.AddStyle("box-shadow", "1px 1px 3px rgba(0,0,0, .5)")
}

// SansSerif styles the tag with some sort of sans-serif font
func (tag *Tag) SansSerif() {
	tag.AddStyle("font-family", "Verdana, Geneva, sans-serif")
}

// CustomSansSerif styles the tag with some sort of sans-serif font,
// where a custom font can be given and put first in the list of fonts.
func (tag *Tag) CustomSansSerif(custom string) {
	tag.AddStyle("font-family", custom+", Verdana, Geneva, sans-serif")
}

// AddGoogleFonts links to a given Google Font name
func AddGoogleFonts(page *Page, googleFonts []string) {
	for _, fontname := range googleFonts {
		page.LinkToGoogleFont(fontname)
	}
}

// SetMarginAndBackgroundImage sets a small margin of 1em and a given background image
func (page *Page) SetMarginAndBackgroundImage(imageURL string, stretchBackground bool) {
	body, _ := page.SetMargin(1)
	if stretchBackground {
		body.AddStyle("background", "url('"+imageURL+"') no-repeat center center fixed")
	} else {
		body.AddStyle("background", "url('"+imageURL+"')")
	}
}

// AddBodyStyle styles the given page with a background image to the given page.
// Deprecated, use SetMarginAndBackgroundImage and also SansSerif instead.
func AddBodyStyle(page *Page, bgimageurl string, stretchBackground bool) {
	body, _ := page.SetMargin(1)
	body.SansSerif()
	if stretchBackground {
		body.AddStyle("background", "url('"+bgimageurl+"') no-repeat center center fixed")
	} else {
		body.AddStyle("background", "url('"+bgimageurl+"')")
	}
}

// AddStyle adds inline CSS to a page. Returns the style tag.
func (page *Page) AddStyle(s string) (*Tag, error) {
	head, err := page.GetTag("head")
	if err != nil {
		return nil, err
	}
	style := head.AddNewTag("style")
	style.AddContent(s)
	return style, nil
}
