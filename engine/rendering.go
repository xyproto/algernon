package engine

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/eknkc/amber"
	"github.com/evanw/esbuild/pkg/api"
	"github.com/flosch/pongo2/v6"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
	log "github.com/sirupsen/logrus"
	"github.com/wellington/sass/compiler"
	"github.com/xyproto/algernon/console"
	"github.com/xyproto/algernon/lua/convert"
	"github.com/xyproto/algernon/themes"
	"github.com/xyproto/algernon/utils"
	lua "github.com/xyproto/gopher-lua"
	"github.com/xyproto/splash"
	"github.com/yosssi/gcss"

	_ "embed"
)

const markdownCodeStyle = "base16-snazzy" // using xyproto/splash

var (
	//go:embed embedded/tex-svg.js
	mathJaxScript string

	formulaPattern = regexp.MustCompile(`(?s)\$\$.*?\$\$|\\\(.*?\\\)|\\\[.*?\\\]`)

	// Available Markdown extensions: https://github.com/gomarkdown/markdown/blob/master/parser/parser.go#L20
	enabledMarkdownExtensions = parser.NoIntraEmphasis | parser.Tables | parser.FencedCode | parser.Autolink | parser.Strikethrough | parser.SpaceHeadings | parser.HeadingIDs | parser.BackslashLineBreak | parser.DefinitionLists | parser.AutoHeadingIDs | parser.Mmark | parser.BackslashLineBreak | parser.MathJax
)

// containsFormula checks if the given Markdown content contains at least one mathematical formula (LaTeX style)
func containsFormula(mdContent []byte) bool {
	return formulaPattern.Match(mdContent)
}

// ValidGCSS checks if the given data is valid GCSS.
// The error value is returned on the channel.
func ValidGCSS(gcssdata []byte, errorReturn chan error) {
	buf := bytes.NewBuffer(gcssdata)
	var w bytes.Buffer
	_, err := gcss.Compile(&w, buf)
	errorReturn <- err
}

