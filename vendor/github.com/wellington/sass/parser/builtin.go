package parser

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/wellington/sass/ast"
	"github.com/wellington/sass/builtin"
	"github.com/wellington/sass/calc"
	"github.com/wellington/sass/token"

	// Include defined builtins
	_ "github.com/wellington/sass/builtin/colors"
	_ "github.com/wellington/sass/builtin/introspect"
	_ "github.com/wellington/sass/builtin/list"
	_ "github.com/wellington/sass/builtin/strops"
	_ "github.com/wellington/sass/builtin/url"
)

var ErrNotFound = errors.New("function does not exist")

type call struct {
	name   string
	params []*ast.KeyValueExpr
	ch     builtin.CallFunc
	handle builtin.CallHandle
}

func (c *call) Pos(key *ast.Ident) int {
	for i, arg := range c.params {
		switch v := arg.Key.(type) {
		case *ast.Ident:
			if key.Name == v.Name {
				return i
			}
		default:
			log.Fatalf("failed to lookup key % #v\n", v)
		}
	}
	return -1
}

type desc struct {
	err error
	c   call
}

func (d *desc) Visit(node ast.Node) ast.Visitor {
	switch v := node.(type) {
	case *ast.RuleSpec:
		for i := range v.Values {
			ast.Walk(d, v.Values[i])
		}
	case *ast.GenDecl:
		for _, spec := range v.Specs {
			ast.Walk(d, spec)
		}
		return nil
	case *ast.CallExpr:
		d.c.name = v.Fun.(*ast.Ident).Name
		for _, arg := range v.Args {
			switch v := arg.(type) {
			case *ast.KeyValueExpr:
				d.c.params = append(d.c.params, v)
			case *ast.Ident:
				d.c.params = append(d.c.params, &ast.KeyValueExpr{
					Key: v,
				})
			default:
				panic(fmt.Errorf("%s failed to parse arg % #v\n",
					d.c.name, v))
			}
		}
		return nil
	case nil:
		return nil
	default:
		panic(fmt.Errorf("illegal walk % #v\n", v))
	}
	return d
}

var builtins = make(map[string]call)

func init() {
	builtin.BindRegister(register)
}

func register(s string, ch builtin.CallFunc, h builtin.CallHandle) {
	fset := token.NewFileSet()
	pf, err := ParseFile(fset, "", s, FuncOnly)
	if err != nil {
		if !strings.HasSuffix(err.Error(), "expected ';', found 'EOF'") {
			log.Fatal(err)
		}
	}
	d := &desc{c: call{
		ch:     ch,
		handle: h,
	}}
	ast.Walk(d, pf.Decls[0])
	if d.err != nil {
		log.Fatal("failed to parse func description", d.err)
	}
	if _, ok := builtins[d.c.name]; ok {
		log.Println("already registered", d.c.name)
	}
	builtins[d.c.name] = d.c
}

// This might not be enough
func evaluateCall(p *parser, scope *ast.Scope, expr *ast.CallExpr) (ast.Expr, error) {
	ident := expr.Fun.(*ast.Ident)
	name := ident.Name

	// First check builtins
	if fn, ok := builtins[name]; ok {
		return callBuiltin(name, fn, expr)
	}
	return p.callInline(scope, expr)
}

// callInline looks for the function within Sass itself
func (p *parser) callInline(scope *ast.Scope, call *ast.CallExpr) (ast.Expr, error) {

	return p.resolveFuncDecl(scope, call)
}

func callBuiltin(name string, fn call, expr *ast.CallExpr) (ast.Expr, error) {

	// Walk through the function
	// These should be processed at registration time
	callargs := make([]ast.Expr, len(fn.params))
	for i := range fn.params {
		expr := fn.params[i].Value
		// if expr != nil {
		// 	callargs[i] = expr.(*ast.BasicLit)
		// }
		callargs[i] = expr
	}
	var argpos int
	incoming := expr.Args

	// Verify args and convert to BasicLit before passing along
	if len(callargs) < len(incoming) {
		for i, p := range incoming {
			lit, ok := p.(*ast.BasicLit)
			if !ok {
				log.Fatalf("failed to convert to lit % #v\n", p)
			}
			log.Printf("inc %d %s:% #v\n", i, lit.Kind, p)
		}
		return nil, fmt.Errorf("mismatched arg count %s got: %d wanted: %d",
			name, len(incoming), len(callargs))
	}

	for i, arg := range incoming {
		if argpos < i {
			argpos = i
		}
		switch v := arg.(type) {
		case *ast.KeyValueExpr:
			pos := fn.Pos(v.Key.(*ast.Ident))
			callargs[pos] = v.Value.(*ast.BasicLit)
		case *ast.ListLit:
			callargs[argpos] = v
		case *ast.Ident:
			if v.Obj != nil {
				ass := v.Obj.Decl.(*ast.AssignStmt)
				callargs[argpos] = ass.Rhs[0]
			} else {
				callargs[argpos] = v
			}

		default:
			lit, err := calc.Resolve(v, true)
			if err == nil {
				callargs[argpos] = lit
			} else {
				return nil, err
			}
		}
	}
	if fn.ch != nil {
		lits := make([]*ast.BasicLit, len(callargs))
		var err error
		for i, x := range callargs {
			lits[i], err = calc.Resolve(x, true)
			// lits[i], ok = exprToLit(x)
			if err != nil {
				return nil, fmt.Errorf("failed to parse arg(%d) in %s: %s", i, fn.name, err)
			}
		}
		return fn.ch(expr, lits...)
	}
	return fn.handle(expr, callargs...)
}
