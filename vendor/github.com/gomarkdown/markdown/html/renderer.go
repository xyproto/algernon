package html

import (
	"bytes"
	"fmt"
	"io"
	"strconv"

	"github.com/gomarkdown/markdown/ast"
)

func (r *Renderer) OutTag(w io.Writer, name string, attrs []string) {
	r.Outs(w, TagWithAttributes(name, attrs))
}

func FootnoteRef(prefix string, node *ast.Link) string {
	urlFrag := prefix + string(Slugify(node.Destination))
	nStr := strconv.Itoa(node.NoteID)
	anchor := `<a href="#fn:` + urlFrag + `">` + nStr + `</a>`
	return `<sup class="footnote-ref" id="fnref:` + urlFrag + `">` + anchor + `</sup>`
}

func FootnoteItem(prefix string, slug []byte) string {
	return `<li id="fn:` + prefix + string(slug) + `">`
}

func FootnoteReturnLink(prefix, returnLink string, slug []byte) string {
	return ` <a class="footnote-return" href="#fnref:` + prefix + string(slug) + `">` + returnLink + `</a>`
}

func ListItemOpenCR(listItem *ast.ListItem) bool {
	if ast.GetPrevNode(listItem) == nil {
		return false
	}
	ld := listItem.Parent.(*ast.List)
	return !ld.Tight && ld.ListFlags&ast.ListTypeDefinition == 0
}

func SkipParagraphTags(para *ast.Paragraph) bool {
	parent := para.Parent
	list, ok := parent.GetParent().(*ast.List)
	return ok && (list.Tight || IsListItemTerm(parent))
}

// Out is a helper to write data to writer
func (r *Renderer) Out(w io.Writer, d []byte) {
	r.lastOutputLen = len(d)
	if r.DisableTags > 0 {
		d = htmlTagRe.ReplaceAll(d, []byte{})
	}
	w.Write(d)
}

// Outs is a helper to write data to writer
func (r *Renderer) Outs(w io.Writer, s string) {
	r.lastOutputLen = len(s)
	if r.DisableTags > 0 {
		s = htmlTagRe.ReplaceAllString(s, "")
	}
	io.WriteString(w, s)
}

// CR writes a new line
func (r *Renderer) CR(w io.Writer) {
	if r.lastOutputLen > 0 {
		r.Outs(w, "\n")
	}
}

func normalizedHeadingLevel(level int) int {
	if level < 1 || level > 5 {
		return 6
	}
	return level
}

func HeadingOpenTagFromLevel(level int) string {
	return "<h" + strconv.Itoa(normalizedHeadingLevel(level))
}

func HeadingCloseTagFromLevel(level int) string {
	return "</h" + strconv.Itoa(normalizedHeadingLevel(level)) + ">"
}

func (r *Renderer) OutHRTag(w io.Writer, attrs []string) {
	hr := TagWithAttributes("<hr", attrs)
	r.OutOneOf(w, r.Opts.Flags&UseXHTML == 0, hr, "<hr />")
}

// Text writes ast.Text node
func (r *Renderer) Text(w io.Writer, text *ast.Text) {
	if r.Opts.Flags&Smartypants != 0 {
		var tmp bytes.Buffer
		EscapeHTML(&tmp, text.Literal)
		r.sr.Process(w, tmp.Bytes())
	} else {
		_, parentIsLink := text.Parent.(*ast.Link)
		if parentIsLink {
			EscLink(w, text.Literal)
		} else {
			EscapeHTML(w, text.Literal)
		}
	}
}

// HardBreak writes ast.Hardbreak node
func (r *Renderer) HardBreak(w io.Writer, node *ast.Hardbreak) {
	r.OutOneOf(w, r.Opts.Flags&UseXHTML == 0, "<br>", "<br />")
	r.CR(w)
}

// NonBlockingSpace writes ast.NonBlockingSpace node
func (r *Renderer) NonBlockingSpace(w io.Writer, node *ast.NonBlockingSpace) {
	r.Outs(w, "&nbsp;")
}