// LoadRenderFunctions adds functions related to rendering text to the given
// Lua state struct
func (ac *Config) LoadRenderFunctions(w http.ResponseWriter, _ *http.Request, L *lua.LState) {

	// Output Markdown as HTML
	L.SetGlobal("mprint", L.NewFunction(func(L *lua.LState) int {
		// Retrieve all the function arguments as a bytes.Buffer
		buf := convert.Arguments2buffer(L, true)
		// Create a Markdown parser with the desired extensions
		extensions := parser.CommonExtensions | parser.AutoHeadingIDs
		mdParser := parser.NewWithExtensions(extensions)
		mdContent := buf.Bytes()
		// Convert the buffer from Markdown to HTML
		htmlData := markdown.ToHTML(mdContent, mdParser, nil)

		// Apply syntax highlighting
		if highlightedHTML, err := splash.Splash(htmlData, markdownCodeStyle); err == nil { // success
			htmlData = highlightedHTML
		}

		// Add a script for rendering MathJax, but only if at least one mathematical formula is present
		if containsFormula(mdContent) {
			js := append([]byte(`<script id="MathJax-script">`), []byte(mathJaxScript)...)
			htmlData = InsertScriptTag(htmlData, js) // also adds the closing </script> tag
		}

		w.Write(htmlData)
		return 0 // number of results
	}))

	// Output text as rendered amber.
	L.SetGlobal("aprint", L.NewFunction(func(L *lua.LState) int {
		// Retrieve all the function arguments as a bytes.Buffer
		buf := convert.Arguments2buffer(L, true)
		// Use the buffer as a template.
		// Options are "Pretty printing, but without line numbers."
		tpl, err := amber.Compile(buf.String(), amber.Options{PrettyPrint: true, LineNumbers: false})
		if err != nil {
			if ac.debugMode {
				fmt.Fprint(w, "Could not compile Amber template:\n\t"+err.Error()+"\n\n"+buf.String())
			} else {
				log.Errorf("Could not compile Amber template:\n%s\n%s", err, buf.String())
			}
			return 0 // number of results
		}
		// Using "MISSING" instead of nil for a slightly better error message
		// if the values in the template should not be found.
		tpl.Execute(w, "MISSING")
		return 0 // number of results
	}))

	// Output text as rendered Pongo2
	L.SetGlobal("poprint", L.NewFunction(func(L *lua.LState) int {
		pongoMap := make(pongo2.Context)

		// Use the first argument as the template and the second argument as the data map
		templateString := L.CheckString(1)

		// If a table is given as the second argument, fill pongoMap with keys and values
		if L.GetTop() >= 2 {
			mapSS, mapSI, _, _ := convert.Table2maps(L.CheckTable(2))
			for k, v := range mapSI {
				pongoMap[k] = v
			}
			for k, v := range mapSS {
				pongoMap[k] = v
			}
		}

		// Retrieve all the function arguments as a bytes.Buffer
		buf := convert.Arguments2buffer(L, true)
		// Use the buffer as a template.
		// Options are "Pretty printing, but without line numbers."
		tpl, err := pongo2.FromString(templateString)
		if err != nil {
			if ac.debugMode {
				fmt.Fprint(w, "Could not compile Pongo2 template:\n\t"+err.Error()+"\n\n"+buf.String())
			} else {
				log.Errorf("Could not compile Pongo2 template:\n%s\n%s", err, buf.String())
			}
			return 0 // number of results
		}
		// nil is the template context (variables etc in a map)
		if err := tpl.ExecuteWriter(pongoMap, w); err != nil {
			if ac.debugMode {
				fmt.Fprint(w, "Could not compile Pongo2:\n\t"+err.Error()+"\n\n"+buf.String())
			} else {
				log.Errorf("Could not compile Pongo2:\n%s\n%s", err, buf.String())
			}
		}
		return 0 // number of results
	}))

	// Output text as rendered GCSS
	L.SetGlobal("gprint", L.NewFunction(func(L *lua.LState) int {
		// Retrieve all the function arguments as a bytes.Buffer
		buf := convert.Arguments2buffer(L, true)
		// Transform GCSS to CSS and output the result.
		// Ignoring the number of bytes written.
		if _, err := gcss.Compile(w, &buf); err != nil {
			if ac.debugMode {
				fmt.Fprint(w, "Could not compile GCSS:\n\t"+err.Error()+"\n\n"+buf.String())
			} else {
				log.Errorf("Could not compile GCSS:\n%s\n%s", err, buf.String())
			}

			// return 0 // number of results
		}
		return 0 // number of results
	}))

	// Output text as rendered JSX for React
	L.SetGlobal("jprint", L.NewFunction(func(L *lua.LState) int {
		// Retrieve all the function arguments as a bytes.Buffer
		buf := convert.Arguments2buffer(L, true)
		// Transform JSX to JavaScript and output the result.
		result := api.Transform(buf.String(), ac.jsxOptions)
		if len(result.Errors) > 0 {
			if ac.debugMode {
				// TODO: Use a similar error page as for Lua
				for _, errMsg := range result.Errors {
					fmt.Fprintf(w, "error: %s %d:%d\n", errMsg.Text, errMsg.Location.Line, errMsg.Location.Column)
				}
				for _, warnMsg := range result.Warnings {
					fmt.Fprintf(w, "warning: %s %d:%d\n", warnMsg.Text, warnMsg.Location.Line, warnMsg.Location.Column)
				}
			} else {
				// TODO: Use a similar error page as for Lua
				for _, errMsg := range result.Errors {
					log.Errorf("error: %s %d:%d\n", errMsg.Text, errMsg.Location.Line, errMsg.Location.Column)
				}
				for _, warnMsg := range result.Warnings {
					log.Errorf("warning: %s %d:%d\n", warnMsg.Text, warnMsg.Location.Line, warnMsg.Location.Column)
				}
			}
			return 0 // number of results

		}
		n, err := w.Write(result.Code)
		if err != nil || n == 0 {
			if ac.debugMode {
				fmt.Fprint(w, "Result from generated JavaScript is empty\n")
			} else {
				log.Error("Result from generated JavaScript is empty\n")
			}
			return 0 // number of results
		}

		return 0 // number of results
	}))

	// Output text as rendered JSX for HyperApp
	L.SetGlobal("hprint", L.NewFunction(func(L *lua.LState) int {
		// Retrieve all the function arguments as a bytes.Buffer
		buf := convert.Arguments2buffer(L, true)
		// Transform JSX to JavaScript and output the result.
		result := api.Transform(buf.String(), ac.jsxOptions)
		if len(result.Errors) > 0 {
			if ac.debugMode {
				// TODO: Use a similar error page as for Lua
				for _, errMsg := range result.Errors {
					fmt.Fprintf(w, "error: %s %d:%d\n", errMsg.Text, errMsg.Location.Line, errMsg.Location.Column)
				}
				for _, warnMsg := range result.Warnings {
					fmt.Fprintf(w, "warning: %s %d:%d\n", warnMsg.Text, warnMsg.Location.Line, warnMsg.Location.Column)
				}
			} else {
				// TODO: Use a similar error page as for Lua
				for _, errMsg := range result.Errors {
					log.Errorf("error: %s %d:%d\n", errMsg.Text, errMsg.Location.Line, errMsg.Location.Column)
				}
				for _, warnMsg := range result.Warnings {
					log.Errorf("warning: %s %d:%d\n", warnMsg.Text, warnMsg.Location.Line, warnMsg.Location.Column)
				}
			}
			return 0 // number of results
		}
		data := result.Code
		// Use "h" instead of "React.createElement" for hyperApp apps
		data = bytes.ReplaceAll(data, []byte("React.createElement("), []byte("h("))
		n, err := w.Write(data)
		if err != nil || n == 0 {
			if ac.debugMode {
				fmt.Fprint(w, "Result from generated JavaScript is empty\n")
			} else {
				log.Error("Result from generated JavaScript is empty\n")
			}
			return 0 // number of results
		}
		return 0 // number of results
	}))

	// Output a simple message HTML page.
	// The first argument is the message (ends up in the <body>).
	// The seconds argument is an optional title.
	// The third argument is an optional page style.
	L.SetGlobal("msgpage", L.NewFunction(func(L *lua.LState) int {
		title := ""
		body := ""
		if L.GetTop() < 2 {
			// Uses an empty string if no first argument is given
			body = L.ToString(1)
		} else {
			title = L.ToString(1)
			body = L.ToString(2)
		}

		// The default theme for single page messages
		theme := "redbox"
		if L.GetTop() >= 3 {
			theme = L.ToString(3)
		}

		// Write a simple HTML page to the client
		w.Write([]byte(themes.MessagePage(title, body, theme)))

		return 0 // number of results
	}))
}

