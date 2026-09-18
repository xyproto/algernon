package html

import (
	"bytes"
	"fmt"
	"io"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
)

func prevVisibleBlock(n ast.Node) ast.Node {
	for prev := ast.GetPrevNode(n); prev != nil; prev = ast.GetPrevNode(prev) {
		if _, hidden := prev.(*ast.ReferenceDefinition); !hidden {
			return prev
		}
	}
	return nil
}

func (r *Renderer) paragraphEnter(w io.Writer, para *ast.Paragraph) {
	// Preserve spacing between visible block-level siblings.
	prev := prevVisibleBlock(para)
	if prev != nil {
		switch prev.(type) {
		case *ast.HTMLBlock, *ast.List, *ast.Paragraph, *ast.Heading, *ast.CaptionFigure, *ast.CodeBlock, *ast.BlockQuote, *ast.Aside, *ast.HorizontalRule:
			r.CR(w)
		}
	}

	if prev == nil {
		switch para.Parent.(type) {
		case *ast.BlockQuote, *ast.Aside:
			r.CR(w)
		}
	}

	tag := TagWithAttributes("<"+r.paragraphTag(), BlockAttrs(para))
	r.Outs(w, tag)
}

func (r *Renderer) paragraphExit(w io.Writer, para *ast.Paragraph) {
	r.Outs(w, "</"+r.paragraphTag()+">")
	if !(IsListItem(para.Parent) && ast.GetNextNode(para) == nil) {
		r.CR(w)
	}
}

func (r *Renderer) paragraphTag() string {
	if r.Opts.ParagraphTag != "" {
		return r.Opts.ParagraphTag
	}
	return "p"
}

// Paragraph writes ast.Paragraph node
func (r *Renderer) Paragraph(w io.Writer, para *ast.Paragraph, entering bool) {
	if SkipParagraphTags(para) {
		return
	}
	if entering {
		r.paragraphEnter(w, para)
	} else {
		r.paragraphExit(w, para)
	}
}

// Code writes ast.Code node
func (r *Renderer) Code(w io.Writer, node *ast.Code) {
	r.Outs(w, "<code>")
	EscapeHTML(w, node.Literal)
	r.Outs(w, "</code>")
}

// HTMLBlock write ast.HTMLBlock node
func (r *Renderer) HTMLBlock(w io.Writer, node *ast.HTMLBlock) {
	if r.Opts.Flags&SkipHTML != 0 {
		return
	}
	r.CR(w)
	r.Out(w, node.Literal)
	r.CR(w)
}

func (r *Renderer) EnsureUniqueHeadingID(id string) string {
	for count, found := r.headingIDs[id]; found; count, found = r.headingIDs[id] {
		tmp := fmt.Sprintf("%s-%d", id, count+1)

		if _, tmpFound := r.headingIDs[tmp]; !tmpFound {
			r.headingIDs[id] = count + 1
			id = tmp
		} else {
			id = id + "-1"
		}
	}

	if _, found := r.headingIDs[id]; !found {
		r.headingIDs[id] = 0
	}

	return id
}

func (r *Renderer) MakeUniqueHeadingID(hdr *ast.Heading) string {
	if hdr.HeadingID == "" {
		return ""
	}
	id := r.EnsureUniqueHeadingID(hdr.HeadingID)
	id = r.Opts.HeadingIDPrefix + id + r.Opts.HeadingIDSuffix
	hdr.HeadingID = id
	return id
}

func (r *Renderer) HeadingEnter(w io.Writer, hdr *ast.Heading) {
	var attrs []string
	if hdr.IsTitleblock {
		attrs = append(attrs, `class="title"`)
	}
	if hdr.IsSpecial {
		attrs = append(attrs, `class="special"`)
	}

	if hdr.HeadingID != "" {
		id := r.MakeUniqueHeadingID(hdr)
		attrs = append(attrs, fmt.Sprintf(`id="%s"`, escapeAttr(id)))
	}
	attrs = append(attrs, BlockAttrs(hdr)...)
	attrs = coalesceClassAttrs(attrs)
	r.CR(w)
	r.OutTag(w, HeadingOpenTagFromLevel(hdr.Level), attrs)
}

func (r *Renderer) HeadingExit(w io.Writer, hdr *ast.Heading) {
	r.Outs(w, HeadingCloseTagFromLevel(hdr.Level))
	if !(IsListItem(hdr.Parent) && ast.GetNextNode(hdr) == nil) {
		r.CR(w)
	}
}

// Heading writes ast.Heading node
func (r *Renderer) Heading(w io.Writer, hdr *ast.Heading, entering bool) {
	if entering {
		r.HeadingEnter(w, hdr)
	} else {
		r.HeadingExit(w, hdr)
	}
}

// HorizontalRule writes ast.HorizontalRule node
func (r *Renderer) HorizontalRule(w io.Writer, node *ast.HorizontalRule) {
	r.CR(w)
	r.OutHRTag(w, BlockAttrs(node))
	r.CR(w)
}

