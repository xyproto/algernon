package main

import (
	"bytes"
	"fmt"
	"github.com/eknkc/amber"
	"github.com/flosch/pongo2"
	"github.com/mamaar/risotto/generator"
	"github.com/mamaar/risotto/parser"
	"github.com/russross/blackfriday"
	log "github.com/sirupsen/logrus"
	"github.com/wellington/sass/compiler"
	"github.com/yosssi/gcss"
	"github.com/yuin/gopher-lua"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
)

// Expose functions that are related to rendering text, to the given Lua state
func exportRenderFunctions(w http.ResponseWriter, req *http.Request, L *lua.LState) {

	// Output Markdown as HTML
	L.SetGlobal("mprint", L.NewFunction(func(L *lua.LState) int {
		// Retrieve all the function arguments as a bytes.Buffer
		buf := arguments2buffer(L, true)
		// Convert the buffer to markdown and output the translated string
		w.Write(blackfriday.MarkdownCommon([]byte(buf.String())))
		return 0 // number of results
	}))

	// Output text as rendered amber.
	L.SetGlobal("aprint", L.NewFunction(func(L *lua.LState) int {
		// Retrieve all the function arguments as a bytes.Buffer
		buf := arguments2buffer(L, true)
		// Use the buffer as a template.
		// Options are "Pretty printing, but without line numbers."
		tpl, err := amber.Compile(buf.String(), amber.Options{PrettyPrint: true, LineNumbers: false})
		if err != nil {
			if debugMode {
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
		buf := arguments2buffer(L, true)
		// Use the buffer as a template.
		// Options are "Pretty printing, but without line numbers."
		tpl, err := pongo2.FromBytes(buf.Bytes())
		if err != nil {
			if debugMode {
				fmt.Fprint(w, "Could not compile Pongo2 template:\n\t"+err.Error()+"\n\n"+buf.String())
			} else {
				log.Errorf("Could not compile Pongo2 template:\n%s\n%s", err, buf.String())
			}
			return 0 // number of results
		}
		// nil is the template context (variables etc in a map)
		if err := tpl.ExecuteWriter(nil, w); err != nil {
			if debugMode {
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
		buf := arguments2buffer(L, true)
		// Transform GCSS to CSS and output the result.
		// Ignoring the number of bytes written.
		if _, err := gcss.Compile(w, &buf); err != nil {
			if debugMode {
				fmt.Fprint(w, "Could not compile GCSS:\n\t"+err.Error()+"\n\n"+buf.String())
			} else {
				log.Errorf("Could not compile GCSS:\n%s\n%s", err, buf.String())
			}
			//return 0 // number of results
		}
		return 0 // number of results
	}))

	// Output text as rendered JSX
	L.SetGlobal("jprint", L.NewFunction(func(L *lua.LState) int {
		// Retrieve all the function arguments as a bytes.Buffer
		buf := arguments2buffer(L, true)
		// Transform JSX to JavaScript and output the result.
		prog, err := parser.ParseFile(nil, "<input>", &buf, parser.IgnoreRegExpErrors)
		if err != nil {
			if debugMode {
				// TODO: Use a similar error page as for Lua
				fmt.Fprint(w, "Could not parse JSX:\n\t"+err.Error()+"\n\n"+buf.String())
			} else {
				log.Errorf("Could not parse JSX:\n%s\n%s", err, buf.String())
			}
			return 0 // number of results
		}
		gen, err := generator.Generate(prog)
		if err != nil {
			if debugMode {
				// TODO: Use a similar error page as for Lua
				fmt.Fprint(w, "Could not generate JavaScript:\n\t"+err.Error()+"\n\n"+buf.String())
			} else {
				log.Errorf("Could not generate JavaScript:\n%s\n%s", err, buf.String())
			}
			return 0 // number of results
		}
		if gen != nil {
			io.Copy(w, gen)
		}
		return 0 // number of results
	}))

}

// HTML to be added for enabling highlighting.
func highlightHTML(codeStyle string) string {
	return `<link rel="stylesheet" href="//cdnjs.cloudflare.com/ajax/libs/highlight.js/9.6.0/styles/` + codeStyle + `.min.css"><script src="//cdnjs.cloudflare.com/ajax/libs/highlight.js/9.6.0/highlight.min.js"></script><script>hljs.initHighlightingOnLoad();</script>`
}

// Write the given source bytes as markdown wrapped in HTML to a writer, with a title
func markdownPage(w http.ResponseWriter, req *http.Request, data []byte, filename string, cache *FileCache) {
	// Prepare for receiving title and codeStyle information
	given := map[string]string{"title": "", "codestyle": "", "theme": "", "replace_with_theme": ""}

	// Also prepare for receiving meta tag information
	addMetaKeywords(given)

	// Extract keywords from the given data, and remove the lines with keywords
	data = extractKeywords(data, given)

	// Convert from Markdown to HTML
	htmlbody := string(blackfriday.MarkdownCommon(data))

	// TODO: Check if handling "# title <tags" on the first line is valid
	// Markdown or not. Submit a patch to blackfriday if it is.

	h1title := ""
	if strings.HasPrefix(htmlbody, "<p>#") {
		fields := strings.Split(htmlbody, "<")
		if len(fields) > 2 {
			h1title = fields[1][2:]
			htmlbody = htmlbody[len("<p>"+h1title):]
			if strings.HasPrefix(h1title, "#") {
				h1title = h1title[1:]
			}
		}
	}

	// Checkboxes
	htmlbody = strings.Replace(htmlbody, "<li>[ ] ", "<li><input type=\"checkbox\" disabled> ", everyInstance)
	htmlbody = strings.Replace(htmlbody, "<li>[x] ", "<li><input type=\"checkbox\" disabled checked> ", everyInstance)
	htmlbody = strings.Replace(htmlbody, "<li>[X] ", "<li><input type=\"checkbox\" disabled checked> ", everyInstance)

	// If there is no given title, use the h1title
	title := given["title"]
	if title == "" {
		if h1title != "" {
			title = h1title
		} else {
			// If no title has been provided, use the filename
			title = filepath.Base(filename)
		}
	}

	// Find the theme that should be used
	theme := given["theme"]
	if theme == "" {
		theme = defaultTheme
	}

	// Check if a specific string should be replaced with the current theme
	replaceWithTheme := given["replace_with_theme"]
	if replaceWithTheme != "" {
		// Replace all instances of the value given with "replace_with_theme: ..." with the current theme name
		htmlbody = strings.Replace(htmlbody, replaceWithTheme, theme, everyInstance)
	}

	// If the theme is a filename, create a custom theme where the file is imported from the CSS
	if strings.Contains(theme, ".") {
		builtinThemes[theme] = "@import url(" + theme + ");"
		defaultCodeStyles[theme] = defaultCustomCodeStyle
	}

	var head bytes.Buffer

	// If style.gcss is present, use that style in <head>
	GCSSfilename := filepath.Join(filepath.Dir(filename), defaultStyleFilename)
	if fs.exists(GCSSfilename) {
		if debugMode {
			gcssblock, err := cache.read(GCSSfilename, shouldCache(".gcss"))
			if err != nil {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
				return
			}
			gcssdata := gcssblock.MustData()
			// Try compiling the GCSS file first
			if err := validGCSS(gcssdata); err != nil {
				// Invalid GCSS, return an error page
				prettyError(w, req, GCSSfilename, gcssdata, err.Error(), "gcss")
				return
			}
		}
		// Link to stylesheet (without checking if the GCSS file is valid first)
		head.WriteString(`<link href="` + defaultStyleFilename + `" rel="stylesheet" type="text/css">`)
	} else {
		// If not, use the theme by inserting the CSS style directly
		head.WriteString("<style>" + builtinThemes[theme] + "</style>")
	}

	// Add syntax highlighting
	codeStyle := given["codestyle"]
	if codeStyle == "" {
		head.WriteString(highlightHTML(defaultCodeStyles[theme]))
	} else {
		head.WriteString(highlightHTML(codeStyle))
	}
	htmlbody = highlightHTMLcode(htmlbody)

	// Add meta tags, if metadata information has been declared
	for _, keyword := range metaKeywords {
		if given[keyword] != "" {
			// Add the meta tag
			head.WriteString(fmt.Sprintf(`<meta name="%s" content="%s" />`, keyword, given[keyword]))
		}
	}

	// Load the default font in <head>
	//head.WriteString(builtinExtraHTML[defaultTheme])

	// Embed the style and rendered markdown into a simple HTML 5 page
	htmldata := []byte(fmt.Sprintf("<!doctype html><html><head><title>%s</title>%s<head><body><h1>%s</h1>%s</body></html>", title, head.String(), h1title, htmlbody))

	// If the auto-refresh feature has been enabled
	if autoRefreshMode {
		// Insert JavaScript for refreshing the page into the generated HTML
		htmldata = insertAutoRefresh(req, htmldata)
	}

	// Write the rendered Markdown page to the client
	NewDataBlock(htmldata).ToClient(w, req)
}

// Check if the given data is valid GCSS
func validGCSS(gcssdata []byte) error {
	buf := bytes.NewBuffer(gcssdata)
	var w bytes.Buffer
	_, err := gcss.Compile(&w, buf)
	return err
}

// Write the given source bytes as Amber converted to HTML, to a writer.
// filename and luafilename are only used if there are errors.
func pongoPage(w http.ResponseWriter, req *http.Request, filename, luafilename string, pongodata []byte, funcs template.FuncMap, cache *FileCache) {

	var buf bytes.Buffer

	linkInGCSS := false
	// If style.gcss is present, and a header is present, and it has not already been linked in, link it in
	GCSSfilename := filepath.Join(filepath.Dir(filename), defaultStyleFilename)
	if fs.exists(GCSSfilename) {
		if debugMode {
			gcssblock, err := cache.read(GCSSfilename, shouldCache(".gcss"))
			if err != nil {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
				return
			}
			gcssdata := gcssblock.MustData()
			// Try compiling the GCSS file before the Amber file
			if err := validGCSS(gcssdata); err != nil {
				// Invalid GCSS, return an error page
				prettyError(w, req, GCSSfilename, gcssdata, err.Error(), "gcss")
				return
			}
		}
		linkInGCSS = true
	}

	// Prepare a Pongo2 template
	tpl, err := pongo2.DefaultSet.FromBytes(pongodata)
	if err != nil {
		if debugMode {
			prettyError(w, req, filename, pongodata, err.Error(), "pongo2")
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
			if debugMode {
				prettyError(w, req, filename, pongodata, errmsg, "pongo2")
			} else {
				log.Errorf("Could not execute Pongo2 template:\n%s", errmsg)
			}
		}
		return
	}()

	// Render the Pongo2 template to the buffer
	if err := tpl.ExecuteWriter(pongo2.Globals, &buf); err != nil {
		//if err := tpl.ExecuteWriterUnbuffered(pongo2.Globals, &buf); err != nil {
		if debugMode {
			prettyError(w, req, filename, pongodata, err.Error(), "pongo2")
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
			linkToStyleHTML(&htmldata, defaultStyleFilename)
			buf.Reset()
			_, err := buf.Write(htmldata)
			if err != nil {
				if debugMode {
					prettyError(w, req, filename, pongodata, err.Error(), "pongo2")
				} else {
					log.Errorf("Can not write bytes to a buffer! Out of memory?\n%s", err)
				}
				return
			}
		}

		// If the auto-refresh feature has been enabled
		if autoRefreshMode {
			// Insert JavaScript for refreshing the page into the generated HTML
			changedBytes := insertAutoRefresh(req, buf.Bytes())

			buf.Reset()
			_, err := buf.Write(changedBytes)
			if err != nil {
				if debugMode {
					prettyError(w, req, filename, pongodata, err.Error(), "pongo2")
				} else {
					log.Errorf("Can not write bytes to a buffer! Out of memory?\n%s", err)
				}
				return
			}
		}

		// If doctype is missing, add doctype for HTML5 at the top
		changedBytes := insertDoctype(buf.Bytes())
		buf.Reset()
		buf.Write(changedBytes)

	}

	// Write the rendered template to the client
	NewDataBlock(buf.Bytes()).ToClient(w, req)
}

// Write the given source bytes as Amber converted to HTML, to a writer.
// filename and luafilename are only used if there are errors.
func amberPage(w http.ResponseWriter, req *http.Request, filename, luafilename string, amberdata []byte, funcs template.FuncMap, cache *FileCache) {

	var buf bytes.Buffer

	// If style.gcss is present, and a header is present, and it has not already been linked in, link it in
	GCSSfilename := filepath.Join(filepath.Dir(filename), defaultStyleFilename)
	if fs.exists(GCSSfilename) {
		if debugMode {
			gcssblock, err := cache.read(GCSSfilename, shouldCache(".gcss"))
			if err != nil {
				fmt.Fprintf(w, "Unable to read %s: %s", filename, err)
				return
			}
			gcssdata := gcssblock.MustData()
			// Try compiling the GCSS file before the Amber file
			if err := validGCSS(gcssdata); err != nil {
				// Invalid GCSS, return an error page
				prettyError(w, req, GCSSfilename, gcssdata, err.Error(), "gcss")
				return
			}
		}
		// Link to stylesheet (without checking if the GCSS file is valid first)
		linkToStyle(&amberdata, defaultStyleFilename)
	}

	// Compile the given amber template
	tpl, err := amber.CompileData(amberdata, filename, amber.Options{PrettyPrint: true, LineNumbers: false})
	if err != nil {
		if debugMode {
			prettyError(w, req, filename, amberdata, err.Error(), "amber")
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
			if debugMode {
				prettyError(w, req, filename, amberdata, errortext, "amber")
			} else {
				errortext = strings.Replace(errortext, "<br>", "\n", 1)
				log.Errorf("Could not execute Amber template:\n%s", errortext)
			}
		} else {
			if debugMode {
				prettyError(w, req, filename, amberdata, err.Error(), "amber")
			} else {
				log.Errorf("Could not execute Amber template:\n%s", err)
			}
		}
		return
	}

	// If the auto-refresh feature has been enabled
	if autoRefreshMode {
		// Insert JavaScript for refreshing the page into the generated HTML
		changedBytes := insertAutoRefresh(req, buf.Bytes())

		buf.Reset()
		_, err := buf.Write(changedBytes)
		if err != nil {
			if debugMode {
				prettyError(w, req, filename, amberdata, err.Error(), "amber")
			} else {
				log.Errorf("Can not write bytes to a buffer! Out of memory?\n%s", err)
			}
			return
		}
	}

	// If doctype is missing, add doctype for HTML5 at the top
	changedBuf := bytes.NewBuffer(insertDoctype(buf.Bytes()))
	buf = *changedBuf

	// Write the rendered template to the client
	NewDataBlock(buf.Bytes()).ToClient(w, req)
}

// Write the given source bytes as GCSS converted to CSS, to a writer.
// filename is only used if there are errors.
func gcssPage(w http.ResponseWriter, req *http.Request, filename string, gcssdata []byte) {
	var buf bytes.Buffer
	if _, err := gcss.Compile(&buf, bytes.NewReader(gcssdata)); err != nil {
		if debugMode {
			fmt.Fprintf(w, "Could not compile GCSS:\n\n%s\n%s", err, string(gcssdata))
		} else {
			log.Errorf("Could not compile GCSS:\n%s\n%s", err, string(gcssdata))
		}
		return
	}
	// Write the resulting CSS to the client
	NewDataBlock(buf.Bytes()).ToClient(w, req)
}

func jsxPage(w http.ResponseWriter, req *http.Request, filename string, jsxdata []byte) {
	prog, err := parser.ParseFile(nil, filename, jsxdata, parser.IgnoreRegExpErrors)
	if err != nil {
		if debugMode {
			prettyError(w, req, filename, jsxdata, err.Error(), "jsx")
		} else {
			log.Errorf("Could not compile JSX:\n%s\n%s", err, string(jsxdata))
		}
		return
	}
	gen, err := generator.Generate(prog)
	if err != nil {
		if debugMode {
			prettyError(w, req, filename, jsxdata, err.Error(), "jsx")
		} else {
			log.Errorf("Could not generate javascript:\n%s\n%s", err, string(jsxdata))
		}
		return
	}
	if gen != nil {
		data, err := ioutil.ReadAll(gen)
		if err != nil {
			log.Error("Could not read bytes from JSX generator:", err)
			return
		}
		// Write the generated data to the client
		NewDataBlock(data).ToClient(w, req)
	}
}

// Write the given source bytes as SCSS converted to CSS, to a writer.
// filename is only used if there are errors.
func scssPage(w http.ResponseWriter, req *http.Request, filename string, scssdata []byte) {
	// Silence the compiler output
	o := Output{}
	o.disable()
	// Compile the given filename. Sass might want to import other file, which is probably
	// why the Sass compiler doesn't support just taking in a slice of bytes.
	cssString, err := compiler.Run(filename)
	o.enable()
	if err != nil {
		if debugMode {
			fmt.Fprintf(w, "Could not compile SCSS:\n\n%s\n%s", err, string(scssdata))
		} else {
			log.Errorf("Could not compile SCSS:\n%s\n%s", err, string(scssdata))
		}
		return
	}
	// Write the resulting CSS to the client
	NewDataBlock([]byte(cssString)).ToClient(w, req)
}
