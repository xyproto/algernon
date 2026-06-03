package themes

import (
	"strings"
	"testing"
)

func TestOneLevelOfIndentation(t *testing.T) {
	data := []byte("html\n  head\n    title\n  body\n    h1\n")
	got := OneLevelOfIndentation(&data, "body")
	if got != "  " {
		t.Errorf("OneLevelOfIndentation = %q, want %q", got, "  ")
	}

	// keyword not present
	got = OneLevelOfIndentation(&data, "footer")
	if got != "" {
		t.Errorf("OneLevelOfIndentation (missing keyword) = %q, want empty", got)
	}

	// tabs
	tabData := []byte("html\n\thead\n\t\ttitle\n\tbody\n")
	got = OneLevelOfIndentation(&tabData, "body")
	if got != "\t" {
		t.Errorf("OneLevelOfIndentation (tabs) = %q, want %q", got, "\t")
	}
}

func TestStyleHead(t *testing.T) {
	// Built-in theme should produce a <style> block
	got := string(StyleHead("dark"))
	if !strings.Contains(got, "<style>") {
		t.Error("StyleHead(dark) should contain <style>")
	}
	if strings.Contains(got, "<link") {
		t.Error("StyleHead(dark) should not contain <link>")
	}

	// CSS file theme should produce a <link> tag
	got = string(StyleHead("custom.css"))
	if !strings.Contains(got, "<link") {
		t.Error("StyleHead(custom.css) should contain <link>")
	}
	if !strings.Contains(got, "custom.css") {
		t.Error("StyleHead(custom.css) should reference the filename")
	}
}

func TestMessagePage(t *testing.T) {
	got := MessagePage("Test Title", "<p>hello</p>", "dark")
	if !strings.Contains(got, "<title>Test Title</title>") {
		t.Error("MessagePage should contain the title")
	}
	if !strings.Contains(got, "<h1>Test Title</h1>") {
		t.Error("MessagePage should contain the title as h1")
	}
	if !strings.Contains(got, "<p>hello</p>") {
		t.Error("MessagePage should contain the body")
	}
}

func TestMessagePageBytes(t *testing.T) {
	got := MessagePageBytes("Title", []byte("body content"), "dark")
	s := string(got)
	if !strings.Contains(s, "<title>Title</title>") {
		t.Error("MessagePageBytes should contain the title")
	}
	if !strings.Contains(s, "body content") {
		t.Error("MessagePageBytes should contain the body")
	}
}

func TestSimpleHTMLPage(t *testing.T) {
	got := string(SimpleHTMLPage([]byte("My Title"), []byte("Headline"), nil, []byte("<p>content</p>"), []byte("en")))
	if !strings.Contains(got, `lang="en"`) {
		t.Error("SimpleHTMLPage should contain language attribute")
	}
	if !strings.Contains(got, "<title>My Title</title>") {
		t.Error("SimpleHTMLPage should contain title")
	}
	if !strings.Contains(got, "<h1>Headline</h1>") {
		t.Error("SimpleHTMLPage should contain headline")
	}

	// Without language
	got = string(SimpleHTMLPage([]byte("T"), nil, nil, []byte("b"), nil))
	if strings.Contains(got, "lang=") {
		t.Error("SimpleHTMLPage without language should not have lang attribute")
	}
}

func TestHTMLLink(t *testing.T) {
	got := HTMLLink("docs", "docs", true)
	if !strings.Contains(got, "docs/") {
		t.Error("HTMLLink for directory should append slash to text")
	}
	if !strings.Contains(got, `href="/docs/"`) {
		t.Error("HTMLLink for directory should append slash to URL")
	}

	got = HTMLLink("file.txt", "file.txt", false)
	if strings.HasSuffix(got, "/") {
		t.Error("HTMLLink for file should not append slash")
	}
}

func TestStyleHTML(t *testing.T) {
	html := []byte("<html><head></head><body><p>hi</p></body></html>")
	got := StyleHTML(html, "style.css")
	if !strings.Contains(string(got), `href="style.css"`) {
		t.Error("StyleHTML should inject stylesheet link")
	}

	// Already present: should not duplicate
	got2 := StyleHTML(got, "style.css")
	if strings.Count(string(got2), "style.css") != 1 {
		t.Error("StyleHTML should not duplicate existing stylesheet")
	}
}

func TestInsertDoctype(t *testing.T) {
	// Missing doctype
	html := []byte("<html>\n<head>\n<title>hi</title>\n</head>\n<body></body></html>")
	got := InsertDoctype(html)
	if !strings.HasPrefix(string(got), "<!doctype html>") {
		t.Error("InsertDoctype should prepend doctype")
	}

	// Already has doctype
	html2 := []byte("<!DOCTYPE html>\n<html>\n<body></body></html>")
	got2 := InsertDoctype(html2)
	if strings.Count(string(got2), "doctype") > 1 && strings.Count(string(got2), "DOCTYPE") > 1 {
		t.Error("InsertDoctype should not add doctype if already present")
	}
}

func TestNoPage(t *testing.T) {
	got := string(NoPage("missing.html", "dark"))
	if !strings.Contains(got, "missing.html") {
		t.Error("NoPage should contain the filename")
	}
	if !strings.Contains(got, "Not found") {
		t.Error("NoPage should contain 'Not found'")
	}
}