// MarkdownPage write the given source bytes as markdown wrapped in HTML to a writer, with a title
func (ac *Config) MarkdownPage(w http.ResponseWriter, req *http.Request, mdContent []byte, filename string) {
	// Prepare for receiving title and codeStyle information
	searchKeywords := []string{"title", "codestyle", "theme", "replace_with_theme", "css", "favicon"}

	// Also prepare for receiving meta tag information
	searchKeywords = append(searchKeywords, themes.MetaKeywords...)

	// Extract keywords from the given data, and remove the lines with keywords,
	// but only the first time that keyword occurs.
	var kwmap map[string][]byte

	mdContent, kwmap = utils.ExtractKeywords(mdContent, searchKeywords)

	// Create a Markdown parser with the desired extensions
	mdParser := parser.NewWithExtensions(enabledMarkdownExtensions)
	// Convert from Markdown to HTML
	htmlbody := markdown.ToHTML(mdContent, mdParser, nil)

	// TODO: Check if handling "# title <tags" on the first line is valid
	// Markdown or not. Submit a patch to gomarkdown/markdown if it is.

	var h1title []byte
	if bytes.HasPrefix(htmlbody, []byte("<p>#")) {
		fields := bytes.Split(htmlbody, []byte("<"))
		if len(fields) > 2 {
			h1title = bytes.TrimPrefix(fields[1][2:], []byte("#"))
			htmlbody = htmlbody[3+len(h1title):] // 3 is the length of <p>
		}
	}

	// Checkboxes
	htmlbody = bytes.ReplaceAll(htmlbody, []byte("<li>[ ] "), []byte("<li><input type=\"checkbox\" disabled> "))
	htmlbody = bytes.ReplaceAll(htmlbody, []byte("<li><p>[ ] "), []byte("<li><p><input type=\"checkbox\" disabled> "))
	htmlbody = bytes.ReplaceAll(htmlbody, []byte("<li>[x] "), []byte("<li><input type=\"checkbox\" disabled checked> "))
	htmlbody = bytes.ReplaceAll(htmlbody, []byte("<li>[X] "), []byte("<li><input type=\"checkbox\" disabled checked> "))
	htmlbody = bytes.ReplaceAll(htmlbody, []byte("<li><p>[x] "), []byte("<li><p><input type=\"checkbox\" disabled checked> "))

	// These should work by default, but does not.
	// TODO: Look into how gomarkdown/markdown handles this.
	htmlbody = bytes.ReplaceAll(htmlbody, []byte("&amp;gt;"), []byte("&gt;"))
	htmlbody = bytes.ReplaceAll(htmlbody, []byte("&amp;lt;"), []byte("&lt;"))

	// If there is no given title, use the h1title
	title := kwmap["title"]
	if len(title) == 0 {
		if len(h1title) != 0 {
			title = h1title
		} else {
			// If no title has been provided, use the filename
			title = []byte(filepath.Base(filename))
		}
	}

	// Find the theme that should be used
	theme := kwmap["theme"]
	if len(theme) == 0 {
		theme = []byte(ac.defaultTheme)
	}

	// Theme aliases. Use a map if there are more than 2 aliases in the future.
	if string(theme) == "default" {
		// Use the "material" theme by default for Markdown
		theme = []byte("material")
	}

	// Check if a specific string should be replaced with the current theme
	replaceWithTheme := kwmap["replace_with_theme"]
	if len(replaceWithTheme) != 0 {
		// Replace all instances of the value given with "replace_with_theme: ..." with the current theme name
		htmlbody = bytes.ReplaceAll(htmlbody, replaceWithTheme, []byte(theme))
	}

	// If the theme is a filename, create a custom theme where the file is imported from the CSS
	if bytes.Contains(theme, []byte(".")) {
		st := string(theme)
		themes.NewTheme(st, []byte("@import url("+st+");"), themes.DefaultCustomCodeStyle)
	}

	var head strings.Builder

	// If a favicon is specified, use that
	favicon := kwmap["favicon"]
	if len(favicon) > 0 {
		head.WriteString(`<link rel="shortcut icon" type="image/`)

		// Switch on the lowercase file extension of the favicon
		switch strings.ToLower(filepath.Ext(string(favicon))) {
		case ".ico":
			head.WriteString("x-icon")
		case ".bmp":
			head.WriteString("bmp")
		case ".gif":
			head.WriteString("gif")
		case ".jpg", ".jpeg":
			head.WriteString("jpeg")
		case ".svg":
			head.WriteString("svg+xml")
		default:
			head.WriteString("png")
		}

		head.WriteString(`" href="`)
		head.Write(favicon)
		head.WriteString(`"/>`)
	}

	// If style.gcss is present, use that style in <head>
	CSSFilename := filepath.Join(filepath.Dir(filename), themes.DefaultCSSFilename)
	GCSSFilename := filepath.Join(filepath.Dir(filename), themes.DefaultGCSSFilename)
	switch {
	case ac.fs.Exists(CSSFilename):
		// Link to stylesheet (without checking if the CSS file is valid first)
		head.WriteString(`<link href="`)
		head.WriteString(themes.DefaultCSSFilename)
		head.WriteString(`" rel="stylesheet" type="text/css">`)
	case ac.fs.Exists(GCSSFilename):
		if ac.debugMode {
			gcssblock, err := ac.cache.Read(GCSSFilename, ac.shouldCache(".gcss"))
			if err != nil {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
				return
			}
			gcssdata := gcssblock.Bytes()

			// Try compiling the GCSS file first
			errChan := make(chan error)
			go ValidGCSS(gcssdata, errChan)
			err = <-errChan
			if err != nil {
				// Invalid GCSS, return an error page
				ac.PrettyError(w, req, GCSSFilename, gcssdata, err.Error(), "gcss")
				return
			}
		}
		// Link to stylesheet (without checking if the GCSS file is valid first)
		head.WriteString(`<link href="`)
		head.WriteString(themes.DefaultGCSSFilename)
		head.WriteString(`" rel="stylesheet" type="text/css">`)
	default:
		// If not, use the theme by inserting the CSS style directly
		head.Write(themes.StyleHead(string(theme)))
	}

	// Additional CSS file
	additionalCSSfile := string(kwmap["css"])
	if additionalCSSfile != "" {
		// If serving a single Markdown file, include the CSS file inline in a style tag
		if ac.markdownMode && ac.fs.Exists(additionalCSSfile) {
			// Cache the CSS only if Markdown should be cached
			cssblock, err := ac.cache.Read(additionalCSSfile, ac.shouldCache(".md"))
			if err != nil {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
				return
			}
			cssdata := cssblock.Bytes()
			head.WriteString("<style>" + string(cssdata) + "</style>")
		} else {
			head.WriteString(`<link href="`)
			head.WriteString(additionalCSSfile)
			head.WriteString(`" rel="stylesheet" type="text/css">`)
		}
	}

	codeStyle := string(kwmap["codestyle"])

	// Add meta tags, if metadata information has been declared
	for _, keyword := range themes.MetaKeywords {
		if len(kwmap[keyword]) != 0 {
			// Add the meta tag
			head.WriteString(`<meta name="`)
			head.WriteString(keyword)
			head.WriteString(`" content="`)
			head.Write(kwmap[keyword])
			head.WriteString(`" />`)
		}
	}

	// Embed the style and rendered markdown into a simple HTML 5 page
	htmldata := themes.SimpleHTMLPage(title, h1title, []byte(head.String()), htmlbody)

	// TODO: Fix the issue in the splash package so that both MathJax and applying syntax highlighting to code can be used at the same time

	// Add a script for rendering MathJax, but only if at least one mathematical formula is present
	if containsFormula(mdContent) {
		js := append([]byte(`<script id="MathJax-script">`), []byte(mathJaxScript)...)
		htmldata = InsertScriptTag(htmldata, js) // also adds the closing </script> tag
	} else if bytes.Contains(htmlbody, []byte("<pre><code ")) { // Add syntax highlighting to the header, but only if '<pre><code ' is present
		// If codeStyle is not "none", highlight the current htmldata
		if codeStyle == "" {
			// Use the highlight style from the current theme
			highlighted, err := splash.UnescapeSplash(htmldata, themes.ThemeToCodeStyle(string(theme)))
			if err != nil {
				log.Error(err)
			} else {
				// Only use the new and highlighted HTML if there were no errors
				//htmldata = highlighted
				htmldata = highlighted
			}
		} else if codeStyle != "none" {
			// Use the highlight style from codeStyle
			highlighted, err := splash.UnescapeSplash(htmldata, codeStyle)
			if err != nil {
				log.Error(err)
			} else {
				// Only use the new HTML if there were no errors
				htmldata = highlighted
			}
		}
	}

	// If the auto-refresh feature has been enabled
	if ac.autoRefresh {
		// Insert JavaScript for refreshing the page into the generated HTML
		htmldata = ac.InsertAutoRefresh(req, htmldata)
	}

	// Write the rendered Markdown page to the client
	ac.DataToClient(w, req, filename, htmldata)
}

