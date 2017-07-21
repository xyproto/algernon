package utils

import (
	"bytes"
	"fmt"
	"strings"
)

const (
	// DefaultStyleFilename is the default stylesheet filename (GCSS)
	DefaultStyleFilename = "style.gcss"

	// DefaultCustomCodeStyle is the default highlight.js style for custom CSS
	DefaultCustomCodeStyle = "github"

	// HighlightJSversion is the version to be used for highlight.js.
	// See https://highlightjs.org/ for the latest version number.
	HighlightJSversion = "9.12.0"
)

var (
	// BuiltinThemes is a map over the available built-in CSS themes. Corresponds with the font themes below.
	// "default" and "gray" are equal. "default" should never be used directly, but is here as a safeguard.
	BuiltinThemes = map[string][]byte{
		"default": []byte("@import url(//fonts.googleapis.com/css?family=Lato:300); body { background-color: #e7eaed; color: #0b0b0b; font-family: 'Lato', sans-serif; font-weight: 300;  margin: 4.5em; font-size: 1em; } a { color: #401010; font-family: courier; } a:hover { color: #801010; } a:active { color: yellow; } h1 { color: #101010; } img { max-width: 100%; } table { font-family: Helvetica, sans-serif; border-collapse: collapse; width: 100%; font-weight: bold; } table td, table th { border: 1px solid #ccc; padding: 0.45em 0.55em 0.45em 0.55em; } table tr:nth-child(even){background-color: #efefef; } table tr:nth-child(odd) { background-color: #fff; } table tr:hover { background-color: #e0f0ff; color: black; } table th { padding-top: 12px; padding-bottom: 12px; text-align: left; background-color: #666; color: white; } code { background-color: rgba(50, 50, 50, 0.1); color: #202020; border-radius: 0.3em; padding: 0.1em 0.4em 0.1em 0.4em; font-size: 90%; word-wrap: break-word; font-family: Menlo, Monaco, Consolas, \"Courier New\", monospace; }"),
		"gray":    []byte("@import url(//fonts.googleapis.com/css?family=Lato:300); body { background-color: #e7eaed; color: #0b0b0b; font-family: 'Lato', sans-serif; font-weight: 300;  margin: 4.5em; font-size: 1em; } a { color: #401010; font-family: courier; } a:hover { color: #801010; } a:active { color: yellow; } h1 { color: #101010; } img { max-width: 100%; } table { font-family: Helvetica, sans-serif; border-collapse: collapse; width: 100%; font-weight: bold; } table td, table th { border: 1px solid #ccc; padding: 0.45em 0.55em 0.45em 0.55em; } table tr:nth-child(even){background-color: #efefef; } table tr:nth-child(odd) { background-color: #fff; } table tr:hover { background-color: #e0f0ff; color: black; } table th { padding-top: 12px; padding-bottom: 12px; text-align: left; background-color: #666; color: white; } code { background-color: rgba(50, 50, 50, 0.1); color: #202020; border-radius: 0.3em; padding: 0.1em 0.4em 0.1em 0.4em; font-size: 90%; word-wrap: break-word; font-family: Menlo, Monaco, Consolas, \"Courier New\", monospace; }"),
		"dark":    []byte("@import url(//fonts.googleapis.com/css?family=Lato:400); body { background-color: #101010; color: #f0f0f0; font-family: 'Lato', sans-serif; font-weight: 400;  margin: 4.5em; font-size: 1em; } a { color: #c0a0a0; font-family: courier; } a:hover { color: #f0a0a0; } a:active { color: yellow; } h1 { color: #f0f0f0; } img { max-width: 100%; } table { font-family: Helvetica, sans-serif; border-collapse: collapse; width: 100%; font-weight: bold; } table td, table th { border: 1px solid #999; padding: 0.45em 0.55em 0.45em 0.55em; } table tr:nth-child(even){background-color: #222; } table tr:nth-child(odd) { background-color: #333; } table tr:hover { background-color: #333; color: #ff8060; } table th { padding-top: 12px; padding-bottom: 12px; text-align: left; background-color: #d7d7d7; color: #292929; } code { background-color: rgba(255, 255, 255, 0.18); color: #e0e0e0; border-radius: 0.3em; padding: 0.1em 0.4em 0.1em 0.4em; font-size: 90%; word-wrap: break-word; font-family: Menlo, Monaco, Consolas, \"Courier New\", monospace; }"),
		"redbox":  []byte("@import url(//fonts.googleapis.com/css?family=Monoton|Monofett);html{background-color:#222;}body{color:#111;background-color:#999;font-family:Impact,'Arial Black',sans-serif;font-size:1.7em;margin:2.7em;padding:0 5em 1em 2em;border-radius:50px;border:solid 10px #a00;box-shadow:10px 10px 16px black, 6px 6px 8px #222 inset;}h2{font-family:Monoton,cursive;color:black;}ul{margin-left:1em;}a{text-decoration:none;color:#b00;}a:hover{color:#dc0;}code{font-family:Monofett,cursive;} img { max-width: 100%; } code { background-color: rgba(255, 255, 255, 0.18); color: #e0e0e0; border-radius: 0.3em; padding: 0.1em 0.4em 0.1em 0.4em; font-size: 90%; word-wrap: break-word; font-family: Menlo, Monaco, Consolas, \"Courier New\", monospace; }"),
		"bw":      []byte("body { font-family: BlinkMacSystemFont, \"Segoe UI\", Helvetica, sans-serif; margin: 1px; padding: 1px; } button { background: black; border-radius: 4px; color: #ffffff; font-size: 20px; padding: 10px 20px 10px 20px; border: none; outline: none; margin-left: 2px; margin-right: 2px; } button:hover { background: orange; text-decoration: none; } button:disabled { background: gray; color: #eee; } input { font-size: 16px; padding: 10px; border: 1px solid #999; } a { padding: 1px; text-decoration: none; color: black; } a:hover { padding: 1px; border-bottom: 1px solid black; } a.button { font-size: 1em; border: 1px solid black; background-color: white; color: black; width: 150px; padding: 10px 0px; margin-right: 10px; margin-top: 5px; margin-bottom: 5px; text-align: center; display: inline-block; } a.button:hover { text-decoration: none; color: white; background-color: black;   } a.button:active { color: white; background-color: black;  } a.go-back { position: absolute; right: 10px; top: 10px; padding: 10px; font-size: 0.8em; background-color: black; color: white; } .ha { box-sizing: border-box; margin: 0 150px; padding: 20px; line-height: 1.5; font-size: 15.5px; line-height: 1.55; } .ha ul { list-style: none; padding: 0 !important; column-count: 2; max-width: 0; column-gap: 20px; min-width: 350px; } .ha ul li { margin-left: 0px; } .ha h1 { margin-top: 0; margin-bottom: 0; } .ha h1 { font-size: 32px; font-weight: 200; margin-top: 0px; margin-bottom: 16px; } .ha h5 { margin-bottom: 2px; } .ha hr { padding: 0; margin: 24px 0; background-color: #e7e7e7; border: 0; height: 0.09em; } .ha li { margin-top: 0.25em; } .gh-btns { float: right; margin-top: -4px; }"), // from MIT licensed HyperApp: hyperapp.glitch.me/style.css
	}

	// DefaultCodeStyles is a map of the themes names corresponding to highlight.js styles
	// See https://github.com/isagalaev/highlight.js/tree/master/src/styles for more styles
	DefaultCodeStyles = map[string]string{"gray": "color-brewer", "dark": "ocean", "redbox": "railscasts", "bw": "github"}

	// MetaKeywords contains a selection of allowed keywords for the HTML meta tag
	MetaKeywords = []string{"application-name", "author", "description", "generator", "keywords", "robots", "language", "googlebot", "Slurp", "bingbot", "geo.position", "geo.placename", "geo.region", "ICBM", "viewport"}
)

