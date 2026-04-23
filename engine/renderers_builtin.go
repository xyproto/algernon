package engine

import (
	"fmt"
	"net/http"
)

// Built-in renderers for FilePage, each delegating to an existing *Page method on *Config

// markdownRenderer handles ".md" and ".markdown"
type markdownRenderer struct{}

func (markdownRenderer) Extensions() []string { return []string{".md", ".markdown"} }

func (markdownRenderer) Render(ac *Config, w http.ResponseWriter, req *http.Request, filename, ext string) error {
	w.Header().Add(contentType, htmlUTF8)
	mdblock, err := ac.ReadAndLogErrors(w, filename, ext)
	if err != nil {
		return nil
	}
	ac.MarkdownPage(w, req, mdblock.Bytes(), filename)
	return nil
}

// gcssRenderer handles ".gcss" (GCSS compiled to CSS)
type gcssRenderer struct{}

func (gcssRenderer) Extensions() []string { return []string{".gcss"} }

func (gcssRenderer) Render(ac *Config, w http.ResponseWriter, req *http.Request, filename, ext string) error {
	gcssblock, err := ac.ReadAndLogErrors(w, filename, ext)
	if err != nil {
		return nil
	}
	w.Header().Add(contentType, "text/css;charset=utf-8")
	ac.GCSSPage(w, req, filename, gcssblock.Bytes())
	return nil
}

// scssRenderer handles ".scss" (Sass compiled to CSS)
type scssRenderer struct{}

func (scssRenderer) Extensions() []string { return []string{".scss"} }

func (scssRenderer) Render(ac *Config, w http.ResponseWriter, req *http.Request, filename, ext string) error {
	scssblock, err := ac.ReadAndLogErrors(w, filename, ext)
	if err != nil {
		return nil
	}
	w.Header().Add(contentType, "text/css;charset=utf-8")
	ac.SCSSPage(w, req, filename, scssblock.Bytes())
	return nil
}

// jsxRenderer handles plain ".jsx". FilePage rewrites ext to ".hyper.js"/".hyper.jsx"
// before dispatch, so those variants land on hyperAppRenderer instead.
type jsxRenderer struct{}

func (jsxRenderer) Extensions() []string { return []string{".jsx"} }

func (jsxRenderer) Render(ac *Config, w http.ResponseWriter, req *http.Request, filename, ext string) error {
	jsxblock, err := ac.ReadAndLogErrors(w, filename, ext)
	if err != nil {
		return fmt.Errorf("%s:%s", filename, err.Error())
	}
	w.Header().Add(contentType, "text/javascript;charset=utf-8")
	ac.JSXPage(w, req, filename, jsxblock.Bytes())
	return nil
}

// hyperAppRenderer handles HyperApp JSX/JS wrappers: .happ, .hyper, .hyper.jsx, .hyper.js
type hyperAppRenderer struct{}

func (hyperAppRenderer) Extensions() []string {
	return []string{".happ", ".hyper", ".hyper.jsx", ".hyper.js"}
}

func (hyperAppRenderer) Render(ac *Config, w http.ResponseWriter, req *http.Request, filename, ext string) error {
	jsxblock, err := ac.ReadAndLogErrors(w, filename, ext)
	if err != nil {
		return fmt.Errorf("%s:%s", filename, err.Error())
	}
	w.Header().Add(contentType, htmlUTF8)
	ac.HyperAppPage(w, req, filename, jsxblock.Bytes())
	return nil
}

func init() {
	defaultRenderers.register(markdownRenderer{})
	defaultRenderers.register(gcssRenderer{})
	defaultRenderers.register(scssRenderer{})
	defaultRenderers.register(jsxRenderer{})
	defaultRenderers.register(hyperAppRenderer{})
}
