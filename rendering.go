package main

import (
	"bytes"
	"fmt"
	"github.com/eknkc/amber"
	"github.com/russross/blackfriday"
	"github.com/yuin/gopher-lua"
	"net/http"
	"strings"
)

// TODO: Check if handling "# title <tags" on the first line is valid Markdown or not.
//       Submit a patch to blackfriday if it is.

// Wraps converted Markdown HTML in a simple HTML page
func markdownPage(title, htmlbody string) string {
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
	return "<!doctype html><html><head><title>" + title + "</title><style>" + style + "</style><head><body><h1>" + h1title + "</h1>" + htmlbody + "</body></html>"
}

// Expose functions that are related to rendering text, to the given Lua state
func exportRenderFunctions(w http.ResponseWriter, req *http.Request, L *lua.LState) {

	// Print markdown text as html
	L.SetGlobal("mprint", L.NewFunction(func(L *lua.LState) int {
		var buf bytes.Buffer
		top := L.GetTop()
		for i := 1; i <= top; i++ {
			buf.WriteString(L.Get(i).String())
			if i != top {
				buf.WriteString(" ")
			}
		}
		buf.WriteString("\n")
		// Convert the buffer to markdown and return the translated string
		w.Write(blackfriday.MarkdownCommon([]byte(buf.String())))
		return 0 // number of results
	}))

	// Print text as rendered amber.
	// TODO: Either caching or separate template compilation and execution is needed to make this faster.
	L.SetGlobal("aprint", L.NewFunction(func(L *lua.LState) int {
		var buf bytes.Buffer
		top := L.GetTop()
		for i := 1; i <= top; i++ {
			buf.WriteString(L.Get(i).String())
			if i != top {
				buf.WriteString(" ")
			}
		}
		buf.WriteString("\n")

		// Use the buffer as a template.
		// Options are "Pretty printing, but without line numbers."
		tpl, err := amber.Compile(buf.String(), amber.Options{true, false})
		if err != nil {
			// TODO: Log to browser or console depending on the debug mode.
			fmt.Fprint(w, "Could not compile amber template:\n\t"+err.Error()+"\n\n"+buf.String())
			return 0
		}
		//somedata := map[string]string{"": ""}
		tpl.Execute(w, nil)
		return 0 // number of results
	}))

	// TODO: Add two functions. One to compile amber templates and store the result by filename and
	//       one to render data by using a compiled template.

}
