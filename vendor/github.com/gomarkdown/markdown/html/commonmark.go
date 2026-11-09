package html

import (
	"bytes"
	"fmt"
	"io"

	"github.com/gomarkdown/markdown/ast"
)

// CommonMark blocks have explicit line endings instead of the historical
// renderer's extra inter-block spacing. All other nodes use the usual renderer.
func (r *Renderer) renderCommonMarkNode(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
	switch n := node.(type) {
	case *ast.Paragraph:
		if SkipParagraphTags(n) {
			if !entering && ast.GetNextNode(n) != nil {
				r.Outs(w, "\n")
			}
		} else if entering {
			r.OutTag(w, "<"+r.paragraphTag(), r.blockAttrs(n))
		} else {
			r.Outs(w, "</"+r.paragraphTag()+">\n")
		}
	case *ast.Heading:
		if entering {
			var own []string
			id := r.MakeUniqueHeadingID(n)
			if id != "" {
				own = []string{"id"}
			}
			attrs := r.blockAttrs(n, own...)
			if id != "" {
				attrs = append(attrs, `id="`+escapeAttr(id)+`"`)
			}
			r.OutTag(w, HeadingOpenTagFromLevel(n.Level), attrs)
		} else {
			r.Outs(w, HeadingCloseTagFromLevel(n.Level)+"\n")
		}
	case *ast.BlockQuote:
		if entering {
			r.OutTag(w, "<blockquote", r.blockAttrs(n))
			r.Outs(w, "\n")
		} else {
			r.Outs(w, "</blockquote>\n")
		}
	case *ast.List:
		if entering {
			var attrs []string
			if n.ListFlags&ast.ListTypeOrdered != 0 && n.Start != 1 {
				attrs = append(r.blockAttrs(n, "start"), fmt.Sprintf(`start="%d"`, n.Start))
			} else {
				attrs = r.blockAttrs(n)
			}
			r.OutTag(w, "<"+listTag(n.ListFlags), attrs)
			r.Outs(w, "\n")
		} else {
			r.Outs(w, "</"+listTag(n.ListFlags)+">\n")
		}
	case *ast.ListItem:
		if entering {
			r.OutTag(w, "<li", r.blockAttrs(n))
			first := firstCommonMarkBlock(n)
			if para, ok := first.(*ast.Paragraph); first != nil && (!ok || !SkipParagraphTags(para)) {
				r.Outs(w, "\n")
			}
		} else {
			r.Outs(w, "</li>\n")
		}
	case *ast.HorizontalRule:
		r.OutHRTag(w, r.blockAttrs(n))
		r.Outs(w, "\n")
	case *ast.CodeBlock:
		r.codeBlockElement(w, n)
		r.Outs(w, "\n")
	case *ast.HTMLBlock:
		if r.Opts.Flags&SkipHTML == 0 {
			r.Out(w, n.Literal)
			if !bytes.HasSuffix(n.Literal, []byte{'\n'}) {
				r.Outs(w, "\n")
			}
		}
	case *ast.Image:
		if r.Opts.Flags&SkipImages == 0 && entering {
			if needSkipImage(r, n.Destination) {
				r.Outs(w, "<tt>")
				r.commonMarkImageText(w, n)
				r.Outs(w, "</tt>")
				return ast.SkipChildren, true
			}
			attrs := r.blockAttrs(n, imageAttrNames(r, n.Title)...)
			if r.Opts.Flags&LazyLoadImages != 0 {
				attrs = append(attrs, `loading="lazy"`)
			}
			r.Outs(w, tagStart("<img", attrs)+` src="`)
			EscapeHTML(w, AddAbsPrefixToImage(n.Destination, r.Opts.AbsolutePrefix))
			r.Outs(w, `" alt="`)
			r.commonMarkImageText(w, n)
			r.Outs(w, `"`)
			if n.Title != nil {
				r.Outs(w, ` title="`)
				EscapeHTML(w, n.Title)
				r.Outs(w, `"`)
			}
			r.Outs(w, r.closeTag)
		}
		return ast.SkipChildren, true
	default:
		return ast.GoToNext, false
	}
	return ast.GoToNext, true
}

func firstCommonMarkBlock(parent ast.Node) ast.Node {
	for _, child := range parent.GetChildren() {
		if _, hidden := child.(*ast.ReferenceDefinition); !hidden {
			return child
		}
	}
	return nil
}

func (r *Renderer) commonMarkImageText(w io.Writer, image ast.Node) {
	ast.WalkFunc(image, func(node ast.Node, entering bool) ast.WalkStatus {
		if entering {
			switch n := node.(type) {
			case *ast.Text:
				EscapeHTML(w, n.Literal)
			case *ast.Code:
				EscapeHTML(w, n.Literal)
			case *ast.Hardbreak:
				r.HardBreak(w, n)
			case *ast.Softbreak:
				r.Outs(w, "\n")
			}
		}
		return ast.GoToNext
	})
}
