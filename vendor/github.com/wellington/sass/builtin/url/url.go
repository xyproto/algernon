package url

import (
	"github.com/wellington/sass/ast"
	"github.com/wellington/sass/token"

	"github.com/wellington/sass/builtin"
)

func init() {
	builtin.Register("url($value)", url)
}

func url(call *ast.CallExpr, args ...*ast.BasicLit) (*ast.BasicLit, error) {
	val := args[0].Value
	if args[0].Kind == token.QSTRING {
		val = `"` + val + `"`
	}
	val = "url(" + val + ")"
	lit := &ast.BasicLit{
		Kind:     token.STRING,
		Value:    val,
		ValuePos: call.Pos(),
	}
	return lit, nil
}
