package parser

import "github.com/wellington/sass/ast"

// mergeExprs looks for interpolation and performs literal merges
// The return is just a string, so YMMV
func itpMerge(in []ast.Expr) (string, bool) {
	var found bool
	var comb string
	for i := range in {
		if _, ok := in[i].(*ast.Interp); ok {
			found = true
		}
		if i+1 >= len(in) {
			continue
		}
		comb += itpExpand(in[i], in[i+1])
	}
	comb += itpExpand(in[len(in)-1], nil)
	return comb, found
}

func itpExpand(left, right ast.Expr) string {
	var s string
	switch v := left.(type) {
	case *ast.Interp:
		s += v.Obj.Decl.(*ast.BasicLit).Value
	case *ast.BasicLit:
		s += v.Value
	}
	if right != nil {
		if left.End() < right.Pos() {
			s += " "
		}
	}
	return s
}
