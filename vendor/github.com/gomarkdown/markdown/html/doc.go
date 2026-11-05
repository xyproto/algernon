/*
Package html implements an HTML renderer of a parsed markdown document.

# Configuring and customizing a renderer

A renderer can be configured with multiple options:

	import "github.com/gomarkdown/markdown/html"

	flags := html.CommonFlags | html.CompletePage | html.HrefTargetBlank
	opts := html.RendererOptions{
		Title: "A custom title",
		Flags: flags,
	}
	renderer := html.NewRenderer(opts)

You can also re-use most of the logic and customize rendering of selected nodes
by providing a node render hook.
This is most useful for rendering nodes that allow for design choices, like
links or code blocks.

	import (
		"github.com/gomarkdown/markdown/html"
		"github.com/gomarkdown/markdown/ast"
	)

	// a simple render hook that will output "code_replacement" instead of
	// <code>${content}</code> emitted by html.Renderer
	func renderHookCodeBlock(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
		_, ok := node.(*ast.CodeBlock)
		if !ok {
			return ast.GoToNext, false
		}
		io.WriteString(w, "code_replacement")
		return ast.GoToNext, true
	}

	opts := html.RendererOptions{
		RenderNodeHook: renderHookCodeBlock,
	}
	renderer := html.NewRenderer(opts)

# Untrusted input

For documents you don't trust, set SkipHTML and Safelink:

	flags := html.CommonFlags | html.SkipHTML | html.Safelink

SkipHTML removes raw HTML and Safelink only allows http, https, ftp and
mailto URLs and relative paths in links and images. With either of them,
attributes that a document sets with the parser.Attributes or
parser.InlineAttributes extension are limited to id, class, title, lang, dir,
role, width, height, align, data-* and aria-*, which can't run script or load
anything. A document never replaces an attribute the renderer writes, like
the href of a link.

These flags don't make the output safe to show as it is. Use an HTML
sanitizer such as github.com/microcosm-cc/bluemonday as well. Render hooks
that write attributes with BlockAttrs get all of them, unfiltered.
*/
package html
