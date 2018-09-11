package builtin

import "github.com/wellington/sass/ast"

// CallFunc describes a Sass function
type CallFunc func(expr *ast.CallExpr, args ...*ast.BasicLit) (*ast.BasicLit, error)

var reg func(s string, ch CallFunc, c CallHandle)

var chs = map[string]CallFunc{}

// BindRegister allows the binding of CallFunc and deprecated
// callhandle
func BindRegister(old func(s string, ch CallFunc, c CallHandle)) {
	reg = old
	for k, v := range chs {
		reg(k, v, nil)
		delete(chs, k)
	}
	for k, v := range cs {
		reg(k, nil, v)
		delete(cs, k)
	}
}

func Register(s string, ch CallFunc) {
	if reg != nil {
		reg(s, ch, nil)
		return
	}
	chs[s] = ch
}

var cs = map[string]CallHandle{}

// CallHandle pass in Expr get out Expr. This replaces
// the limited CallHandler which can't work on map or lists
type CallHandle func(expr *ast.CallExpr, args ...ast.Expr) (ast.Expr, error)

// Reg registers a CallHandle for use by parser
func Reg(s string, ch CallHandle) {
	if reg != nil {
		reg(s, nil, ch)
		return
	}
	cs[s] = ch
}
