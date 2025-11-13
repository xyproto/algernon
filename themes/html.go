package themes

// Functions directly related to HTML, CSS, Amber or GCSS

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

const (
	// DefaultCSSFilename is the default CSS stylesheet
	DefaultCSSFilename = "style.css"

	// DefaultGCSSFilename is the default GCSS stylesheet
	DefaultGCSSFilename = "style.gcss"

	// DefaultTheme is the default theme when rendering Markdown and for error pages
	DefaultTheme = "default"
)

// MetaKeywords contains a selection of allowed keywords for the HTML meta tag
var MetaKeywords = []string{"application-name", "author", "description", "generator", "keywords", "robots", "language", "googlebot", "Slurp", "bingbot", "geo.position", "geo.placename", "geo.region", "ICBM", "viewport"}

// MessagePage is an easy way to output a HTML page only given a title, the body
// (will be placed between the <body></body> tags) and the name of one of the
// built-in themes. Does not close <body> and <html>.
// Deprecated
func MessagePage(title, body, theme string) string {
	return fmt.Sprintf("<!doctype html><html><head><title>%s</title>%s</head><body><h1>%s</h1>%s", title, StyleHead(theme), title, body)
}

var policy = bluemonday.UGCPolicy()

// StyleHead returns contents that goes in "<head>", as bytes.
// This is either CSS wrapped in a "<style>" tag, or "<link>" tags to CSS and JS.
func StyleHead(theme string) []byte {

	// Sanitize the theme name
	theme = policy.Sanitize(theme)

	var buf bytes.Buffer
	if theme == "material" {
		buf.WriteString(MaterialHead())
	}

	if strings.HasSuffix(theme, ".css") {
		buf.WriteString("<style>html { margin: 3em; }</style>")
		buf.WriteString("<link rel=\"stylesheet\" href=\"" + theme + "\">")
	} else {
		buf.WriteString("<style>")
		buf.Write(builtinThemes[theme])
		buf.WriteString("</style>")
	}
	return buf.Bytes()
}

// MessagePageBytes provides the same functionalityt as MessagePage,
// but with []byte instead of string, and without closing </body></html>
func MessagePageBytes(title string, body []byte, theme string) []byte {

	// Sanitize the theme and title
	theme = policy.Sanitize(theme)
	title = policy.Sanitize(title)

	var buf bytes.Buffer
	buf.WriteString("<!doctype html><html><head><title>")
	buf.WriteString(title)
	buf.WriteString("</title>")
	buf.Write(StyleHead(theme))
	buf.WriteString("</head><body><h1>")
	buf.WriteString(title)
	buf.WriteString("</h1>")
	buf.Write(body)
	return buf.Bytes()
}

// SimpleHTMLPage provides a quick way to build a HTML page
func SimpleHTMLPage(title, headline, inhead, body, language []byte) []byte {

	// Sanitize the title, headline and language
	titleString := policy.Sanitize(string(title))
	headlineString := policy.Sanitize(string(headline))
	languageString := policy.Sanitize(string(language))

	var buf bytes.Buffer
	if len(languageString) > 0 {
		buf.WriteString("<!doctype html><html lang=\"")
		buf.WriteString(languageString)
		buf.WriteString("\">")
	} else {
		buf.WriteString("<!doctype html><html>")
	}
	if len(titleString) > 0 {
		buf.WriteString("<head><title>")
		buf.WriteString(titleString)
		buf.WriteString("</title></head>")
	}
	buf.Write(inhead)
	buf.WriteString("<body>")
	if len(headlineString) > 0 {
		buf.WriteString("<h1>")
		buf.WriteString(headlineString)
		buf.WriteString("</h1>")
	}
	buf.Write(body)
	return buf.Bytes()
}

// HTMLLink builds an HTML link given the link text, the URL to a file/directory
// and a boolean that is true if the given URL is to a directory.
func HTMLLink(text, url string, isDirectory bool) string {

	// Sanitize the link text and the link URL
	text = policy.Sanitize(text)
	url = policy.Sanitize(url)

	// Add a final slash, if needed
	if isDirectory {
		text += "/"
		url += "/"
	}
	return "<a href=\"/" + url + "\">" + text + "</a><br>"
}

// StyleAmber modifies Amber source code so that a link to the given stylesheet URL is added
func StyleAmber(amberdata []byte, url string) []byte {

	// Sanitize the URL
	url = policy.Sanitize(url)

	// If the given url is not already mentioned and the data contains "body"
	if !bytes.Contains(amberdata, []byte(url)) && bytes.Contains(amberdata, []byte("html")) && bytes.Contains(amberdata, []byte("body")) {
		// Extract one level of indendation
		whitespace := OneLevelOfIndentation(&amberdata, "body")
		// Check if there already is a head section
		if bytes.Contains(amberdata, []byte("head")) {
			// Add a link to the stylesheet
			return bytes.Replace(amberdata, []byte("head\n"), []byte("head\n"+whitespace+whitespace+`link[href="`+url+`"][rel="stylesheet"][type="text/css"]`+"\n"), 1)
		} else if bytes.Contains(amberdata, []byte("body")) {
			// Add a link to the stylesheet
			return bytes.Replace(amberdata, []byte("html\n"), []byte("html\n"+whitespace+"head\n"+whitespace+whitespace+`link[href="`+url+`"][rel="stylesheet"][type="text/css"]`+"\n"), 1)
		}
	}
	return amberdata
}

// StyleHTML modifies HTML source code so that a link to the given stylesheet URL is added
func StyleHTML(htmldata []byte, url string) []byte {

	// Sanitize the URL
	url = policy.Sanitize(url)

	// If the given url is not already mentioned and the data contains "body"
	if !bytes.Contains(htmldata, []byte(url)) && bytes.Contains(htmldata, []byte("body")) {
		if bytes.Contains(htmldata, []byte("</head>")) {
			return bytes.Replace(htmldata, []byte("</head>"), []byte("  <link rel=\"stylesheet\" href=\""+url+"\">\n  </head>"), 1)
		} else if bytes.Contains(htmldata, []byte("<body>")) {
			return bytes.Replace(htmldata, []byte("<body>"), []byte("  <head>\n  <link rel=\"stylesheet\" href=\""+url+"\">\n  </head>\n  <body>"), 1)
		}
	}
	return htmldata
}

// InsertDoctype inserts <doctype html> to the HTML, if missing.
// Does not check if the given data is HTML. Assumes it to be HTML.
func InsertDoctype(htmldata []byte) []byte {
	// If there are more than two lines
	if bytes.Count(htmldata, []byte("\n")) > 2 {
		fields := bytes.SplitN(htmldata, []byte("\n"), 3)
		line1 := strings.ToLower(string(fields[0]))
		line2 := strings.ToLower(string(fields[1]))
		if strings.Contains(line1, "doctype") || strings.Contains(line2, "doctype") {
			return htmldata
		}
		// Doctype is missing from the first two lines, add it
		return []byte("<!doctype html>\n" + string(htmldata))
	}
	return htmldata
}

// NoPage generates a HTML page for when a file is not found
func NoPage(filename, theme string) []byte {

	// Sanitize the filename and the theme name
	filename = policy.Sanitize(filename)
	theme = policy.Sanitize(theme)

	// Return a HTML page
	return MessagePageBytes("Not found", []byte("File not found: "+filename), theme)
}
