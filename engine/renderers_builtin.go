package engine

import (
	"fmt"
	"net/http"
	"path/filepath"
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

// jsxRenderer handles plain ".jsx". When the file is named "index.jsx", it is
// served as a full React page. The React version is read from an optional
// "// React: <N>" comment at the top of the source file, defaulting to
// defaultReactVersion.
type jsxRenderer struct{}

func (jsxRenderer) Extensions() []string { return []string{".jsx"} }

func (jsxRenderer) Render(ac *Config, w http.ResponseWriter, req *http.Request, filename, ext string) error {
	jsxblock, err := ac.ReadAndLogErrors(w, filename, ext)
	if err != nil {
		return fmt.Errorf("%s:%s", filename, err.Error())
	}

	// Serve index.jsx as a full React HTML page
	if filepath.Base(filename) == "index.jsx" {
		ver := defaultReactVersion
		if v := parseReactVersion(jsxblock.Bytes()); v > 0 {
			ver = v
		}
		w.Header().Add(contentType, htmlUTF8)
		ac.ReactPage(w, req, filename, jsxblock.Bytes(), ver)
		return nil
	}

	w.Header().Add(contentType, "text/javascript;charset=utf-8")
	ac.JSXPage(w, req, filename, jsxblock.Bytes())
	return nil
}

// tsxRenderer handles ".ts" and ".tsx" (TypeScript compiled to JavaScript).
// When the file is named "index.tsx", it is served as a full React page.
// The React version is read from an optional "// React: <N>" comment at the
// top of the source file, defaulting to defaultReactVersion.
type tsxRenderer struct{}

func (tsxRenderer) Extensions() []string { return []string{".ts", ".tsx"} }

func (tsxRenderer) Render(ac *Config, w http.ResponseWriter, req *http.Request, filename, ext string) error {
	tsxblock, err := ac.ReadAndLogErrors(w, filename, ext)
	if err != nil {
		return fmt.Errorf("%s:%s", filename, err.Error())
	}

	// Serve index.tsx as a full React HTML page
	if filepath.Base(filename) == "index.tsx" {
		ver := defaultReactVersion
		if v := parseReactVersion(tsxblock.Bytes()); v > 0 {
			ver = v
		}
		w.Header().Add(contentType, htmlUTF8)
		ac.ReactPage(w, req, filename, tsxblock.Bytes(), ver)
		return nil
	}

	w.Header().Add(contentType, "text/javascript;charset=utf-8")
	ac.TSXPage(w, req, filename, tsxblock.Bytes())
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
	defaultRenderers.register(tsxRenderer{})
	defaultRenderers.register(hyperAppRenderer{})
}
