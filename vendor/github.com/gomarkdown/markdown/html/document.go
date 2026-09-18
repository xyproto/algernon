package html

import (
	"bytes"
	"fmt"
	"io"

	"github.com/gomarkdown/markdown/ast"
)

// RenderHeader writes HTML document preamble and TOC if requested.
func (r *Renderer) RenderHeader(w io.Writer, doc ast.Node) {
	r.writeDocumentHeader(w)
	if r.Opts.Flags&TOC != 0 {
		r.writeTOC(w, doc)
	}
}

// RenderFooter writes HTML document footer.
func (r *Renderer) RenderFooter(w io.Writer, _ ast.Node) {
	if r.documentMatter != ast.DocumentMatterNone {
		r.Outs(w, "</section>\n")
	}
	if r.Opts.Flags&CompletePage != 0 {
		io.WriteString(w, "\n</body>\n</html>\n")
	}
}

func (r *Renderer) writeDocumentHeader(w io.Writer) {
	if r.Opts.Flags&CompletePage == 0 {
		return
	}
	ending := ""
	if r.Opts.Flags&UseXHTML != 0 {
		io.WriteString(w, "<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.0 Transitional//EN\" ")
		io.WriteString(w, "\"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd\">\n")
		io.WriteString(w, "<html xmlns=\"http://www.w3.org/1999/xhtml\">\n")
		ending = " /"
	} else {
		io.WriteString(w, "<!DOCTYPE html>\n<html>\n")
	}
	io.WriteString(w, "<head>\n  <title>")
	if r.Opts.Flags&Smartypants != 0 {
		r.sr.Process(w, []byte(r.Opts.Title))
	} else {
		EscapeHTML(w, []byte(r.Opts.Title))
	}
	io.WriteString(w, "</title>\n")
	io.WriteString(w, r.Opts.Generator)
	io.WriteString(w, "\"")
	io.WriteString(w, ending)
	io.WriteString(w, ">\n  <meta charset=\"utf-8\"")
	io.WriteString(w, ending)
	io.WriteString(w, ">\n")
	if r.Opts.CSS != "" {
		writeHeadLink(w, "stylesheet", "text/css", r.Opts.CSS, ending)
	}
	if r.Opts.Icon != "" {
		writeHeadLink(w, "icon", "image/x-icon", r.Opts.Icon, ending)
	}
	if r.Opts.Head != nil {
		w.Write(r.Opts.Head)
	}
	io.WriteString(w, "</head>\n<body>\n\n")
}

func writeHeadLink(w io.Writer, rel, typ, href, ending string) {
	fmt.Fprintf(w, `  <link rel="%s" type="%s" href="`, rel, typ)
	EscapeHTML(w, []byte(href))
	io.WriteString(w, `"`+ending+">\n")
}

func (r *Renderer) writeTOC(w io.Writer, doc ast.Node) {
	var buf bytes.Buffer
	inHeading := false
	tocLevel, headingCount := 0, 0

	ast.WalkFunc(doc, func(node ast.Node, entering bool) ast.WalkStatus {
		heading, ok := node.(*ast.Heading)
		if ok && !heading.IsTitleblock {
			inHeading = entering
			if !entering {
				buf.WriteString("</a>")
				return ast.GoToNext
			}
			if heading.HeadingID == "" {
				heading.HeadingID = fmt.Sprintf("toc_%d", headingCount)
			}
			switch {
			case heading.Level == tocLevel:
				buf.WriteString("</li>\n\n<li>")
			case heading.Level < tocLevel:
				for heading.Level < tocLevel {
					tocLevel--
					buf.WriteString("</li>\n</ul>")
				}
				buf.WriteString("</li>\n\n<li>")
			default:
				for heading.Level > tocLevel {
					tocLevel++
					buf.WriteString("\n<ul>\n<li>")
				}
			}
			fmt.Fprintf(&buf, `<a href="#%s">`, escapeAttr(heading.HeadingID))
			headingCount++
			return ast.GoToNext
		}
		if inHeading {
			return r.RenderNode(&buf, node, entering)
		}
		return ast.GoToNext
	})

	for ; tocLevel > 0; tocLevel-- {
		buf.WriteString("</li>\n</ul>")
	}
	tocLen := buf.Len()
	if tocLen > 0 {
		io.WriteString(w, "<nav>\n")
		buf.WriteTo(w)
		io.WriteString(w, "\n\n</nav>\n")
	}
	r.lastOutputLen = tocLen
}
