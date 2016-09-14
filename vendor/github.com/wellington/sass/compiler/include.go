package compiler

import (
	"fmt"
	"log"

	"github.com/wellington/sass/ast"
)

func printInclude(ctx *Context, n ast.Node) {
	panic("dont call this")
	spec := n.(*ast.IncludeSpec)

	name := spec.Name.String()
	var params []*ast.Field
	if spec.Params != nil {
		params = spec.Params.List
	}
	numargs := spec.Params.NumFields()

	mix, err := ctx.scope.Mixin(name, numargs)
	if err != nil {
		log.Fatal(err)
	}

	// Add new scope, register args
	ctx.scope = NewScope(ctx.scope)

	mixargs := mix.fn.Type.Params.List
	for i := range mixargs {
		// Param passed by include
		var param *ast.Field
		if len(params) > i {
			param = params[i]
		}
		var (
			key *ast.BasicLit
			val *ast.Ident
		)
		_, _ = key, val
		switch v := mixargs[i].Type.(type) {
		case *ast.BasicLit:
			key = v
		case *ast.KeyValueExpr:
			key = v.Key.(*ast.BasicLit)
			val = ast.ToIdent(v.Value)
		}

		if param != nil {
			switch v := param.Type.(type) {
			case *ast.KeyValueExpr:
				// Key args specify their argument, so use their key
				// instead of the mixins argument for this position
				// Params with defaults
				key = v.Key.(*ast.BasicLit)
				val = ast.ToIdent(v.Value)
			case *ast.Ident:
				val = v
			case *ast.BasicLit:
				val = ast.ToIdent(v)
			default:
				fmt.Printf("dropped param: % #v\n", v)
			}
		}
		// if key != nil && val != nil {
		// 	ctx.scope.Insert(key.Value, val.Name)
		// }
	}
	if len(params) > len(mixargs) {
		fmt.Printf("dropped extra params: % #v\n", params[len(mixargs):])
	}

	for _, stmt := range mix.fn.Body.List {
		ast.Walk(ctx, stmt)
	}

	// Exit new scope, removing args
	ctx.scope = CloseScope(ctx.scope)
}