// PongoPage write the given source bytes (ina Pongo2) converted to HTML, to a writer.
// The filename is only used in error messages, if any.
func (ac *Config) PongoPage(w http.ResponseWriter, req *http.Request, filename string, pongodata []byte, funcs template.FuncMap) {
	var (
		buf                   bytes.Buffer
		linkInGCSS, linkInCSS bool
		dirName               = filepath.Dir(filename)
		GCSSFilename          = filepath.Join(dirName, themes.DefaultGCSSFilename)
		CSSFilename           = filepath.Join(dirName, themes.DefaultCSSFilename)
	)

	// If style.gcss is present, and a header is present, and it has not already been linked in, link it in
	if ac.fs.Exists(CSSFilename) {
		linkInCSS = true
	} else if ac.fs.Exists(GCSSFilename) {
		if ac.debugMode {
			gcssblock, err := ac.cache.Read(GCSSFilename, ac.shouldCache(".gcss"))
			if err != nil {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
				return
			}
			gcssdata := gcssblock.Bytes()

			// Try compiling the GCSS file before the Pongo2 file
			errChan := make(chan error)
			go ValidGCSS(gcssdata, errChan)
			err = <-errChan
			if err != nil {
				// Invalid GCSS, return an error page
				ac.PrettyError(w, req, GCSSFilename, gcssdata, err.Error(), "gcss")
				return
			}
		}
		linkInGCSS = true
	}

	// Set the base directory for Pongo2 to the one where the given filename is
	if err := pongo2.DefaultLoader.SetBaseDir(dirName); err != nil {
		if ac.debugMode {
			ac.PrettyError(w, req, filename, pongodata, err.Error(), "pongo2")
		} else {
			log.Errorf("Could not set base directory for Pongo2 to %s:\n%s", dirName, err)
		}
		return
	}

	// Prepare a Pongo2 template
	tpl, err := pongo2.DefaultSet.FromBytes(pongodata)
	if err != nil {
		if ac.debugMode {
			ac.PrettyError(w, req, filename, pongodata, err.Error(), "pongo2")
		} else {
			log.Errorf("Could not compile Pongo2 template:\n%s\n%s", err, string(pongodata))
		}
		return
	}

	okfuncs := make(pongo2.Context)

	// Go through the global Lua scope
	for k, v := range funcs {
		// Skip the ones starting with an underscore
		//if strings.HasPrefix(k, "_") {
		//	continue
		//}

		// Check if the name in question is a function
		if f, ok := v.(func(...string) (any, error)); ok {

			// For the closure to correctly wrap the key value
			k := k

			// Wrap the Lua functions as Pongo2 functions
			wrapfunc := func(vals ...*pongo2.Value) *pongo2.Value {
				// Convert the Pongo2 arguments to string arguments
				strs := make([]string, len(vals))
				for i, sv := range vals {
					strs[i] = sv.String()
				}

				// Call the Lua function
				retval, err := f(strs...)
				// Return the error if things go wrong
				if err != nil {
					return pongo2.AsValue(err)
				}
				// Return the returned value if things went well
				return pongo2.AsValue(retval)
			}
			// Save the wrapped function for the pongo2 template execution
			okfuncs[k] = wrapfunc

		} else if s, ok := v.(string); ok {
			// String variables
			okfuncs[k] = s
		} else {
			// Exposing variable as it is.
			// TODO: Add more tests for this codepath
			okfuncs[k] = v
		}
	}

	// Make the Lua functions available to Pongo
	pongo2.Globals.Update(okfuncs)

	defer func() {
		if r := recover(); r != nil {
			errmsg := fmt.Sprintf("Pongo2 error: %s", r)
			if ac.debugMode {
				ac.PrettyError(w, req, filename, pongodata, errmsg, "pongo2")
			} else {
				log.Errorf("Could not execute Pongo2 template:\n%s", errmsg)
			}
		}
	}()

	// Render the Pongo2 template to the buffer
	err = tpl.ExecuteWriter(pongo2.Globals, &buf)
	if err != nil {
		// if err := tpl.ExecuteWriterUnbuffered(pongo2.Globals, &buf); err != nil {
		if ac.debugMode {
			ac.PrettyError(w, req, filename, pongodata, err.Error(), "pongo2")
		} else {
			log.Errorf("Could not execute Pongo2 template:\n%s", err)
		}
		return
	}

	// Check if we are dealing with HTML
	if strings.Contains(buf.String(), "<html>") {

		if linkInCSS || linkInGCSS {
			// Link in stylesheet
			htmldata := buf.Bytes()
			if linkInCSS {
				htmldata = themes.StyleHTML(htmldata, themes.DefaultCSSFilename)
			} else if linkInGCSS {
				htmldata = themes.StyleHTML(htmldata, themes.DefaultGCSSFilename)
			}
			buf.Reset()
			_, err := buf.Write(htmldata)
			if err != nil {
				if ac.debugMode {
					ac.PrettyError(w, req, filename, pongodata, err.Error(), "pongo2")
				} else {
					log.Errorf("Can not write bytes to a buffer! Out of memory?\n%s", err)
				}
				return
			}
		}

		// If the auto-refresh feature has been enabled
		if ac.autoRefresh {
			// Insert JavaScript for refreshing the page into the generated HTML
			changedBytes := ac.InsertAutoRefresh(req, buf.Bytes())

			buf.Reset()
			_, err := buf.Write(changedBytes)
			if err != nil {
				if ac.debugMode {
					ac.PrettyError(w, req, filename, pongodata, err.Error(), "pongo2")
				} else {
					log.Errorf("Can not write bytes to a buffer! Out of memory?\n%s", err)
				}
				return
			}
		}

		// If doctype is missing, add doctype for HTML5 at the top
		changedBytes := themes.InsertDoctype(buf.Bytes())
		buf.Reset()
		buf.Write(changedBytes)
	}

	// Write the rendered template to the client
	ac.DataToClient(w, req, filename, buf.Bytes())
}

