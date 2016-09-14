package calc

import (
	"fmt"
	"log"
	"strings"

	"github.com/wellington/sass/ast"
	"github.com/wellington/sass/token"
)

// Resolve simple math to create a basic lit
func Resolve(in ast.Expr, doOp bool) (*ast.BasicLit, error) {
	return resolve(in, doOp)
}

func resolve(in ast.Expr, doOp bool) (*ast.BasicLit, error) {
	x := &ast.BasicLit{
		ValuePos: in.Pos(),
	}
	var err error
	switch v := in.(type) {
	case *ast.StringExpr:
		list := make([]string, 0, len(v.List))
		for _, l := range v.List {
			lit, err := resolve(l, doOp)
			if err != nil {
				return nil, err
			}
			list = append(list, lit.Value)
		}
		x.Kind = token.QSTRING
		x.Value = strings.Join(list, "")
	case *ast.ListLit:
		// During expr simplification, list are just string
		delim := " "
		if v.Comma {
			delim = ", "
		}
		var k token.Token
		ss := make([]string, len(v.Value))
		for i := range v.Value {
			lit, err := resolve(v.Value[i], doOp)
			k = lit.Kind
			if err != nil {
				return nil, err
			}
			ss[i] = lit.Value
		}
		x = &ast.BasicLit{
			Kind:     token.STRING,
			Value:    strings.Join(ss, delim),
			ValuePos: v.Pos(),
		}
		if len(v.Value) == 1 {
			x.Kind = k
		}
	case *ast.UnaryExpr:
		x = v.X.(*ast.BasicLit)
	case *ast.BinaryExpr:
		x, err = binary(v, doOp)
	case *ast.BasicLit:
		x = v
	case *ast.Ident:
		if v.Obj == nil {
			return nil, fmt.Errorf("calc: undefined variable %s", v.Name)
		}
		// FIXME: we shouldn't attempt to resolve invalid values ie.
		// interp inside @each
		if v.Obj.Decl == nil {
			log.Println("warning, resolution was attempted on an invalid value")
			return x, nil
		}
		rhs := v.Obj.Decl.(*ast.AssignStmt).Rhs
		kind := token.INT
		var val []string
		for _, x := range rhs {
			lit, err := resolve(x, doOp)
			if err != nil {
				return nil, err
			}
			// TODO: insufficient!
			if lit.Kind != kind {
				kind = lit.Kind
			}
			val = append(val, lit.Value)
		}
		// TODO: commas are missing
		x.Value = strings.Join(val, ", ")
		x.Kind = kind
	case *ast.CallExpr:
		x, err = resolve(v.Resolved, doOp)
	case *ast.Interp:
		if v.Obj == nil {
			panic("unresolved interpolation")
		}
		x, err = resolve(v.Obj.Decl.(ast.Expr), doOp)
	default:
		err = fmt.Errorf("unsupported calc.resolve % #v\n", v)
		panic(err)
	}
	return x, err
}

// binary takes a BinaryExpr and simplifies it to a basiclit
func binary(in *ast.BinaryExpr, doOp bool) (*ast.BasicLit, error) {
	var hasList bool
	// fuq, look for paren wrapped lists
	if lit, ok := in.X.(*ast.ListLit); ok {
		if lit.Paren == true && len(lit.Value) == 1 {
			doOp = true
		} else {
			hasList = true
		}
	}

	if lit, ok := in.Y.(*ast.ListLit); ok {
		if lit.Paren == true && len(lit.Value) == 1 {
			doOp = true
		} else {
			hasList = true
		}
	}

	// Division may ignore math, but nothing else does
	if in.Op != token.QUO && !hasList {
		doOp = true
	}

	if _, ok := in.X.(*ast.Ident); ok {
		doOp = true
	}
	if _, ok := in.Y.(*ast.Ident); ok {
		doOp = true
	}

	left, err := resolve(in.X, doOp)
	if err != nil {
		return nil, err
	}
	right, err := resolve(in.Y, doOp)
	if err != nil {
		return nil, err
	}
	out := &ast.BasicLit{
		ValuePos: left.Pos(),
		Kind:     token.STRING,
	}
	if doOp {
		// So actually, we could be a valid type
		out.Kind = left.Kind
	}
	switch in.Op {
	case token.ADD, token.SUB, token.MUL, token.QUO:
		return combineLits(in.Op, left, right, doOp)
	case token.EQL:
		out.Value = "false"
		if left.Value == right.Value {
			out.Value = "true"
		} else {
			log.Printf("not equal % #v: % #v\n", left, right)
		}
	default:
		fmt.Printf("l: % #v\nr: % #v\n", left, right)
		err = fmt.Errorf("unsupported Operation %s", in.Op)
	}
	return out, err
}

func combineLits(op token.Token, left, right *ast.BasicLit, force bool) (*ast.BasicLit, error) {
	return ast.Op(op, left, right, force)

}

// matchTypes looks for a compatiable token for all passed lit
// The default is string, but if INT or FLOAT suffices those are
// used
func matchTypes(lits ...*ast.BasicLit) token.Token {
	if len(lits) == 0 {
		return token.ILLEGAL
	}

	tok := lits[0].Kind
	// colors are special
	if tok == token.COLOR {
		return tok
	}
	for i := range lits[1:] {
		if lits[i].Kind != tok {
			return token.STRING
		}
	}
	return tok
}
