package html

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/gomarkdown/markdown/ast"
)

// RenderHeader writes HTML document preamble and TOC if requested.
func (r *Renderer) RenderHeader(w io.Writer, doc ast.Node) {
	// Each document starts with empty heading ids. Reusing one renderer must
	// not turn the next document's "intro" into "intro-1", and must not keep
	// the previous document's heading nodes alive.
	r.headingIDs = make(map[string]int)
	r.headingAnchors = nil
	root, ok := doc.(*ast.Document)
	r.commonMark = ok && root.CommonMark
	r.writeDocumentHeader(w)
	if r.Opts.Flags&TOC != 0 {
		r.writeTOC(w, doc)
	}
}

// RenderFooter writes HTML document footer.
func (r *Renderer) RenderFooter(w io.Writer, _ ast.Node) {
	r.closeMatterSection(w)
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
	// Escape before smart punctuation. Process leaves a raw '<' in place, and
	// CommonFlags turns smart punctuation on, so a title would otherwise be
	// able to close the <title> element.
	var title bytes.Buffer
	EscapeHTML(&title, []byte(r.Opts.Title))
	if r.Opts.Flags&Smartypants != 0 {
		r.sr.Process(w, title.Bytes())
	} else {
		title.WriteTo(w)
	}
	io.WriteString(w, "</title>\n")
	io.WriteString(w, r.Opts.Generator)
	io.WriteString(w, "\n  <meta charset=\"utf-8\"")
	io.WriteString(w, ending)
	io.WriteString(w, ">\n")
	writeHeadLink(w, "stylesheet", "text/css", r.Opts.CSS, ending)
	writeHeadLink(w, "icon", "image/x-icon", r.Opts.Icon, ending)
	if r.Opts.Head != nil {
		w.Write(r.Opts.Head)
	}
	io.WriteString(w, "</head>\n<body>\n\n")
}

// writeHeadLink writes a <link> to href in the document head, if href is set.
func writeHeadLink(w io.Writer, rel, typ, href, ending string) {
	if href == "" {
		return
	}
	fmt.Fprintf(w, `  <link rel="%s" type="%s" href="`, rel, typ)
	EscapeHTML(w, []byte(href))
	io.WriteString(w, `"`+ending+">\n")
}

// closeTOCLevels closes n levels of nested TOC lists.
func closeTOCLevels(buf *bytes.Buffer, n int) {
	buf.WriteString(strings.Repeat("</li>\n</ul>", n))
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
			// The same id the heading element gets, including the prefix,
			// suffix and any suffix added to keep ids unique.
			headingID := r.MakeUniqueHeadingID(heading)
			if heading.Level > tocLevel {
				buf.WriteString(strings.Repeat("\n<ul>\n<li>", heading.Level-tocLevel))
			} else {
				closeTOCLevels(&buf, tocLevel-heading.Level)
				buf.WriteString("</li>\n\n<li>")
			}
			tocLevel = heading.Level
			fmt.Fprintf(&buf, `<a href="#%s">`, escapeAttr(headingID))
			headingCount++
			return ast.GoToNext
		}
		if inHeading {
			return r.RenderNode(&buf, node, entering)
		}
		return ast.GoToNext
	})

	closeTOCLevels(&buf, tocLevel)
	tocLen := buf.Len()
	if tocLen > 0 {
		io.WriteString(w, "<nav>\n")
		buf.WriteTo(w)
		io.WriteString(w, "\n\n</nav>\n")
	}
	r.lastOutputLen = tocLen
}
