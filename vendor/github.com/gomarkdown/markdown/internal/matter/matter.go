// Package matter names the document matters (front, main and back), so the
// parser and the renderers read and write the same tags.
package matter

import "github.com/gomarkdown/markdown/ast"

type names struct{ name, tag string }

var table = [...]names{
	ast.DocumentMatterFront: {"front", "{frontmatter}"},
	ast.DocumentMatterMain:  {"main", "{mainmatter}"},
	ast.DocumentMatterBack:  {"back", "{backmatter}"},
}

// Kinds lists the document matters that have a name.
var Kinds = [...]ast.DocumentMatters{ast.DocumentMatterFront, ast.DocumentMatterMain, ast.DocumentMatterBack}

// lookup returns the names of m, which are empty for a matter without them.
func lookup(m ast.DocumentMatters) names {
	if m < 0 || int(m) >= len(table) {
		return names{}
	}
	return table[m]
}

// Name returns "front", "main" or "back", or "" for any other value.
func Name(m ast.DocumentMatters) string { return lookup(m).name }

// Tag returns the markdown tag that starts a matter, like "{frontmatter}",
// or "" for any other value.
func Tag(m ast.DocumentMatters) string { return lookup(m).tag }
