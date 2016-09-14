package list

import (
	"fmt"
	"strconv"

	"github.com/wellington/sass/ast"
	"github.com/wellington/sass/builtin"
)

func init() {
	builtin.Reg("nth($list, $pos)", nth)
}

func nth(call *ast.CallExpr, args ...ast.Expr) (ast.Expr, error) {
	in := args[0]
	x := args[1]
	s := x.(*ast.BasicLit)
	pos, err := strconv.Atoi(s.Value)
	if err != nil {
		return nil, err
	}

	list, ok := in.(*ast.ListLit)
	if !ok {
		list = &ast.ListLit{
			Value: []ast.Expr{in},
		}
	}

	if pos > len(list.Value) {
		return nil, fmt.Errorf("index out of bounds for `nth($list, $n)` at %d", in.Pos())
	}

	return list.Value[pos-1], nil
}
