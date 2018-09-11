package onthefly

import (
	"strconv"
	"strings"
)

type (
	// Various function signatures for handling requests
	SimpleWebHandle (func(string) string)
	TemplateValues  map[string]string
)

// Create a blank HTML5 page
func NewHTML5Page(titleText string) *Page {
	page := NewPage(titleText, "<!doctype html>")
	html := page.root.AddNewTag("html")
	head := html.AddNewTag("head")
	title := head.AddNewTag("title")
	title.AddContent(titleText)
	html.AddNewTag("body")
	return page
}

// Create a blank HTML5 page that links with Angular.JS
func NewAngularPage(titleText, angularVersion string) *Page {
	page := NewPage(titleText, "<!doctype html>")
	html := page.root.AddNewTag("html")
	html.AddSingularAttrib("ng-app")
	head := html.AddNewTag("head")
	title := head.AddNewTag("title")
	title.AddContent(titleText)
	html.AddNewTag("body")
	// Must be added after head has been added
	page.LinkToJS("//ajax.googleapis.com/ajax/libs/angularjs/" + angularVersion + "/angular.min.js")
	return page
}

// Set the margins of the body
func (page *Page) SetMargin(em int) (*Tag, error) {
	value := strconv.Itoa(em) + "em"
	return page.bodyAttr("margin", value)
}

// Disable scrollbars. Needed when using "<!doctype html>" together with fullscreen canvas/webgl
func (page *Page) NoScrollbars() (*Tag, error) {
	return page.bodyAttr("overflow", "hidden")
}

// Prepare for a canvas/webgl tag that covers the entire page
func (page *Page) FullCanvas() {
	page.SetMargin(0)
	// overflow:hidden
	page.NoScrollbars()
	// Inline CSS
	page.AddStyle("canvas { width: 100%; height: 100%; }")
}

// Set one of the CSS styles of the body
func (page *Page) bodyAttr(key, value string) (*Tag, error) {
	tag, err := page.root.GetTag("body")
	if err == nil {
		tag.AddStyle(key, value)
	}
	return tag, err
}

// Set the foreground and background color of the body
func (page *Page) SetColor(fgColor string, bgColor string) (*Tag, error) {
	tag, err := page.root.GetTag("body")
	if err == nil {
		tag.AddStyle("color", fgColor)
		tag.AddStyle("background-color", bgColor)
	}
	return tag, err
}

// Set the font family
func (page *Page) SetFontFamily(fontFamily string) (*Tag, error) {
	return page.bodyAttr("font-family", fontFamily)
}

// Add a box, for testing
func (page *Page) addBox(id string, rounded bool) (*Tag, error) {
	tag, err := page.root.GetTag("body")
	if err == nil {
		return tag.AddBox(id, rounded, "0.9em", "Speaks browser so you don't have to", "white", "black", "3em"), nil
	}
	return tag, err
}

// Link a page up with a CSS file
// Takes the url to a CSS file as a string
// The given page must have a "head" tag for this to work
// Returns an error if no "head" tag is found, or nil
func (page *Page) LinkToCSS(cssurl string) error {
	head, err := page.GetTag("head")
	if err == nil {
		link := head.AddNewTag("link")
		link.AddAttrib("rel", "stylesheet")
		link.AddAttrib("href", cssurl)
		link.AddAttrib("type", "text/css")
	}
	return err
}

// Link a page up with a Favicon file
// Takes the url to a favicon file as a string
// The given page must have a "head" tag for this to work
// Returns an error if no "head" tag is found, or nil
func (page *Page) LinkToFavicon(favurl string) error {
	head, err := page.root.GetTag("head")
	if err == nil {
		link := head.AddNewTag("link")
		link.AddAttrib("rel", "shortcut icon")
		link.AddAttrib("href", favurl)
	}
	return err
}

// Takes a charset, for example UTF-8, and creates a <meta> tag in <head>
func (page *Page) MetaCharset(charset string) error {
	// Add a meta tag
	head, err := page.GetTag("head")
	if err == nil {
		meta := head.AddNewTag("meta")
		meta.AddAttrib("http-equiv", "Content-Type")
		meta.AddAttrib("content", "text/html; charset="+charset)
	}
	return err
}

// Link to Google Fonts
func (page *Page) LinkToGoogleFont(name string) error {
	url := "http://fonts.googleapis.com/css?family="
	// Replace space with +, if needed
	if strings.Contains(name, " ") {
		url += strings.Replace(name, " ", "+", -1)
	} else {
		url += name
	}
	// Link to the CSS for the given font name
	return page.LinkToCSS(url)
}

// Add javascript to the header and specify UTF-8 as the charset
func AddHeader(page *Page, js string) {
	page.MetaCharset("UTF-8")
	AddScriptToHeader(page, js)
}