// AmberPage the given source bytes (in Amber) converted to HTML, to a writer.
// The filename is only used in error messages, if any.
func (ac *Config) AmberPage(w http.ResponseWriter, req *http.Request, filename string, amberdata []byte, funcs template.FuncMap) {
	var (
		buf bytes.Buffer
		// If style.gcss is present, and a header is present, and it has not already been linked in, link it in
		dirName      = filepath.Dir(filename)
		GCSSFilename = filepath.Join(dirName, themes.DefaultGCSSFilename)
		CSSFilename  = filepath.Join(dirName, themes.DefaultCSSFilename)
	)

	if ac.fs.Exists(CSSFilename) {
		// Link to stylesheet (without checking if the GCSS file is valid first)
		amberdata = themes.StyleAmber(amberdata, themes.DefaultCSSFilename)
	} else if ac.fs.Exists(GCSSFilename) {
		if ac.debugMode {
			gcssblock, err := ac.cache.Read(GCSSFilename, ac.shouldCache(".gcss"))
			if err != nil {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
				return
			}
			gcssdata := gcssblock.Bytes()

			// Try compiling the GCSS file before the Amber file
			errChan := make(chan error)
			go ValidGCSS(gcssdata, errChan)
			err = <-errChan
			if err != nil {
				// Invalid GCSS, return an error page
				ac.PrettyError(w, req, GCSSFilename, gcssdata, err.Error(), "gcss")
				return
			}
		}
		// Link to stylesheet (without checking if the GCSS file is valid first)
		amberdata = themes.StyleAmber(amberdata, themes.DefaultGCSSFilename)
	}

	// Compile the given amber template
	tpl, err := amber.CompileData(amberdata, filename, amber.Options{PrettyPrint: true, LineNumbers: false})
	if err != nil {
		if ac.debugMode {
			ac.PrettyError(w, req, filename, amberdata, err.Error(), "amber")
		} else {
			log.Errorf("Could not compile Amber template:\n%s\n%s", err, string(amberdata))
		}
		return
	}

	// Render the Amber template to the buffer
	if err := tpl.Execute(&buf, funcs); err != nil {

		// If it was one particular error, where the template can not find the
		// function or variable name that is used, give the user a friendlier
		// message.
		if strings.TrimSpace(err.Error()) == "reflect: call of reflect.Value.Type on zero Value" {
			errortext := "Could not execute Amber template!<br>One of the functions called by the template is not available."
			if ac.debugMode {
				ac.PrettyError(w, req, filename, amberdata, errortext, "amber")
			} else {
				errortext = strings.Replace(errortext, "<br>", "\n", 1)
				log.Errorf("Could not execute Amber template:\n%s", errortext)
			}
		} else {
			if ac.debugMode {
				ac.PrettyError(w, req, filename, amberdata, err.Error(), "amber")
			} else {
				log.Errorf("Could not execute Amber template:\n%s", err)
			}
		}
		return
	}

	// If the auto-refresh feature has been enabled
	if ac.autoRefresh {
		// Insert JavaScript for refreshing the page into the generated HTML
		changedBytes := ac.InsertAutoRefresh(req, buf.Bytes())

		buf.Reset()
		_, err := buf.Write(changedBytes)
		if err != nil {
			if ac.debugMode {
				ac.PrettyError(w, req, filename, amberdata, err.Error(), "amber")
			} else {
				log.Errorf("Can not write bytes to a buffer! Out of memory?\n%s", err)
			}
			return
		}
	}

	// If doctype is missing, add doctype for HTML5 at the top
	changedBuf := bytes.NewBuffer(themes.InsertDoctype(buf.Bytes()))
	buf = *changedBuf

	// Write the rendered template to the client
	ac.DataToClient(w, req, filename, buf.Bytes())
}

