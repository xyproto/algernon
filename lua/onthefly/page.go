package onthefly

import (
	"github.com/sirupsen/logrus"
	lua "github.com/xyproto/gopher-lua"
)

// Return the Page as an HTML string
func pageString(L *lua.LState) int {
	page := checkPage(L) // arg 1
	L.Push(lua.LString(page.String()))
	return 1 // number of results
}

// Take a Page and a boolean indicating if the XML should be indented.
// Returns the XML rendering of the Page.
func pageGetXML(L *lua.LState) int {
	page := checkPage(L) // arg 1
	L.Push(lua.LString(page.GetXML(L.ToBool(2))))
	return 1 // number of results
}

// Return the HTML rendering of the Page.
func pageGetHTML(L *lua.LState) int {
	page := checkPage(L) // arg 1
	L.Push(lua.LString(page.GetHTML()))
	return 1 // number of results
}

// Return the CSS rendering of the Page.
func pageGetCSS(L *lua.LState) int {
	page := checkPage(L) // arg 1
	L.Push(lua.LString(page.GetCSS()))
	return 1 // number of results
}

// Return the root Tag of the Page.
func pageGetRoot(L *lua.LState) int {
	page := checkPage(L) // arg 1
	pushTag(L, page.GetRoot())
	return 1 // number of results
}

// Take a Page and a tag name. Returns the first matching Tag or nil.
func pageGetTag(L *lua.LState) int {
	page := checkPage(L) // arg 1
	name := L.ToString(2)
	if name == "" {
		L.ArgError(2, "tag name expected")
		return 0 // no results
	}
	tag, err := page.GetTag(name)
	if err != nil {
		L.Push(lua.LNil)
		return 1 // number of results
	}
	pushTag(L, tag)
	return 1 // number of results
}

// Take a Page and content to add to the <body> tag.
// Returns the <body> Tag or nil if no body could be found.
func pageAddContent(L *lua.LState) int {
	page := checkPage(L) // arg 1
	body, err := page.AddContent(L.ToString(2))
	if err != nil {
		logrus.Error(err)
		L.Push(lua.LNil)
		return 1 // number of results
	}
	pushTag(L, body)
	return 1 // number of results
}

// Take a Page and a URL to a CSS file. Adds a <link> in the <head> tag.
// Returns the Page.
func pageLinkToCSS(L *lua.LState) int {
	page := checkPage(L) // arg 1
	if err := page.LinkToCSS(L.ToString(2)); err != nil {
		logrus.Error(err)
	}
	return pushSelf(L)
}

// Take a Page and a charset name (e.g. "UTF-8").
// Adds a <meta> tag in the <head> tag. Returns the Page.
func pageMetaCharset(L *lua.LState) int {
	page := checkPage(L) // arg 1
	if err := page.MetaCharset(L.ToString(2)); err != nil {
		logrus.Error(err)
	}
	return pushSelf(L)
}

// Take a Page and a URL to a JavaScript file.
// Links to the JS file from the <head> tag. Returns the Page.
func pageLinkToJS(L *lua.LState) int {
	page := checkPage(L) // arg 1
	if err := page.LinkToJS(L.ToString(2)); err != nil {
		logrus.Error(err)
	}
	return pushSelf(L)
}

// Take a Page and a URL to a JavaScript file.
// Links to the JS file from the <head> tag. Returns the new <script> Tag or nil.
func pageLinkToJSInHead(L *lua.LState) int {
	page := checkPage(L) // arg 1
	tag, err := page.LinkToJSInHead(L.ToString(2))
	if err != nil {
		logrus.Error(err)
		L.Push(lua.LNil)
		return 1 // number of results
	}
	pushTag(L, tag)
	return 1 // number of results
}

// Take a Page and a URL to a JavaScript file.
// Links to the JS file from the <body> tag. Returns the new <script> Tag or nil.
func pageLinkToJSInBody(L *lua.LState) int {
	page := checkPage(L) // arg 1
	tag, err := page.LinkToJSInBody(L.ToString(2))
	if err != nil {
		logrus.Error(err)
		L.Push(lua.LNil)
		return 1 // number of results
	}
	pushTag(L, tag)
	return 1 // number of results
}

// Take a Page and a JavaScript source string.
// Adds the JS in a new <script> tag in the <head>. Returns the new Tag or nil.
func pageAddScriptToHead(L *lua.LState) int {
	page := checkPage(L) // arg 1
	tag, err := page.AddScriptToHead(L.ToString(2))
	if err != nil {
		logrus.Error(err)
		L.Push(lua.LNil)
		return 1 // number of results
	}
	pushTag(L, tag)
	return 1 // number of results
}

// Take a Page and a JavaScript source string.
// Adds the JS in a new <script> tag at the end of the <body>.
// Returns the new Tag or nil.
func pageAddScriptToBody(L *lua.LState) int {
	page := checkPage(L) // arg 1
	tag, err := page.AddScriptToBody(L.ToString(2))
	if err != nil {
		logrus.Error(err)
		L.Push(lua.LNil)
		return 1 // number of results
	}
	pushTag(L, tag)
	return 1 // number of results
}

// Take a Page and a CSS source string.
// Adds the CSS in a new <style> tag in the <head>. Returns the new Tag or nil.
func pageAddStyle(L *lua.LState) int {
	page := checkPage(L) // arg 1
	tag, err := page.AddStyle(L.ToString(2))
	if err != nil {
		logrus.Error(err)
		L.Push(lua.LNil)
		return 1 // number of results
	}
	pushTag(L, tag)
	return 1 // number of results
}

// The Page method table that is to be registered
var pageMethods = map[string]lua.LGFunction{
	"__tostring":      pageString,
	"xml":             pageGetXML,
	"html":            pageGetHTML,
	"css":             pageGetCSS,
	"root":            pageGetRoot,
	"tag":             pageGetTag,
	"addContent":      pageAddContent,
	"linkToCSS":       pageLinkToCSS,
	"metaCharset":     pageMetaCharset,
	"linkToJS":        pageLinkToJS,
	"linkToJSInHead":  pageLinkToJSInHead,
	"linkToJSInBody":  pageLinkToJSInBody,
	"addScriptToHead": pageAddScriptToHead,
	"addScriptToBody": pageAddScriptToBody,
	"addStyle":        pageAddStyle,
}