// OutOneOf writes first or second depending on outFirst
func (r *Renderer) OutOneOf(w io.Writer, outFirst bool, first string, second string) {
	if outFirst {
		r.Outs(w, first)
	} else {
		r.Outs(w, second)
	}
}

// OutOneOfCr writes CR + first or second + CR depending on outFirst
func (r *Renderer) OutOneOfCr(w io.Writer, outFirst bool, first string, second string) {
	if outFirst {
		r.CR(w)
		r.Outs(w, first)
	} else {
		r.Outs(w, second)
		r.CR(w)
	}
}

// HTMLSpan writes ast.HTMLSpan node
func (r *Renderer) HTMLSpan(w io.Writer, span *ast.HTMLSpan) {
	if r.Opts.Flags&SkipHTML == 0 {
		r.Out(w, span.Literal)
	}
}

func (r *Renderer) linkEnter(w io.Writer, link *ast.Link) {
	attrs := link.AdditionalAttributes
	dest := link.Destination
	dest = AddAbsPrefix(dest, r.Opts.AbsolutePrefix)
	var hrefBuf bytes.Buffer
	hrefBuf.WriteString("href=\"")
	EscLink(&hrefBuf, dest)
	hrefBuf.WriteByte('"')
	attrs = append(attrs, hrefBuf.String())
	if link.NoteID != 0 {
		r.Outs(w, FootnoteRef(r.Opts.FootnoteAnchorPrefix, link))
		return
	}

	attrs = appendLinkAttrs(attrs, r.Opts.Flags, dest)
	if len(link.Title) > 0 {
		var titleBuff bytes.Buffer
		titleBuff.WriteString("title=\"")
		EscapeHTML(&titleBuff, link.Title)
		titleBuff.WriteByte('"')
		attrs = append(attrs, titleBuff.String())
	}
	attrs = append(attrs, BlockAttrs(link)...)
	attrs = coalesceClassAttrs(attrs)
	r.OutTag(w, "<a", attrs)
}

func (r *Renderer) linkExit(w io.Writer, link *ast.Link) {
	if link.NoteID == 0 {
		r.Outs(w, "</a>")
	}
}

// Link writes ast.Link node
func (r *Renderer) Link(w io.Writer, link *ast.Link, entering bool) {
	// mark it but don't link it if it is not a safe link: no smartypants
	if needSkipLink(r, link.Destination) {
		r.OutOneOf(w, entering, "<tt>", "</tt>")
		return
	}

	if entering {
		r.linkEnter(w, link)
	} else {
		r.linkExit(w, link)
	}
}

func (r *Renderer) imageEnter(w io.Writer, image *ast.Image) {
	r.DisableTags++
	if r.DisableTags > 1 {
		return
	}
	src := image.Destination
	src = AddAbsPrefixToImage(src, r.Opts.AbsolutePrefix)
	attrs := BlockAttrs(image)
	if r.Opts.Flags&LazyLoadImages != 0 {
		attrs = append(attrs, `loading="lazy"`)
	}

	r.Outs(w, tagStart("<img", attrs)+` src="`)
	EscLink(w, src)
	r.Outs(w, `" alt="`)
}

func (r *Renderer) imageExit(w io.Writer, image *ast.Image) {
	r.DisableTags--
	if r.DisableTags > 0 {
		return
	}
	if image.Title != nil {
		r.Outs(w, `" title="`)
		EscapeHTML(w, image.Title)
	}
	r.Outs(w, `" />`)
}

// Image writes ast.Image node
func (r *Renderer) Image(w io.Writer, node *ast.Image, entering bool) {
	if entering {
		r.imageEnter(w, node)
	} else {
		r.imageExit(w, node)
	}
}