// GCSSPage writes the given source bytes (in GCSS) converted to CSS, to a writer.
// The filename is only used in the error message, if any.
func (ac *Config) GCSSPage(w http.ResponseWriter, req *http.Request, filename string, gcssdata []byte) {
	var buf bytes.Buffer
	if _, err := gcss.Compile(&buf, bytes.NewReader(gcssdata)); err != nil {
		if ac.debugMode {
			fmt.Fprintf(w, "Could not compile GCSS:\n\n%s\n%s", err, string(gcssdata))
		} else {
			log.Errorf("Could not compile GCSS:\n%s\n%s", err, string(gcssdata))
		}
		return
	}
	// Write the resulting CSS to the client
	ac.DataToClient(w, req, filename, buf.Bytes())
}

// JSXPage writes the given source bytes (in JSX) converted to JS, to a writer.
// The filename is only used in the error message, if any.
func (ac *Config) JSXPage(w http.ResponseWriter, req *http.Request, filename string, jsxdata []byte) {
	var buf bytes.Buffer
	buf.Write(jsxdata)

	// Convert JSX to JS
	result := api.Transform(buf.String(), ac.jsxOptions)
	if len(result.Errors) > 0 {
		if ac.debugMode {
			var sb strings.Builder
			for _, errMsg := range result.Errors {
				sb.WriteString(fmt.Sprintf("error: %s %s:%d:%d\n", errMsg.Text, filename, errMsg.Location.Line, errMsg.Location.Column))
			}
			for _, warnMsg := range result.Warnings {
				sb.WriteString(fmt.Sprintf("warning: %s %s:%d:%d\n", warnMsg.Text, filename, warnMsg.Location.Line, warnMsg.Location.Column))
			}
			ac.PrettyError(w, req, filename, jsxdata, sb.String(), "jsx")
		} else {
			// TODO: Use a similar error page as for Lua
			for _, errMsg := range result.Errors {
				log.Errorf("error: %s %s:%d:%d\n", errMsg.Text, filename, errMsg.Location.Line, errMsg.Location.Column)
			}
			for _, warnMsg := range result.Warnings {
				log.Errorf("warning: %s %s:%d:%d\n", warnMsg.Text, filename, warnMsg.Location.Line, warnMsg.Location.Column)
			}
		}
		return
	}
	data := result.Code

	// Use "h" instead of "React.createElement" for hyperApp apps
	if ac.hyperApp {
		data = bytes.ReplaceAll(data, []byte("React.createElement("), []byte("h("))
	}

	ac.DataToClient(w, req, filename, data)
}