// EscapeHTMLCallouts writes html-escaped d to w. It escapes &, <, > and " characters, *but*
// expands callouts <<N>> with the callout HTML, i.e. by calling r.callout() with a newly created
// ast.Callout node.
func (r *Renderer) EscapeHTMLCallouts(w io.Writer, d []byte) {
	ld := len(d)
Parse:
	for i := 0; i < ld; i++ {
		for _, comment := range r.Opts.Comments {
			// try every configured marker, not just the first
			if !bytes.HasPrefix(d[i:], comment) {
				continue
			}

			lc := len(comment)
			if i+lc < ld {
				if id, consumed := parser.IsCallout(d[i+lc:]); consumed > 0 {
					// We have seen a callout
					callout := &ast.Callout{ID: id}
					r.Callout(w, callout)
					i += consumed + lc - 1
					continue Parse
				}
			}
		}

		escSeq := Escaper[d[i]]
		if escSeq != nil {
			w.Write(escSeq)
		} else {
			w.Write(d[i : i+1])
		}
	}
}

// CodeBlock writes ast.CodeBlock node
func (r *Renderer) CodeBlock(w io.Writer, codeBlock *ast.CodeBlock) {
	attrs := appendLanguageAttr(nil, codeBlock.Info)
	attrs = append(attrs, BlockAttrs(codeBlock)...)
	attrs = coalesceClassAttrs(attrs)
	r.CR(w)

	r.Outs(w, "<pre>")
	code := TagWithAttributes("<code", attrs)
	r.Outs(w, code)
	if r.Opts.Comments != nil {
		r.EscapeHTMLCallouts(w, codeBlock.Literal)
	} else {
		EscapeHTML(w, codeBlock.Literal)
	}
	r.Outs(w, "</code></pre>")
	if !IsListItem(codeBlock.Parent) {
		r.CR(w)
	}
}

// Caption writes ast.Caption node
func (r *Renderer) Caption(w io.Writer, caption *ast.Caption, entering bool) {
	r.OutOneOf(w, entering, "<figcaption>", "</figcaption>")
}

// CaptionFigure writes ast.CaptionFigure node
func (r *Renderer) CaptionFigure(w io.Writer, figure *ast.CaptionFigure, entering bool) {
	var attrs []string
	if figure.HeadingID != "" {
		attrs = append(attrs, `id="`+escapeAttr(figure.HeadingID)+`"`)
	}
	r.OutOneOf(w, entering, TagWithAttributes("<figure", attrs), "\n</figure>\n")
}

// TableCell writes ast.TableCell node
func (r *Renderer) TableCell(w io.Writer, tableCell *ast.TableCell, entering bool) {
	if !entering {
		r.OutOneOf(w, tableCell.IsHeader, "</th>", "</td>")
		r.CR(w)
		return
	}

	// entering
	var attrs []string
	openTag := "<td"
	if tableCell.IsHeader {
		openTag = "<th"
	}
	align := tableCell.Align.String()
	if align != "" {
		attrs = append(attrs, fmt.Sprintf(`align="%s"`, align))
	}
	if colspan := tableCell.ColSpan; colspan > 0 {
		attrs = append(attrs, fmt.Sprintf(`colspan="%d"`, colspan))
	}
	if ast.GetPrevNode(tableCell) == nil {
		r.CR(w)
	}
	r.OutTag(w, openTag, attrs)
}

// TableBody writes ast.TableBody node
func (r *Renderer) TableBody(w io.Writer, node *ast.TableBody, entering bool) {
	if entering {
		r.CR(w)
		r.Outs(w, "<tbody>")
		// Preserve the historical newline for empty table bodies.
		if ast.GetFirstChild(node) == nil {
			r.CR(w)
		}
	} else {
		r.Outs(w, "</tbody>")
		r.CR(w)
	}
}

// DocumentMatter writes ast.DocumentMatter
func (r *Renderer) DocumentMatter(w io.Writer, node *ast.DocumentMatter, entering bool) {
	if !entering {
		return
	}
	if r.documentMatter != ast.DocumentMatterNone {
		r.Outs(w, "</section>\n")
	}
	switch node.Matter {
	case ast.DocumentMatterFront:
		r.Outs(w, `<section data-matter="front">`)
	case ast.DocumentMatterMain:
		r.Outs(w, `<section data-matter="main">`)
	case ast.DocumentMatterBack:
		r.Outs(w, `<section data-matter="back">`)
	}
	r.documentMatter = node.Matter
}

// Citation writes ast.Citation node
func (r *Renderer) Citation(w io.Writer, node *ast.Citation) {
	for i, c := range node.Destination {
		attr := []string{`class="none"`}
		switch node.Type[i] {
		case ast.CitationTypeNormative:
			attr[0] = `class="normative"`
		case ast.CitationTypeInformative:
			attr[0] = `class="informative"`
		case ast.CitationTypeSuppressed:
			attr[0] = `class="suppressed"`
		}
		r.OutTag(w, "<cite", attr)
		// Escape the destination for both the href attribute and the visible
		// text so untrusted [@dest] cannot inject markup (GHSA-g6w5-3rhc-c379).
		dest := escapeAttr(string(c))
		r.Outs(w, fmt.Sprintf(`<a href="#%s">`+r.Opts.CitationFormatString+`</a>`, dest, dest))
		r.Outs(w, "</cite>")
	}
}

// Callout writes ast.Callout node
func (r *Renderer) Callout(w io.Writer, node *ast.Callout) {
	r.OutTag(w, "<span", []string{`class="callout"`})
	r.Out(w, node.ID)
	r.Outs(w, "</span>")
}

// Index writes ast.Index node
func (r *Renderer) Index(w io.Writer, node *ast.Index) {
	// there is no in-text representation.
	r.OutTag(w, "<span", []string{`class="index"`, fmt.Sprintf(`id="%s"`, node.ID)})
	r.Outs(w, "</span>")
}
