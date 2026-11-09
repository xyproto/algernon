package html

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/internal/textutil"
)

// crAfterBlock ends the line after a block, except for the last block of a
// list item, which the item's closing tag follows directly.
func (r *Renderer) crAfterBlock(w io.Writer, node ast.Node) {
	if !(IsListItem(node.GetParent()) && ast.GetNextNode(node) == nil) {
		r.CR(w)
	}
}

// blockElement writes the opening tag of a block element, with the node's
// block attributes, or its closing tag, each on a line of its own.
func (r *Renderer) blockElement(w io.Writer, node ast.Node, name string, entering bool) {
	tag := TagWithAttributes("<"+name, r.blockAttrs(node))
	r.OutOneOfCr(w, entering, tag, "</"+name+">")
}

// enterOrExit renders node with enter when the walk enters it and with exit
// when it leaves it.
func enterOrExit[N any](r *Renderer, w io.Writer, node N, entering bool, enter, exit func(*Renderer, io.Writer, N)) {
	if entering {
		enter(r, w, node)
	} else {
		exit(r, w, node)
	}
}

func IsList(node ast.Node) bool { _, ok := node.(*ast.List); return ok }

func IsListTight(node ast.Node) bool { list, ok := node.(*ast.List); return ok && list.Tight }

func IsListItem(node ast.Node) bool { _, ok := node.(*ast.ListItem); return ok }

func IsListItemTerm(node ast.Node) bool {
	item, ok := node.(*ast.ListItem)
	return ok && item.ListFlags&ast.ListTypeTerm != 0
}

// Slugify creates a URL-safe fragment slug.
func Slugify(in []byte) []byte { return textutil.Slugify(in) }

// BlockAttrs returns the serialized block attributes attached to node.
//
// The renderer itself doesn't write all of them: see Renderer.blockAttrs.
func BlockAttrs(node ast.Node) []string { return blockAttrs(node, nil) }

// blockAttrs is BlockAttrs keeping only the key="value" attributes for which
// keep, if not nil, returns true.
func blockAttrs(node ast.Node, keep func(name string) bool) []string {
	var attr *ast.Attribute
	if container := node.AsContainer(); container != nil {
		attr = container.Attribute
	} else if leaf := node.AsLeaf(); leaf != nil {
		attr = leaf.Attribute
	}
	if attr == nil {
		return nil
	}

	var attrs []string
	if attr.ID != nil {
		attrs = append(attrs, fmt.Sprintf(`%s="%s"`, IDTag, escapeAttr(string(attr.ID))))
	}
	if len(attr.Classes) > 0 {
		classes := make([]string, len(attr.Classes))
		for i, class := range attr.Classes {
			classes[i] = escapeAttr(string(class))
		}
		attrs = append(attrs, fmt.Sprintf(`class="%s"`, strings.Join(classes, " ")))
	}

	keys := make([]string, 0, len(attr.Attrs))
	for key := range attr.Attrs {
		if isSafeAttrName(key) && (keep == nil || keep(key)) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		attrs = append(attrs, fmt.Sprintf(`%s="%s"`, key, escapeAttr(string(attr.Attrs[key]))))
	}
	return attrs
}

// blockAttrs returns the attributes a document gave node with the
// Attributes or InlineAttributes extension. It leaves out those the renderer
// writes itself on the element, given as own, which a document could
// otherwise use to replace the destination of a link or image after Safelink
// checked it. With SkipHTML or Safelink, which are for documents that aren't
// trusted, it only keeps attributes that can't run script or load anything.
func (r *Renderer) blockAttrs(node ast.Node, own ...string) []string {
	trusted := r.Opts.Flags&(SkipHTML|Safelink) == 0
	return blockAttrs(node, func(name string) bool {
		name = strings.ToLower(name)
		for _, o := range own {
			if name == o {
				return false
			}
		}
		return trusted || isInertAttr(name)
	})
}

// attrNames returns the names of attributes in name="value" form.
func attrNames(attrs []string) []string {
	names := make([]string, 0, len(attrs))
	for _, attr := range attrs {
		name, _, _ := strings.Cut(attr, "=")
		names = append(names, strings.ToLower(name))
	}
	return names
}

// isInertAttr returns true for the attributes an untrusted document can set:
// they can't run script, load a resource or style the page.
func isInertAttr(name string) bool {
	switch name {
	case "id", "class", "title", "lang", "dir", "role", "width", "height", "align":
		return true
	}
	return strings.HasPrefix(name, "data-") || strings.HasPrefix(name, "aria-")
}

// coalesceClassAttrs merges multiple class="..." attributes into one.
func coalesceClassAttrs(attrs []string) []string {
	const prefix = `class="`
	classes := make([]string, 0, len(attrs))
	result := make([]string, 0, len(attrs))
	classIndex := -1
	for _, attr := range attrs {
		if strings.HasPrefix(attr, prefix) && strings.HasSuffix(attr, `"`) {
			if classIndex < 0 {
				classIndex = len(result)
				result = append(result, "")
			}
			if class := attr[len(prefix) : len(attr)-1]; class != "" {
				classes = append(classes, class)
			}
			continue
		}
		result = append(result, attr)
	}
	if classIndex < 0 {
		return attrs
	}
	result[classIndex] = fmt.Sprintf(`class="%s"`, strings.Join(classes, " "))
	return result
}

// TagWithAttributes creates an HTML tag with the given name and attributes.
func TagWithAttributes(name string, attrs []string) string {
	return tagStart(name, attrs) + ">"
}

func tagStart(name string, attrs []string) string {
	if len(attrs) == 0 {
		return name
	}
	return name + " " + strings.Join(attrs, " ")
}
