package main

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/eknkc/amber"
	"github.com/russross/blackfriday"
	"github.com/yosssi/gcss"
	"github.com/yuin/gopher-lua"
	"html/template"
	"io"
	"net/http"
	"path"
	"strings"
)

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
		// Using "MISSING" instead of nil for slightly better error messages
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
func markdownPage(w io.Writer, b []byte, title string) {

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

	htmlbytes := []byte("<!doctype html><html><head><title>" + title + "</title><style>" + style + "</style><head><body><h1>" + h1title + "</h1>" + htmlbody + "</body></html>")

	// Write the rendered Markdown page to the http.ResponseWriter
	w.Write(htmlbytes)
}

// Write the given source bytes as Amber converted to HTML, to a writer.
// filename and luafilename are only used if there are errors.
func amberPage(w http.ResponseWriter, filename, luafilename string, amberdata []byte, funcs template.FuncMap) {

	var buf bytes.Buffer

	// If style.gcss is present, and a header is present, and it has not already been linked in, link it in
	if exists(path.Join(path.Dir(filename), "style.gcss")) {
		linkToStyle(&amberdata, "style.gcss")
	}

	// Compile the given amber template
	tpl, err := amber.Compile(string(amberdata), amber.Options{true, false})
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
		if DEBUG_MODE {
			// TODO: Use a pretty error page, similar to the one for Lua
			if strings.TrimSpace(err.Error()) == "reflect: call of reflect.Value.Type on zero Value" {

				// If there were errors, display an error page
				errortext := "Could not execute Amber template!<br>One of the functions called by the template is not available."
				// Default title
				prettyError(w, filename, amberdata, errortext, "amber")
			} else {
				// Default title
				prettyError(w, filename, amberdata, err.Error(), "amber")
			}
		} else {
			log.Errorf("Could not execute Amber template:\n%s", err)
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