// MessagePage is an easy way to output a HTML page only given a title, the body
// (will be placed between the <body></body> tags) and the name of one of the
// built-in themes.
// Deprecated
func MessagePage(title, body, theme string) string {
	return fmt.Sprintf("<!doctype html><html><head><title>%s</title>%s<style>%s</style><head><body><h1>%s</h1>%s</body></html>", title, "", string(BuiltinThemes[theme]), title, body)
}

// MessagePageBytes provides the same functionalityt as MessagePage,
// but with []byte instead of string, and without closing </body></html>
func MessagePageBytes(title string, body []byte, theme string) []byte {
	var buf bytes.Buffer
	buf.WriteString("<!doctype html><html><head><title>")
	buf.WriteString(title)
	buf.WriteString("</title><style>")
	buf.Write(BuiltinThemes[theme])
	buf.WriteString("</style><head><body><h1>")
	buf.WriteString(title)
	buf.WriteString("</h1>")
	buf.Write(body)
	return buf.Bytes()
}

// SimpleHTMLPage provides a quick way to build a HTML page
func SimpleHTMLPage(title, headline, inhead, body []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString("<!doctype html><html><head><title>")
	buf.Write(title)
	buf.WriteString("</title>")
	buf.Write(inhead)
	buf.WriteString("<head><body><h1>")
	buf.Write(headline)
	buf.WriteString("</h1>")
	buf.Write(body)
	return buf.Bytes()
}

// HTMLLink builds an HTML link given the link text, the URL to a file/directory
// and a boolean that is true if the given URL is to a directory.
func HTMLLink(text, url string, isDirectory bool) string {
	// Add a final slash, if needed
	if isDirectory {
		text += "/"
		url += "/"
	}
	return "<a href=\"/" + url + "\">" + text + "</a><br>"
}

// HighlightHTML creates the HTML code for linking with a stylesheet for
// highlighting <code> tags with the given highlight.js code style
func HighlightHTML(codeStyle string) string {
	return `<link rel="stylesheet" href="//cdnjs.cloudflare.com/ajax/libs/highlight.js/` + HighlightJSversion + `/styles/` + codeStyle + `.min.css"><script src="//cdnjs.cloudflare.com/ajax/libs/highlight.js/` + HighlightJSversion + `/highlight.min.js"></script><script>hljs.initHighlightingOnLoad();</script>`
}

// StyleAmber modifies Amber source code so that a link to the given stylesheet URL is added
func StyleAmber(amberdata *[]byte, url string) {
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

// StyleHTML modifies HTML source code so that a link to the given stylesheet URL is added
func StyleHTML(htmldata *[]byte, url string) {
	// If the given url is not already mentioned and the data contains "body"
	if !bytes.Contains(*htmldata, []byte(url)) && bytes.Contains(*htmldata, []byte("body")) {
		if bytes.Contains(*htmldata, []byte("</head>")) {
			*htmldata = bytes.Replace(*htmldata, []byte("</head>"), []byte("  <link rel=\"stylesheet\" href=\""+url+"\">\n  </head>"), 1)
		} else if bytes.Contains(*htmldata, []byte("<body>")) {
			*htmldata = bytes.Replace(*htmldata, []byte("<body>"), []byte("  <head>\n  <link rel=\"stylesheet\" href=\""+url+"\">\n  </head>\n  <body>"), 1)
		}
	}
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
