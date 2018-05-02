package engine

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/eknkc/amber"
	"github.com/flosch/pongo2"
	"github.com/jvatic/goja-babel"
	"github.com/russross/blackfriday"
	log "github.com/sirupsen/logrus"
	"github.com/wellington/sass/compiler"
	"github.com/xyproto/algernon/console"
	"github.com/xyproto/algernon/lua/convert"
	"github.com/xyproto/algernon/themes"
	"github.com/xyproto/algernon/utils"
	"github.com/xyproto/splash"
	"github.com/yosssi/gcss"
	"github.com/yuin/gopher-lua"
)

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
func (ac *Config) LoadRenderFunctions(w http.ResponseWriter, req *http.Request, L *lua.LState) {

	// Output Markdown as HTML
	L.SetGlobal("mprint", L.NewFunction(func(L *lua.LState) int {
		// Retrieve all the function arguments as a bytes.Buffer
		buf := convert.Arguments2buffer(L, true)
		// Convert the buffer to markdown and output the translated string
		w.Write(blackfriday.MarkdownCommon(buf.Bytes()))
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
		// Retrieve all the function arguments as a bytes.Buffer
		buf := convert.Arguments2buffer(L, true)
		// Use the buffer as a template.
		// Options are "Pretty printing, but without line numbers."
		tpl, err := pongo2.FromBytes(buf.Bytes())
		if err != nil {
			if ac.debugMode {
				fmt.Fprint(w, "Could not compile Pongo2 template:\n\t"+err.Error()+"\n\n"+buf.String())
			} else {
				log.Errorf("Could not compile Pongo2 template:\n%s\n%s", err, buf.String())
			}
			return 0 // number of results
		}
		// nil is the template context (variables etc in a map)
		if err := tpl.ExecuteWriter(nil, w); err != nil {
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
			//return 0 // number of results
		}
		return 0 // number of results
	}))

	// Output text as rendered JSX for React
	L.SetGlobal("jprint", L.NewFunction(func(L *lua.LState) int {
		// Retrieve all the function arguments as a bytes.Buffer
		buf := convert.Arguments2buffer(L, true)
		// Transform JSX to JavaScript and output the result.
		res, err := babel.Transform(&buf, ac.jsxOptions)
		if err != nil {
			if ac.debugMode {
				// TODO: Use a similar error page as for Lua
				fmt.Fprint(w, "Could not generate JavaScript:\n\t"+err.Error()+"\n\n"+buf.String())
			} else {
				log.Errorf("Could not generate JavaScript:\n%s\n%s", err, buf.String())
			}
			return 0 // number of results
		}
		if res != nil {
			io.Copy(w, res)
		}
		return 0 // number of results
	}))

	// Output text as rendered JSX for HyperApp
	L.SetGlobal("hprint", L.NewFunction(func(L *lua.LState) int {
		// Retrieve all the function arguments as a bytes.Buffer
		buf := convert.Arguments2buffer(L, true)
		// Transform JSX to JavaScript and output the result.
		res, err := babel.Transform(&buf, ac.jsxOptions)
		if err != nil {
			if ac.debugMode {
				// TODO: Use a similar error page as for Lua
				fmt.Fprint(w, "Could not generate JavaScript:\n\t"+err.Error()+"\n\n"+buf.String())
			} else {
				log.Errorf("Could not generate JavaScript:\n%s\n%s", err, buf.String())
			}
			return 0 // number of results
		}
		if res != nil {
			data, err := ioutil.ReadAll(res)
			if err != nil {
				log.Error("Could not read bytes from JSX generator:", err)
				return 0 // number of results
			}

			// Use "h" instead of "React.createElement" for hyperApp apps
			data = bytes.Replace(data, []byte("React.createElement("), []byte("h("), utils.EveryInstance)
			w.Write(data)
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
func (ac *Config) MarkdownPage(w http.ResponseWriter, req *http.Request, data []byte, filename string) {
	// Prepare for receiving title and codeStyle information
	searchKeywords := []string{"title", "codestyle", "theme", "replace_with_theme", "css"}

	// Also prepare for receiving meta tag information
	searchKeywords = append(searchKeywords, themes.MetaKeywords...)

	// Extract keywords from the given data, and remove the lines with keywords
	var kwmap map[string][]byte
	data, kwmap = utils.ExtractKeywords(data, searchKeywords)

	// Convert from Markdown to HTML
	htmlbody := blackfriday.MarkdownCommon(data)

	// TODO: Check if handling "# title <tags" on the first line is valid
	// Markdown or not. Submit a patch to blackfriday if it is.

	var h1title []byte
	if bytes.HasPrefix(htmlbody, []byte("<p>#")) {
		fields := bytes.Split(htmlbody, []byte("<"))
		if len(fields) > 2 {
			h1title = bytes.TrimPrefix(fields[1][2:], []byte("#"))
			htmlbody = htmlbody[3+len(h1title):] // 3 is the length of <p>
		}
	}

	// Checkboxes
	htmlbody = bytes.Replace(htmlbody, []byte("<li>[ ] "), []byte("<li><input type=\"checkbox\" disabled> "), utils.EveryInstance)
	htmlbody = bytes.Replace(htmlbody, []byte("<li>[x] "), []byte("<li><input type=\"checkbox\" disabled checked> "), utils.EveryInstance)
	htmlbody = bytes.Replace(htmlbody, []byte("<li>[X] "), []byte("<li><input type=\"checkbox\" disabled checked> "), utils.EveryInstance)

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
		htmlbody = bytes.Replace(htmlbody, replaceWithTheme, []byte(theme), utils.EveryInstance)
	}

	// If the theme is a filename, create a custom theme where the file is imported from the CSS
	if bytes.Contains(theme, []byte(".")) {
		st := string(theme)
		themes.NewTheme(st, []byte("@import url("+st+");"), themes.DefaultCustomCodeStyle)
	}

	var head bytes.Buffer

	// If style.gcss is present, use that style in <head>
	GCSSfilename := filepath.Join(filepath.Dir(filename), themes.DefaultStyleFilename)
	if ac.fs.Exists(GCSSfilename) {
		if ac.debugMode {
			gcssblock, err := ac.cache.Read(GCSSfilename, ac.shouldCache(".gcss"))
			if err != nil {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
				return
			}
			gcssdata := gcssblock.MustData()

			// Try compiling the GCSS file first
			errChan := make(chan error)
			go ValidGCSS(gcssdata, errChan)
			err = <-errChan
			if err != nil {
				// Invalid GCSS, return an error page
				ac.PrettyError(w, req, GCSSfilename, gcssdata, err.Error(), "gcss")
				return
			}
		}
		// Link to stylesheet (without checking if the GCSS file is valid first)
		head.WriteString(`<link href="`)
		head.WriteString(themes.DefaultStyleFilename)
		head.WriteString(`" rel="stylesheet" type="text/css">`)
	} else {
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
			cssdata := cssblock.MustData()
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
	htmldata := themes.SimpleHTMLPage(title, h1title, head.Bytes(), htmlbody)

	// Add syntax highlighting to the header, but only if "<pre" is present
	if bytes.Contains(htmlbody, []byte("<pre")) {
		// If codeStyle is not "none", highlight the current htmldata
		if codeStyle == "" {
			// Use the highlight style from the current theme
			highlighted, err := splash.UnescapeSplash(htmldata, themes.ThemeToCodeStyle(string(theme)))
			if err != nil {
				log.Error(err)
			} else {
				// Only use the new and highlighted HTML if there were no errors
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

	var buf bytes.Buffer

	linkInGCSS := false
	// If style.gcss is present, and a header is present, and it has not already been linked in, link it in
	GCSSfilename := filepath.Join(filepath.Dir(filename), themes.DefaultStyleFilename)
	if ac.fs.Exists(GCSSfilename) {
		if ac.debugMode {
			gcssblock, err := ac.cache.Read(GCSSfilename, ac.shouldCache(".gcss"))
			if err != nil {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
				return
			}
			gcssdata := gcssblock.MustData()

			// Try compiling the GCSS file before the Pongo2 file
			errChan := make(chan error)
			go ValidGCSS(gcssdata, errChan)
			err = <-errChan
			if err != nil {
				// Invalid GCSS, return an error page
				ac.PrettyError(w, req, GCSSfilename, gcssdata, err.Error(), "gcss")
				return
			}
		}
		linkInGCSS = true
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
		if f, ok := v.(func(...string) (interface{}, error)); ok {

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
		//if err := tpl.ExecuteWriterUnbuffered(pongo2.Globals, &buf); err != nil {
		if ac.debugMode {
			ac.PrettyError(w, req, filename, pongodata, err.Error(), "pongo2")
		} else {
			log.Errorf("Could not execute Pongo2 template:\n%s", err)
		}
		return
	}

	// Check if we are dealing with HTML
	if bytes.Contains(buf.Bytes(), []byte("<html>")) {

		if linkInGCSS {
			// Link in stylesheet
			htmldata := buf.Bytes()
			themes.StyleHTML(&htmldata, themes.DefaultStyleFilename)
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

	var buf bytes.Buffer

	// If style.gcss is present, and a header is present, and it has not already been linked in, link it in
	GCSSfilename := filepath.Join(filepath.Dir(filename), themes.DefaultStyleFilename)
	if ac.fs.Exists(GCSSfilename) {
		if ac.debugMode {
			gcssblock, err := ac.cache.Read(GCSSfilename, ac.shouldCache(".gcss"))
			if err != nil {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
				return
			}
			gcssdata := gcssblock.MustData()

			// Try compiling the GCSS file before the Amber file
			errChan := make(chan error)
			go ValidGCSS(gcssdata, errChan)
			err = <-errChan
			if err != nil {
				// Invalid GCSS, return an error page
				ac.PrettyError(w, req, GCSSfilename, gcssdata, err.Error(), "gcss")
				return
			}
		}
		// Link to stylesheet (without checking if the GCSS file is valid first)
		themes.StyleAmber(&amberdata, themes.DefaultStyleFilename)
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
	res, err := babel.Transform(&buf, ac.jsxOptions)
	if err != nil {
		if ac.debugMode {
			ac.PrettyError(w, req, filename, jsxdata, err.Error(), "jsx")
		} else {
			log.Errorf("Could not generate javascript:\n%s\n%s", err, buf.String())
		}
		return
	}
	if res != nil {
		data, err := ioutil.ReadAll(res)
		if err != nil {
			log.Error("Could not read bytes from JSX generator:", err)
			return
		}

		// Use "h" instead of "React.createElement" for hyperApp apps
		if ac.hyperApp {
			data = bytes.Replace(data, []byte("React.createElement("), []byte("h("), utils.EveryInstance)
		}

		ac.DataToClient(w, req, filename, data)
	}
}

// HyperAppPage writes the given source bytes (in JSX for HyperApp) converted to JS, to a writer.
// The filename is only used in the error message, if any.
func (ac *Config) HyperAppPage(w http.ResponseWriter, req *http.Request, filename string, jsxdata []byte) {
	var (
		htmlbuf bytes.Buffer
		jsxbuf  bytes.Buffer
	)

	// Wrap the rendered HyperApp JSX in some HTML
	htmlbuf.WriteString("<!doctype html><html><head>")

	// If style.gcss is present, use that style in <head>
	GCSSfilename := filepath.Join(filepath.Dir(filename), themes.DefaultStyleFilename)
	if ac.fs.Exists(GCSSfilename) {
		if ac.debugMode {
			gcssblock, err := ac.cache.Read(GCSSfilename, ac.shouldCache(".gcss"))
			if err != nil {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
				return
			}
			gcssdata := gcssblock.MustData()

			// Try compiling the GCSS file first
			errChan := make(chan error)
			go ValidGCSS(gcssdata, errChan)
			err = <-errChan
			if err != nil {
				// Invalid GCSS, return an error page
				ac.PrettyError(w, req, GCSSfilename, gcssdata, err.Error(), "gcss")
				return
			}
		}
		// Link to stylesheet (without checking if the GCSS file is valid first)
		htmlbuf.WriteString(`<link href="`)
		htmlbuf.WriteString(themes.DefaultStyleFilename)
		htmlbuf.WriteString(`" rel="stylesheet" type="text/css">`)
	} else {
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
	jsxGenerator, err := babel.Transform(&jsxbuf, ac.jsxOptions)
	if err != nil {
		if ac.debugMode {
			ac.PrettyError(w, req, filename, jsxdata, err.Error(), "jsx")
		} else {
			log.Errorf("Could not generate javascript:\n%s\n%s", err, jsxbuf.String())
		}
		return
	}

	// Include the hyperapp javascript from unpkg.com
	//htmlbuf.WriteString("</head><body><script src=\"https://unpkg.com/hyperapp\"></script><script>")

	// Embed the hyperapp script directly, for speed
	htmlbuf.WriteString("</head><body><script>")
	htmlbuf.Write(hyperAppJSBytes)

	// The HyperApp library + compiled JSX can live in the same script tag. No need for this:
	//htmlbuf.WriteString("</script><script>")

	if jsxGenerator != nil {
		// Read from the generator
		jsxData, err := ioutil.ReadAll(jsxGenerator)
		if err != nil {
			log.Error("Could not read bytes from JSX generator:", err)
			return
		}

		// Use "h" instead of "React.createElement"
		jsxData = bytes.Replace(jsxData, []byte("React.createElement("), []byte("h("), utils.EveryInstance)

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
		ac.DataToClient(w, req, filename, htmlbuf.Bytes())
	}
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