// prevVisibleBlock skips siblings that emit no HTML, such as
// reference definitions, so inter-block spacing stays stable.
// RenderNode renders a markdown node to HTML
func (r *Renderer) RenderNode(w io.Writer, node ast.Node, entering bool) ast.WalkStatus {
	if r.Opts.RenderNodeHook != nil {
		status, didHandle := r.Opts.RenderNodeHook(w, node, entering)
		if didHandle {
			return status
		}
	}
	switch node := node.(type) {
	case *ast.Text:
		r.Text(w, node)
	case *ast.Softbreak:
		r.CR(w)
	case *ast.Hardbreak:
		r.HardBreak(w, node)
	case *ast.NonBlockingSpace:
		r.NonBlockingSpace(w, node)
	case *ast.Emph:
		r.OutOneOf(w, entering, "<em>", "</em>")
	case *ast.Strong:
		r.OutOneOf(w, entering, "<strong>", "</strong>")
	case *ast.Del:
		r.OutOneOf(w, entering, "<del>", "</del>")
	case *ast.BlockQuote:
		tag := TagWithAttributes("<blockquote", BlockAttrs(node))
		r.OutOneOfCr(w, entering, tag, "</blockquote>")
	case *ast.Aside:
		tag := TagWithAttributes("<aside", BlockAttrs(node))
		r.OutOneOfCr(w, entering, tag, "</aside>")
	case *ast.Link:
		r.Link(w, node, entering)
	case *ast.CrossReference:
		link := &ast.Link{Destination: append([]byte("#"), node.Destination...)}
		r.Link(w, link, entering)
	case *ast.Citation:
		r.Citation(w, node)
	case *ast.Image:
		if r.Opts.Flags&SkipImages != 0 {
			return ast.SkipChildren
		}
		r.Image(w, node, entering)
	case *ast.Code:
		r.Code(w, node)
	case *ast.CodeBlock:
		r.CodeBlock(w, node)
	case *ast.Caption:
		r.Caption(w, node, entering)
	case *ast.CaptionFigure:
		r.CaptionFigure(w, node, entering)
	case *ast.Document:
	case *ast.Paragraph:
		r.Paragraph(w, node, entering)
	case *ast.HTMLSpan:
		r.HTMLSpan(w, node)
	case *ast.HTMLBlock:
		r.HTMLBlock(w, node)
	case *ast.Heading:
		r.Heading(w, node, entering)
	case *ast.HorizontalRule:
		r.HorizontalRule(w, node)
	case *ast.List:
		r.List(w, node, entering)
	case *ast.ListItem:
		r.ListItem(w, node, entering)
	case *ast.Table:
		tag := TagWithAttributes("<table", BlockAttrs(node))
		r.OutOneOfCr(w, entering, tag, "</table>")
	case *ast.TableCell:
		r.TableCell(w, node, entering)
	case *ast.TableHeader:
		r.OutOneOfCr(w, entering, "<thead>", "</thead>")
	case *ast.TableBody:
		r.TableBody(w, node, entering)
	case *ast.TableRow:
		r.OutOneOfCr(w, entering, "<tr>", "</tr>")
	case *ast.TableFooter:
		r.OutOneOfCr(w, entering, "<tfoot>", "</tfoot>")
	case *ast.Math:
		r.Outs(w, `<span class="math inline">\(`)
		EscapeHTML(w, node.Literal)
		r.Outs(w, `\)</span>`)
	case *ast.MathBlock:
		r.OutOneOf(w, entering, `<p><span class="math display">\[`, `\]</span></p>`)
		if entering {
			EscapeHTML(w, node.Literal)
		}
	case *ast.DocumentMatter:
		r.DocumentMatter(w, node, entering)
	case *ast.Callout:
		r.Callout(w, node)
	case *ast.Index:
		r.Index(w, node)
	case *ast.Subscript:
		r.Outs(w, "<sub>")
		if entering {
			Escape(w, node.Literal)
		}
		r.Outs(w, "</sub>")
	case *ast.Superscript:
		r.Outs(w, "<sup>")
		if entering {
			Escape(w, node.Literal)
		}
		r.Outs(w, "</sup>")
	case *ast.Footnotes:
		// nothing by default; just output the list.
	case *ast.ReferenceDefinition:
		// Definitions are resolved onto links at parse time; omit from HTML.
	default:
		panic(fmt.Sprintf("Unknown node %T", node))
	}
	return ast.GoToNext
}
