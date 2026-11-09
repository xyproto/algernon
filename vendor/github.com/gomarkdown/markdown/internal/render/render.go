// Package render walks an AST for the renderers.
package render

import (
	"io"

	"github.com/gomarkdown/markdown/ast"
)

// Tree writes node and its descendants to w with renderNode, which is
// called on entering and on leaving each node.
func Tree(w io.Writer, node ast.Node, renderNode func(w io.Writer, node ast.Node, entering bool) ast.WalkStatus) {
	ast.WalkFunc(node, func(n ast.Node, entering bool) ast.WalkStatus {
		return renderNode(w, n, entering)
	})
}
