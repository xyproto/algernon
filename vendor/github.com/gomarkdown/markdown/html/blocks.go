package html

import (
	"bytes"
	"fmt"
	"io"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/internal/matter"
	"github.com/gomarkdown/markdown/parser"
)

// emitsHTML reports whether block n produces HTML output; reference
// definitions do not.
func emitsHTML(n ast.Node) bool {
	_, hidden := n.(*ast.ReferenceDefinition)
	return !hidden
}

// prevVisibleBlock returns the previous sibling of n, skipping siblings that
// emit no HTML, such as reference definitions, so inter-block spacing stays
// stable.
func prevVisibleBlock(n ast.Node) ast.Node {
	for prev := ast.GetPrevNode(n); prev != nil; prev = ast.GetPrevNode(prev) {
		if emitsHTML(prev) {
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

	tag := TagWithAttributes("<"+r.paragraphTag(), r.blockAttrs(para))
	r.Outs(w, tag)
}

func (r *Renderer) paragraphExit(w io.Writer, para *ast.Paragraph) {
	r.Outs(w, "</"+r.paragraphTag()+">")
	r.crAfterBlock(w, para)
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
	enterOrExit(r, w, para, entering, (*Renderer).paragraphEnter, (*Renderer).paragraphExit)
}

// Code writes ast.Code node
func (r *Renderer) Code(w io.Writer, node *ast.Code) {
	r.Outs(w, "<code>")
	EscapeHTML(w, node.Literal)
	r.Outs(w, "</code>")
}

// HTMLBlock writes ast.HTMLBlock node
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
	if anchor, ok := r.headingAnchors[hdr]; ok && anchor.base == hdr.HeadingID {
		return anchor.id
	}
	id := r.Opts.HeadingIDPrefix + r.EnsureUniqueHeadingID(hdr.HeadingID) + r.Opts.HeadingIDSuffix
	if r.headingAnchors == nil {
		r.headingAnchors = map[*ast.Heading]headingAnchor{}
	}
	r.headingAnchors[hdr] = headingAnchor{base: hdr.HeadingID, id: id}
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
	attrs = append(attrs, r.blockAttrs(hdr)...)
	attrs = coalesceClassAttrs(attrs)
	r.CR(w)
	r.OutTag(w, HeadingOpenTagFromLevel(hdr.Level), attrs)
}

func (r *Renderer) HeadingExit(w io.Writer, hdr *ast.Heading) {
	r.Outs(w, HeadingCloseTagFromLevel(hdr.Level))
	r.crAfterBlock(w, hdr)
}

// Heading writes ast.Heading node
func (r *Renderer) Heading(w io.Writer, hdr *ast.Heading, entering bool) {
	enterOrExit(r, w, hdr, entering, (*Renderer).HeadingEnter, (*Renderer).HeadingExit)
}

// HorizontalRule writes ast.HorizontalRule node
func (r *Renderer) HorizontalRule(w io.Writer, node *ast.HorizontalRule) {
	r.CR(w)
	r.OutHRTag(w, r.blockAttrs(node))
	r.CR(w)
}

// EscapeHTMLCallouts writes html-escaped d to w. It escapes &, <, > and " characters, *but*
// expands callouts <<N>> with the callout HTML, i.e. by calling r.callout() with a newly created
// ast.Callout node.
func (r *Renderer) EscapeHTMLCallouts(w io.Writer, d []byte) {
	ld := len(d)
	// start is the first byte of the current run of characters that need no escape.
	start := 0
	flush := func(end int) {
		if end > start {
			w.Write(d[start:end])
		}
	}
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
					flush(i)
					r.Callout(w, &ast.Callout{ID: id})
					i += consumed + lc - 1
					start = i + 1
					continue Parse
				}
			}
		}

		if escSeq := Escaper[d[i]]; escSeq != nil {
			flush(i)
			w.Write(escSeq)
			start = i + 1
		}
	}
	flush(ld)
}

// CodeBlock writes ast.CodeBlock node
func (r *Renderer) CodeBlock(w io.Writer, codeBlock *ast.CodeBlock) {
	r.CR(w)
	r.codeBlockElement(w, codeBlock)
	r.crIf(w, !IsListItem(codeBlock.Parent))
}

// codeBlockElement writes the <pre><code> element of a code block, with its
// language and block attributes and its escaped text.
func (r *Renderer) codeBlockElement(w io.Writer, codeBlock *ast.CodeBlock) {
	attrs := appendLanguageAttr(nil, codeBlock.Info)
	attrs = coalesceClassAttrs(append(attrs, r.blockAttrs(codeBlock)...))
	r.Outs(w, "<pre>")
	r.OutTag(w, "<code", attrs)
	if r.Opts.Comments != nil {
		r.EscapeHTMLCallouts(w, codeBlock.Literal)
	} else {
		EscapeHTML(w, codeBlock.Literal)
	}
	r.Outs(w, "</code></pre>")
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
	r.crIf(w, ast.GetPrevNode(tableCell) == nil)
	r.OutTag(w, openTag, attrs)
}

// TableBody writes ast.TableBody node
func (r *Renderer) TableBody(w io.Writer, node *ast.TableBody, entering bool) {
	if entering {
		r.CR(w)
		r.Outs(w, "<tbody>")
		// Preserve the historical newline for empty table bodies.
		r.crIf(w, ast.GetFirstChild(node) == nil)
	} else {
		r.Outs(w, "</tbody>")
		r.CR(w)
	}
}

// closeMatterSection closes the <section> of the current document matter, if
// one is open.
func (r *Renderer) closeMatterSection(w io.Writer) {
	if r.documentMatter != ast.DocumentMatterNone {
		r.Outs(w, "</section>\n")
	}
}

// DocumentMatter writes ast.DocumentMatter
func (r *Renderer) DocumentMatter(w io.Writer, node *ast.DocumentMatter, entering bool) {
	if !entering {
		return
	}
	r.closeMatterSection(w)
	if name := matter.Name(node.Matter); name != "" {
		r.Outs(w, `<section data-matter="`+name+`">`)
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
	EscapeHTML(w, node.ID)
	r.Outs(w, "</span>")
}

// Index writes ast.Index node
func (r *Renderer) Index(w io.Writer, node *ast.Index) {
	// there is no in-text representation.
	r.OutTag(w, "<span", []string{`class="index"`, `id="` + escapeAttr(node.ID) + `"`})
	r.Outs(w, "</span>")
}
