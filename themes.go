package main

import (
	"bytes"
	"fmt"
	"strings"
)

const (
	// Default stylesheet filename (GCSS)
	defaultStyleFilename = "style.gcss"

	// highlight.js style for custom CSS
	defaultCustomCodeStyle = "github"
)

var (
	// The available built-in CSS themes. Corresponds with the font themes below.
	builtinThemes = map[string]string{
		"gray":   "<style>@import url(//fonts.googleapis.com/css?family=Lato:300); body { background-color: #e7eaed; color: #0b0b0b; font-family: 'Lato', sans-serif; font-weight: 300;  margin: 4.5em; font-size: 1em; } a { color: #401010; font-family: courier; } a:hover { color: #801010; } a:active { color: yellow; } h1 { color: #101010; } img { max-width: 100%; }</style>",
		"dark":   "<style>@import url(//fonts.googleapis.com/css?family=Lato:400); body { background-color: #101010; color: #f0f0f0; font-family: 'Lato', sans-serif; font-weight: 400;  margin: 4.5em; font-size: 1em; } a { color: #c0a0a0; font-family: courier; } a:hover { color: #f0a0a0; } a:active { color: yellow; } h1 { color: #f0f0f0; } img { max-width: 100%; }</style>",
		"redbox": "<style>@import url(//fonts.googleapis.com/css?family=Monoton|Monofett);html{background-color:#222;}body{color:#111;background-color:#999;font-family:Impact,'Arial Black',sans-serif;font-size:1.7em;margin:2.7em;padding:0 5em 1em 2em;border-radius:50px;border:solid 10px #a00;box-shadow:10px 10px 16px black, 6px 6px 8px #222 inset;}h2{font-family:Monoton,cursive;color:black;}ul{margin-left:1em;}a{text-decoration:none;color:#b00;}a:hover{color:#dc0;}code{font-family:Monofett,cursive;} img { max-width: 100%; }</style>",
	}

	// Built in themes corresponding to highlight.js styles
	// See https://github.com/isagalaev/highlight.js/tree/master/src/styles for more styles
	defaultCodeStyles = map[string]string{"gray": "color-brewer", "dark": "ocean", "redbox": "railscasts"}

	// A selection of allowed keywords for the HTML meta tag
	metaKeywords = []string{"application-name", "author", "description", "generator", "keywords", "robots", "language", "googlebot", "Slurp", "bingbot", "geo.position", "geo.placename", "geo.region", "ICBM", "viewport"}
)

// Easy way to output a HTML page
func easyPage(title, body, theme string) string {
	return fmt.Sprintf("<!doctype html><html><head><title>%s</title>%s<style>%s</style><head><body><h1>%s</h1>%s</body></html>", title, "", builtinThemes[theme], title, body)
}

// Easy way to build links to directories
func easyLink(text, url string, isDirectory bool) string {
	// Add a final slash, if needed
	if isDirectory {
		text += "/"
		url += "/"
	}
	return "<a href=\"/" + url + "\">" + text + "</a><br>"
}

// HTML to be added for enabling highlighting.
func highlightHTML(codeStyle string) string {
	return `<link rel="stylesheet" href="//cdnjs.cloudflare.com/ajax/libs/highlight.js/9.6.0/styles/` + codeStyle + `.min.css"><script src="//cdnjs.cloudflare.com/ajax/libs/highlight.js/9.6.0/highlight.min.js"></script><script>hljs.initHighlightingOnLoad();</script>`
}

// Add a link to a stylesheet in the given Amber code
func linkToStyle(amberdata *[]byte, url string) {
	// If the given url is not already mentioned and the data contains "body"
	if !bytes.Contains(*amberdata, []byte(url)) && bytes.Contains(*amberdata, []byte("html")) && bytes.Contains(*amberdata, []byte("body")) {
		// Extract one level of indendation
		whitespace := oneLevelOfIndentation(amberdata, "body")
		// Check if there already is a head section
		if bytes.Contains(*amberdata, []byte("head")) {
			// Add a link to the stylesheet
			*amberdata = bytes.Replace(*amberdata, []byte("head\n"), []byte("head\n"+whitespace+whitespace+`link[href="`+url+`"][rel="stylesheet"][type="text/css"]`+"\n"), 1)

		} else if bytes.Contains(*amberdata, []byte("body")) {

			// Add a link to the stylesheet
			*amberdata = bytes.Replace(*amberdata, []byte("html\n"), []byte("html\n"+whitespace+"head\n"+whitespace+whitespace+`link[href="`+url+`"][rel="stylesheet"][type="text/css"]`+"\n"), 1)
		}
	}
}

// Add a link to a stylesheet in the given HTML code
func linkToStyleHTML(htmldata *[]byte, url string) {
	// If the given url is not already mentioned and the data contains "body"
	if !bytes.Contains(*htmldata, []byte(url)) && bytes.Contains(*htmldata, []byte("body")) {
		if bytes.Contains(*htmldata, []byte("</head>")) {
			*htmldata = bytes.Replace(*htmldata, []byte("</head>"), []byte("  <link rel=\"stylesheet\" href=\""+url+"\">\n  </head>"), 1)
		} else if bytes.Contains(*htmldata, []byte("<body>")) {
			*htmldata = bytes.Replace(*htmldata, []byte("<body>"), []byte("  <head>\n  <link rel=\"stylesheet\" href=\""+url+"\">\n  </head>\n  <body>"), 1)
		}
	}
}

// Add meta tag names to the given map
func addMetaKeywords(keywords map[string]string) {
	for _, keyword := range metaKeywords {
		keywords[keyword] = ""
	}
}

// Insert doctype in HTML, if missing
// Does not check if the given data is HTML. Assumes it to be HTML.
func insertDoctype(htmldata []byte) []byte {
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
