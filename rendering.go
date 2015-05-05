package main

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/mamaar/risotto/generator"
	"github.com/mamaar/risotto/parser"
	"github.com/russross/blackfriday"
	"github.com/xyproto/amber"
	"github.com/yosssi/gcss"
	"github.com/yuin/gopher-lua"
	"html/template"
	"io"
	"net/http"
	"path"
	"strings"
)

// Default stylesheet filename (GCSS)
const defaultStyleFilename = "style.gcss"

// Expose functions that are related to rendering text, to the given Lua state
func exportRenderFunctions(w http.ResponseWriter, req *http.Request, L *lua.LState) {

	// Output Markdown as HTML
	L.SetGlobal("mprint", L.NewFunction(func(L *lua.LState) int {
		// Retrieve all the function arguments as a bytes.Buffer
		buf := arguments2buffer(L)
		// Convert the buffer to markdown and return the translated string
		w.Write(blackfriday.MarkdownCommon([]byte(buf.String())))
		return 0 // number of results
	}))

	// TODO: Add two functions. One to compile amber templates and
	// store the result by filename and one to render data by using
	// compiled templates.

	// Output text as rendered amber.
	// TODO: Add caching, compilation and reuse
	L.SetGlobal("aprint", L.NewFunction(func(L *lua.LState) int {
		// Retrieve all the function arguments as a bytes.Buffer
		buf := arguments2buffer(L)

		// Use the buffer as a template.
		// Options are "Pretty printing, but without line numbers."
		tpl, err := amber.Compile(buf.String(), amber.Options{true, false})
		if err != nil {
			if DEBUG_MODE {
				// TODO: Use a similar error page as for Lua
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

	// Output text as rendered GCSS
	// TODO: Add caching, compilation and reuse
	L.SetGlobal("gprint", L.NewFunction(func(L *lua.LState) int {
		// Retrieve all the function arguments as a bytes.Buffer
		buf := arguments2buffer(L)
		// Transform GCSS to CSS and output the result.
		// Ignoring the number of bytes written.
		if _, err := gcss.Compile(w, bytes.NewReader(buf.Bytes())); err != nil {
			if DEBUG_MODE {
				// TODO: Use a similar error page as for Lua
				fmt.Fprint(w, "Could not compile GCSS:\n\t"+err.Error()+"\n\n"+buf.String())
			} else {
				log.Errorf("Could not compile GCSS:\n%s\n%s", err, buf.String())
			}
			//return 0 // number of results
		}
		return 0 // number of results
	}))

}

// Write the given source bytes as markdown wrapped in HTML to a writer, with a title
func markdownPage(w io.Writer, b []byte, filename string) {

	var title string

	// If the first line is "title: ...", use that as the title
	// and don't convert it to Markdown. This is a subset of MultiMarkdown.
	if bytes.HasPrefix(b, []byte("title:")) {
		fields := bytes.Split(b, []byte("\n"))
		if len(fields) > 1 {
			// Replace the title with the found title
			title = strings.TrimSpace(string(fields[0])[6:])
			// Remove the first line
			b = b[len(fields[0]):]
		}
	}

	// Convert from Markdown to HTML
	htmlbody := string(blackfriday.MarkdownCommon(b))

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

	// If there is no title, use the h1title
	if title == "" {
		if h1title != "" {
			title = h1title
		} else {
			// If no title has been provided, use the filename
			title = path.Base(filename)
		}
	}

	// If style.gcss is present, use that style in <head>
	var markdownStyle string
	if exists(path.Join(path.Dir(filename), defaultStyleFilename)) {
		markdownStyle = `<link href="` + defaultStyleFilename + `" rel="stylesheet" type="text/css">`
	} else {
		// If not, use the default style in <head>
		markdownStyle = "<style>" + defaultStyle + "</style>"
	}

	// Embed the style and rendered markdown into a simple HTML 5 page
	htmlbytes := []byte("<!doctype html><html><head><title>" + title + "</title>" + markdownStyle + "<head><body><h1>" + h1title + "</h1>" + htmlbody + "</body></html>")

	// Write the rendered Markdown page to the http.ResponseWriter
	w.Write(htmlbytes)
}

// Write the given source bytes as Amber converted to HTML, to a writer.
// filename and luafilename are only used if there are errors.
func amberPage(w http.ResponseWriter, filename, luafilename string, amberdata []byte, funcs template.FuncMap) {

	var buf bytes.Buffer

	// If style.gcss is present, and a header is present, and it has not already been linked in, link it in
	if exists(path.Join(path.Dir(filename), defaultStyleFilename)) {
		linkToStyle(&amberdata, defaultStyleFilename)
	}

	// Compile the given amber template
	tpl, err := amber.CompileData(amberdata, filename, amber.Options{true, false})
	if err != nil {
		if DEBUG_MODE {
			prettyError(w, filename, amberdata, err.Error(), "amber")
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
			if DEBUG_MODE {
				prettyError(w, filename, amberdata, errortext, "amber")
			} else {
				errortext = strings.Replace(errortext, "<br>", "\n", 1)
				log.Errorf("Could not execute Amber template:\n%s", errortext)
			}
		} else {
			if DEBUG_MODE {
				prettyError(w, filename, amberdata, err.Error(), "amber")
			} else {
				log.Errorf("Could not execute Amber template:\n%s", err)
			}
		}
		return
	}
	// Write the rendered template to the http.ResponseWriter
	buf.WriteTo(w)
}

// Write the given source bytes as GCSS converted to CSS, to a writer.
// filename is only used if there are errors.
func gcssPage(w http.ResponseWriter, filename string, gcssdata []byte) {
	if _, err := gcss.Compile(w, bytes.NewReader(gcssdata)); err != nil {
		if DEBUG_MODE {
			prettyError(w, filename, gcssdata, err.Error(), "gcss")
		} else {
			log.Errorf("Could not compile GCSS:\n%s\n%s", err, string(gcssdata))
		}
		return
	}
}

func jsxPage(w http.ResponseWriter, filename string, jsxdata []byte) {
	prog, err := parser.ParseFile(nil, filename, jsxdata, parser.IgnoreRegExpErrors)
	if err != nil {
		if DEBUG_MODE {
			prettyError(w, filename, jsxdata, err.Error(), "jsx")
		} else {
			log.Errorf("Could not compile JSX:\n%s\n%s", err, string(jsxdata))
		}
		return
	}
	gen, err := generator.Generate(prog)
	if err != nil {
		if DEBUG_MODE {
			prettyError(w, filename, jsxdata, err.Error(), "jsx")
		} else {
			log.Errorf("Could not generate javascript:\n%s\n%s", err, string(jsxdata))
		}
		return
	}
	if gen != nil {
		//w.Header().Add("Content-type", "text/javascript")
		io.Copy(w, gen)
	}
}
