package html

import (
	"fmt"
	"io"

	"github.com/gomarkdown/markdown/ast"
)

// flagTag is the tag of a list or list item whose flags include flag. A
// zero flag matches any flags.
type flagTag struct {
	flag ast.ListType
	tag  string
}

var (
	listTags     = []flagTag{{ast.ListTypeDefinition, "dl"}, {ast.ListTypeOrdered, "ol"}, {0, "ul"}}
	listItemTags = []flagTag{{ast.ListTypeTerm, "dt"}, {ast.ListTypeDefinition, "dd"}, {0, "li"}}
)

// tagForFlags returns the tag of the first entry of tags that matches flags.
func tagForFlags(flags ast.ListType, tags []flagTag) string {
	for _, t := range tags {
		if t.flag == 0 || flags&t.flag != 0 {
			return t.tag
		}
	}
	return ""
}

func listTag(flags ast.ListType) string     { return tagForFlags(flags, listTags) }
func listItemTag(flags ast.ListType) string { return tagForFlags(flags, listItemTags) }

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
	attrs = append(attrs, r.blockAttrs(nodeData, attrNames(attrs)...)...)
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
	enterOrExit(r, w, list, entering, (*Renderer).listEnter, (*Renderer).listExit)
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
	enterOrExit(r, w, listItem, entering, (*Renderer).listItemEnter, (*Renderer).listItemExit)
}
