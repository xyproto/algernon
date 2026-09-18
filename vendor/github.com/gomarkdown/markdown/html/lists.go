package html

import (
	"fmt"
	"io"

	"github.com/gomarkdown/markdown/ast"
)

func listTag(flags ast.ListType) string {
	switch {
	case flags&ast.ListTypeDefinition != 0:
		return "dl"
	case flags&ast.ListTypeOrdered != 0:
		return "ol"
	default:
		return "ul"
	}
}

func listItemTag(flags ast.ListType) string {
	switch {
	case flags&ast.ListTypeTerm != 0:
		return "dt"
	case flags&ast.ListTypeDefinition != 0:
		return "dd"
	default:
		return "li"
	}
}

func (r *Renderer) listEnter(w io.Writer, nodeData *ast.List) {
	var attrs []string

	if nodeData.IsFootnotesList {
		r.Outs(w, "\n<div class=\"footnotes\">\n\n")
		if r.Opts.Flags&FootnoteNoHRTag == 0 {
			r.OutHRTag(w, nil)
			r.CR(w)
		}
	}
	r.CR(w)
	if IsListItem(nodeData.Parent) {
		grand := nodeData.Parent.GetParent()
		if IsListTight(grand) {
			r.CR(w)
		}
	}

	if nodeData.ListFlags&ast.ListTypeOrdered != 0 {
		if nodeData.Start > 0 {
			attrs = append(attrs, fmt.Sprintf(`start="%d"`, nodeData.Start))
		}
	}
	attrs = append(attrs, BlockAttrs(nodeData)...)
	r.OutTag(w, "<"+listTag(nodeData.ListFlags), attrs)
	r.CR(w)
}

func (r *Renderer) listExit(w io.Writer, list *ast.List) {
	r.Outs(w, "</"+listTag(list.ListFlags)+">")
	parent := list.Parent
	switch parent.(type) {
	case *ast.ListItem:
		if ast.GetNextNode(list) != nil {
			r.CR(w)
		}
	case *ast.Document, *ast.BlockQuote, *ast.Aside:
		r.CR(w)
	}

	if list.IsFootnotesList {
		r.Outs(w, "\n</div>\n")
	}
}

// List writes ast.List node
func (r *Renderer) List(w io.Writer, list *ast.List, entering bool) {
	if entering {
		r.listEnter(w, list)
	} else {
		r.listExit(w, list)
	}
}

func (r *Renderer) listItemEnter(w io.Writer, listItem *ast.ListItem) {
	if ListItemOpenCR(listItem) {
		r.CR(w)
	}
	if listItem.RefLink != nil {
		slug := Slugify(listItem.RefLink)
		r.Outs(w, FootnoteItem(r.Opts.FootnoteAnchorPrefix, slug))
		return
	}

	r.Outs(w, "<"+listItemTag(listItem.ListFlags)+">")
}

func (r *Renderer) listItemExit(w io.Writer, listItem *ast.ListItem) {
	if listItem.RefLink != nil && r.Opts.Flags&FootnoteReturnLinks != 0 {
		slug := Slugify(listItem.RefLink)
		prefix := r.Opts.FootnoteAnchorPrefix
		link := r.Opts.FootnoteReturnLinkContents
		s := FootnoteReturnLink(prefix, link, slug)
		r.Outs(w, s)
	}

	r.Outs(w, "</"+listItemTag(listItem.ListFlags)+">")
	r.CR(w)
}

// ListItem writes ast.ListItem node
func (r *Renderer) ListItem(w io.Writer, listItem *ast.ListItem, entering bool) {
	if entering {
		r.listItemEnter(w, listItem)
	} else {
		r.listItemExit(w, listItem)
	}
}
