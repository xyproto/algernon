package utils

import (
	"bytes"
	"fmt"
	"strings"
)

const (
	// Default stylesheet filename (GCSS)
	DefaultStyleFilename = "style.gcss"

	// highlight.js style for custom CSS
	DefaultCustomCodeStyle = "github"
)

var (
	// The available built-in CSS themes. Corresponds with the font themes below.
	BuiltinThemes = map[string]string{
		"gray":   "@import url(//fonts.googleapis.com/css?family=Lato:300); body { background-color: #e7eaed; color: #0b0b0b; font-family: 'Lato', sans-serif; font-weight: 300;  margin: 4.5em; font-size: 1em; } a { color: #401010; font-family: courier; } a:hover { color: #801010; } a:active { color: yellow; } h1 { color: #101010; } img { max-width: 100%; } table { font-family: Helvetica, sans-serif; border-collapse: collapse; width: 100%; font-weight: bold; } table td, table th { border: 1px solid #ccc; padding: 0.45em 0.55em 0.45em 0.55em; } table tr:nth-child(even){background-color: #efefef; } table tr:nth-child(odd) { background-color: #fff; } table tr:hover { background-color: #e0f0ff; color: black; } table th { padding-top: 12px; padding-bottom: 12px; text-align: left; background-color: #666; color: white; } code { background-color: rgba(50, 50, 50, 0.1); color: #202020; border-radius: 0.3em; padding: 0.1em 0.4em 0.1em 0.4em; font-size: 90%; word-wrap: break-word; font-family: Menlo, Monaco, Consolas, \"Courier New\", monospace; }",
		"dark":   "@import url(//fonts.googleapis.com/css?family=Lato:400); body { background-color: #101010; color: #f0f0f0; font-family: 'Lato', sans-serif; font-weight: 400;  margin: 4.5em; font-size: 1em; } a { color: #c0a0a0; font-family: courier; } a:hover { color: #f0a0a0; } a:active { color: yellow; } h1 { color: #f0f0f0; } img { max-width: 100%; } table { font-family: Helvetica, sans-serif; border-collapse: collapse; width: 100%; font-weight: bold; } table td, table th { border: 1px solid #999; padding: 0.45em 0.55em 0.45em 0.55em; } table tr:nth-child(even){background-color: #222; } table tr:nth-child(odd) { background-color: #333; } table tr:hover { background-color: #333; color: #ff8060; } table th { padding-top: 12px; padding-bottom: 12px; text-align: left; background-color: #d7d7d7; color: #292929; } code { background-color: rgba(255, 255, 255, 0.18); color: #e0e0e0; border-radius: 0.3em; padding: 0.1em 0.4em 0.1em 0.4em; font-size: 90%; word-wrap: break-word; font-family: Menlo, Monaco, Consolas, \"Courier New\", monospace; }",
		"redbox": "@import url(//fonts.googleapis.com/css?family=Monoton|Monofett);html{background-color:#222;}body{color:#111;background-color:#999;font-family:Impact,'Arial Black',sans-serif;font-size:1.7em;margin:2.7em;padding:0 5em 1em 2em;border-radius:50px;border:solid 10px #a00;box-shadow:10px 10px 16px black, 6px 6px 8px #222 inset;}h2{font-family:Monoton,cursive;color:black;}ul{margin-left:1em;}a{text-decoration:none;color:#b00;}a:hover{color:#dc0;}code{font-family:Monofett,cursive;} img { max-width: 100%; } code { background-color: rgba(255, 255, 255, 0.18); color: #e0e0e0; border-radius: 0.3em; padding: 0.1em 0.4em 0.1em 0.4em; font-size: 90%; word-wrap: break-word; font-family: Menlo, Monaco, Consolas, \"Courier New\", monospace; }",
	}

	// Built in themes corresponding to highlight.js styles
	// See https://github.com/isagalaev/highlight.js/tree/master/src/styles for more styles
	DefaultCodeStyles = map[string]string{"gray": "color-brewer", "dark": "ocean", "redbox": "railscasts"}

	// A selection of allowed keywords for the HTML meta tag
	MetaKeywords = []string{"application-name", "author", "description", "generator", "keywords", "robots", "language", "googlebot", "Slurp", "bingbot", "geo.position", "geo.placename", "geo.region", "ICBM", "viewport"}
)

// Easy way to output a HTML page
func MessagePage(title, body, theme string) string {
	return fmt.Sprintf("<!doctype html><html><head><title>%s</title>%s<style>%s</style><head><body><h1>%s</h1>%s</body></html>", title, "", BuiltinThemes[theme], title, body)
}

// Easy way to build links to directories
func HTMLLink(text, url string, isDirectory bool) string {
	// Add a final slash, if needed
	if isDirectory {
		text += "/"
		url += "/"
	}
	return "<a href=\"/" + url + "\">" + text + "</a><br>"
}

// HTML to be added for enabling highlighting.
func HighlightHTML(codeStyle string) string {
	return `<link rel="stylesheet" href="//cdnjs.cloudflare.com/ajax/libs/highlight.js/9.6.0/styles/` + codeStyle + `.min.css"><script src="//cdnjs.cloudflare.com/ajax/libs/highlight.js/9.6.0/highlight.min.js"></script><script>hljs.initHighlightingOnLoad();</script>`
}

// Add a link to a stylesheet in the given Amber code
func LinkToStyle(amberdata *[]byte, url string) {
	// If the given url is not already mentioned and the data contains "body"
	if !bytes.Contains(*amberdata, []byte(url)) && bytes.Contains(*amberdata, []byte("html")) && bytes.Contains(*amberdata, []byte("body")) {
		// Extract one level of indendation
		whitespace := OneLevelOfIndentation(amberdata, "body")
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
func LinkToStyleHTML(htmldata *[]byte, url string) {
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
func AddMetaKeywords(keywords map[string]string) {
	for _, keyword := range MetaKeywords {
		keywords[keyword] = ""
	}
}

// Insert doctype in HTML, if missing
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