// HyperAppPage writes the given source bytes (in JSX for HyperApp) converted to JS, to a writer.
// The filename is only used in the error message, if any.
func (ac *Config) HyperAppPage(w http.ResponseWriter, req *http.Request, filename string, jsxdata []byte) {
	var (
		htmlbuf strings.Builder
		jsxbuf  bytes.Buffer
	)

	// Wrap the rendered HyperApp JSX in some HTML
	htmlbuf.WriteString("<!doctype html><html><head>")

	// If style.gcss is present, use that style in <head>
	CSSFilename := filepath.Join(filepath.Dir(filename), themes.DefaultCSSFilename)
	GCSSFilename := filepath.Join(filepath.Dir(filename), themes.DefaultGCSSFilename)
	switch {
	case ac.fs.Exists(CSSFilename):
		// Link to stylesheet (without checking if the GCSS file is valid first)
		htmlbuf.WriteString(`<link href="`)
		htmlbuf.WriteString(themes.DefaultCSSFilename)
		htmlbuf.WriteString(`" rel="stylesheet" type="text/css">`)
	case ac.fs.Exists(GCSSFilename):
		if ac.debugMode {
			gcssblock, err := ac.cache.Read(GCSSFilename, ac.shouldCache(".gcss"))
			if err != nil {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
				return
			}
			gcssdata := gcssblock.Bytes()

			// Try compiling the GCSS file first
			errChan := make(chan error)
			go ValidGCSS(gcssdata, errChan)
			err = <-errChan
			if err != nil {
				// Invalid GCSS, return an error page
				ac.PrettyError(w, req, GCSSFilename, gcssdata, err.Error(), "gcss")
				return
			}
		}
		// Link to stylesheet (without checking if the GCSS file is valid first)
		htmlbuf.WriteString(`<link href="`)
		htmlbuf.WriteString(themes.DefaultGCSSFilename)
		htmlbuf.WriteString(`" rel="stylesheet" type="text/css">`)
	default:
		// If not, use the default hyperapp theme by inserting the CSS style directly
		theme := ac.defaultTheme

		// Use the "neon" theme by default for HyperApp
		if theme == "default" {
			theme = "neon"
		}

		htmlbuf.Write(themes.StyleHead(theme))
	}

	// Convert JSX to JS
	jsxbuf.Write(jsxdata)
	jsxResult := api.Transform(jsxbuf.String(), ac.jsxOptions)

	if len(jsxResult.Errors) > 0 {
		if ac.debugMode {
			var sb strings.Builder
			for _, errMsg := range jsxResult.Errors {
				sb.WriteString(fmt.Sprintf("error: %s %s:%d:%d\n", errMsg.Text, filename, errMsg.Location.Line, errMsg.Location.Column))
			}
			for _, warnMsg := range jsxResult.Warnings {
				sb.WriteString(fmt.Sprintf("warning: %s %s:%d:%d\n", warnMsg.Text, filename, warnMsg.Location.Line, warnMsg.Location.Column))
			}
			ac.PrettyError(w, req, filename, jsxdata, sb.String(), "jsx")
		} else {
			// TODO: Use a similar error page as for Lua
			for _, errMsg := range jsxResult.Errors {
				log.Errorf("error: %s %s:%d:%d\n", errMsg.Text, filename, errMsg.Location.Line, errMsg.Location.Column)
			}
			for _, warnMsg := range jsxResult.Warnings {
				log.Errorf("warning: %s %s:%d:%d\n", warnMsg.Text, filename, warnMsg.Location.Line, warnMsg.Location.Column)
			}
		}
		return
	}

	// Include the hyperapp javascript from unpkg.com
	// htmlbuf.WriteString("</head><body><script src=\"https://unpkg.com/hyperapp\"></script><script>")

	// Embed the hyperapp script directly, for speed
	htmlbuf.WriteString("</head><body><script>")
	htmlbuf.Write(hyperAppJSBytes)

	// The HyperApp library + compiled JSX can live in the same script tag. No need for this:
	// htmlbuf.WriteString("</script><script>")

	jsxData := jsxResult.Code

	// Use "h" instead of "React.createElement"
	jsxData = bytes.ReplaceAll(jsxData, []byte("React.createElement("), []byte("h("))

	// If the file does not seem to contain the hyper app import: add it to the top of the script
	// TODO: Consider making a more robust (and slower) check that splits the data into words first
	if !bytes.Contains(jsxData, []byte("import { h,")) {
		htmlbuf.WriteString("const { h, app } = hyperapp;")
	}

	// Insert the JS data
	htmlbuf.Write(jsxData)

	// Tail of the HTML wrapper page
	htmlbuf.WriteString("</script></body>")

	// Output HTML + JS to browser
	ac.DataToClient(w, req, filename, []byte(htmlbuf.String()))
}

// SCSSPage writes the given source bytes (in SCSS) converted to CSS, to a writer.
// The filename is only used in the error message, if any.
func (ac *Config) SCSSPage(w http.ResponseWriter, req *http.Request, filename string, scssdata []byte) {
	// TODO: Gather stderr and print with log.Errorf if needed
	o := console.Output{}
	// Silence the compiler output
	if !ac.debugMode {
		o.Disable()
	}
	// Compile the given filename. Sass might want to import other file, which is probably
	// why the Sass compiler doesn't support just taking in a slice of bytes.
	cssString, err := compiler.Run(filename)
	if !ac.debugMode {
		o.Enable()
	}
	if err != nil {
		if ac.debugMode {
			fmt.Fprintf(w, "Could not compile SCSS:\n\n%s\n%s", err, string(scssdata))
		} else {
			log.Errorf("Could not compile SCSS:\n%s\n%s", err, string(scssdata))
		}
		return
	}
	// Write the resulting CSS to the client
	ac.DataToClient(w, req, filename, []byte(cssString))
}
