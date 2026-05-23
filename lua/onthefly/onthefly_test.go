package onthefly

import (
	"strings"
	"testing"

	lua "github.com/xyproto/gopher-lua"
)

// runLua loads onthefly into a fresh Lua state and executes the given script.
// Returns the last printed string (via a captured `out` global) if the script
// stored one in the `out` global, otherwise the error.
func runLua(t *testing.T, script string) string {
	t.Helper()
	L := lua.NewState()
	defer L.Close()
	Load(L)
	if err := L.DoString(script); err != nil {
		t.Fatalf("lua error: %v", err)
	}
	return L.GetGlobal("out").String()
}

func TestHTML5Basic(t *testing.T) {
	html := runLua(t, `
		local page = HTML5("Hi")
		out = tostring(page)
	`)
	for _, want := range []string{"<!doctype html>", "<html>", "<title>Hi</title>", "<body"} {
		if !strings.Contains(html, want) {
			t.Errorf("expected %q in output, got:\n%s", want, html)
		}
	}
}

func TestTagAttributesAndContent(t *testing.T) {
	html := runLua(t, `
		local page = HTML5("X")
		local body = page:tag("body")
		body:addContent("Hello")
		local p = body:addNewTag("p")
		p:addAttrib("class", "lead")
		p:addContent("World")
		out = page:html()
	`)
	for _, want := range []string{`class="lead"`, "Hello", "World"} {
		if !strings.Contains(html, want) {
			t.Errorf("expected %q in output, got:\n%s", want, html)
		}
	}
}

func TestPageCSS(t *testing.T) {
	css := runLua(t, `
		local page = HTML5("X")
		local body = page:tag("body")
		body:addStyle("margin", "2em")
		body:addStyle("color", "white")
		body:addStyle("background-color", "black")
		out = page:css()
	`)
	for _, want := range []string{"margin: 2em", "color: white", "background-color: black"} {
		if !strings.Contains(css, want) {
			t.Errorf("expected %q in CSS, got:\n%s", want, css)
		}
	}
}

func TestScriptInjection(t *testing.T) {
	html := runLua(t, `
		local page = HTML5("X")
		page:addScriptToHead("var x = 1;")
		out = page:html()
	`)
	if !strings.Contains(html, "var x = 1;") {
		t.Errorf("expected injected script, got:\n%s", html)
	}
	if !strings.Contains(html, `type="text/javascript"`) {
		t.Errorf("expected script type attribute, got:\n%s", html)
	}
}

func TestTagSearch(t *testing.T) {
	out := runLua(t, `
		local page = HTML5("X")
		local body = page:tag("body")
		local d = body:addNewTag("div")
		d:addAttrib("id", "main")
		local found = body:findChildByAttribute("id", "main")
		out = found:name()
	`)
	if out != "div" {
		t.Errorf("expected div, got %q", out)
	}
}

func TestChaining(t *testing.T) {
	// Tag setters return self
	html := runLua(t, `
		local page = HTML5("X")
		page:tag("body")
			:addNewTag("p")
				:addAttrib("class", "lead")
				:addAttrib("id", "p1")
				:addContent("Hello")
				:appendContent(" World")
		out = page:html()
	`)
	for _, want := range []string{`class="lead"`, `id="p1"`, "Hello World"} {
		if !strings.Contains(html, want) {
			t.Errorf("expected %q in output, got:\n%s", want, html)
		}
	}

	// Page setters return self
	html = runLua(t, `
		local page = HTML5("X")
		page:metaCharset("UTF-8"):linkToCSS("/s.css"):linkToJS("/a.js")
		out = page:html()
	`)
	for _, want := range []string{"charset=UTF-8", `href="/s.css"`, `src="/a.js"`} {
		if !strings.Contains(html, want) {
			t.Errorf("expected %q in output, got:\n%s", want, html)
		}
	}
}

// Backwards compatibility: the original 7-element API surface.
func TestBackwardsCompat(t *testing.T) {
	// Page(title, root)
	out := runLua(t, `
		local p = Page("T", "root")
		out = tostring(p)
	`)
	if !strings.Contains(out, "<root") {
		t.Errorf("Page(title, root) regression: %s", out)
	}
	// Tag(name) + addNewTag
	out = runLua(t, `
		local t = Tag("a")
		local b = t:addNewTag("b")
		out = tostring(t)
	`)
	if !strings.Contains(out, "<a>") || !strings.Contains(out, "<b") {
		t.Errorf("Tag/addNewTag regression: %s", out)
	}
}
