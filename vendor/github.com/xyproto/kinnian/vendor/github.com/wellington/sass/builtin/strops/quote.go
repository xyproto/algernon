package strops

import (
	"strconv"

	"github.com/wellington/sass/ast"
	"github.com/wellington/sass/builtin"
	"github.com/wellington/sass/strops"
	"github.com/wellington/sass/token"
)

func init() {
	builtin.Register("unquote($string)", unquote)
	builtin.Reg("length($value)", length)
}

func unquote(call *ast.CallExpr, args ...*ast.BasicLit) (*ast.BasicLit, error) {
	in := *args[0]
	lit := &ast.BasicLit{
		Kind:     token.STRING,
		ValuePos: in.ValuePos,
		Value:    strops.Unquote(in.Value),
	}
	// Because in Ruby Sass, there is no failure though libSass fails
	// very easily
	return lit, nil
}

func length(call *ast.CallExpr, args ...ast.Expr) (ast.Expr, error) {

	lit := &ast.BasicLit{
		Kind:     token.INT,
		Value:    "1",
		ValuePos: args[0].Pos(),
	}

	switch v := args[0].(type) {
	case *ast.ListLit:
		lit.Value = strconv.Itoa(len(v.Value))
	}

	return lit, nil
}
