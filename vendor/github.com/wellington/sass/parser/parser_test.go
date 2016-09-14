package parser

import (
	"fmt"
	"testing"

	"github.com/wellington/sass/ast"
	"github.com/wellington/sass/token"
)

func testString(t *testing.T, in string, mode Mode) (*ast.File, *token.FileSet) {
	fset := token.NewFileSet()
	f, err := ParseFile(fset, "testfile", in, mode)
	if err != nil {
		t.Fatal(err)
	}
	return f, fset

}

func TestObjects(t *testing.T) {
	const src = `
$color: red;
$list: 1 2 $color;
`
	f, err := ParseFile(token.NewFileSet(), "", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	objects := map[string]ast.ObjKind{
		"$color": ast.Var,
		"$list":  ast.Var,
	}

	ast.Inspect(f, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok {

			obj := ident.Obj
			if obj == nil {
				if objects[ident.Name] != ast.Bad {
					t.Errorf("no object for %s", ident.Name)
				}
				return true
			}
			if obj.Name != ident.Name {
				t.Errorf("names don't match: obj.Name = %s, ident.Name = %s", obj.Name, ident.Name)
			}
			kind := objects[ident.Name]
			if obj.Kind != kind {
				t.Errorf("%s: obj.Kind = %s; want %s", ident.Name, obj.Kind, kind)
			}
		}
		return true
	})
}

func TestUnitMath(t *testing.T) {
	const src = `
$a: 3px + 3px;
div {
  width: $a;
}
`
	f, err := ParseFile(token.NewFileSet(), "", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	objects := map[string]ast.ObjKind{
		"$color": ast.Var,
		"$list":  ast.Var,
	}
	ast.Inspect(f, func(n ast.Node) bool {
		if spec, ok := n.(*ast.RuleSpec); ok {
			ast.Print(token.NewFileSet(), spec)
		}
		return true
		if ident, ok := n.(*ast.Ident); ok {
			return true
			obj := ident.Obj
			if obj == nil {
				if objects[ident.Name] != ast.Bad {
					t.Errorf("no object for %s", ident.Name)
				}
				return true
			}
			if obj.Name != ident.Name {
				t.Errorf("names don't match: obj.Name = %s, ident.Name = %s", obj.Name, ident.Name)
			}
			kind := objects[ident.Name]
			if obj.Kind != kind {
				t.Errorf("%s: obj.Kind = %s; want %s", ident.Name, obj.Kind, kind)
			}
		}
		return true
	})
}

func TestSelMath(t *testing.T) {
	// Selectors act like boolean math
	in := `
div ~ span { }`
	f, fset := testString(t, in, 0)
	_ = fset
	sel, ok := f.Decls[0].(*ast.SelDecl)
	if !ok {
		t.Fatal("SelDecl expected")
	}

	bexpr, ok := sel.Sel.(*ast.BinaryExpr)
	if !ok {
		t.Fatal("BinaryExpr expected")
	}

	lit, ok := bexpr.X.(*ast.BasicLit)
	if !ok {
		t.Fatal("BasicLit expected")
	}

	if e := "div"; lit.Value != e {
		t.Errorf("got: %s wanted: %s", lit.Value, e)
	}

	if e := token.TIL; bexpr.Op != e {
		t.Errorf("got: %s wanted: %s", bexpr.Op, e)
	}

	lit, ok = bexpr.Y.(*ast.BasicLit)
	if !ok {
		t.Fatal("BasicLit expected")
	}

	if e := "span"; lit.Value != e {
		t.Errorf("got: %s wanted: %s", lit.Value, e)
	}
}

func TestBackRef(t *testing.T) {
	// Selectors act like boolean math
	in := `div { & { color: red; } }`
	f, fset := testString(t, in, 0)
	_, _ = f, fset

	decl, ok := f.Decls[0].(*ast.SelDecl)
	if !ok {
		t.Fatal("SelDecl expected")
	}
	sel := decl.SelStmt
	lit, ok := sel.Sel.(*ast.BasicLit)
	if !ok {
		t.Fatal("BasicLit expected")
	}
	if e := "div"; lit.Value != e {
		t.Errorf("got: %s wanted: %s", lit.Value, e)
	}

	nested, ok := sel.Body.List[0].(*ast.SelStmt)
	if !ok {
		t.Fatal("expected SelStmt")
	}

	if e := "&"; nested.Name.String() != e {
		t.Fatalf("got: %s wanted: %s", nested.Name.String(), e)
	}

	if e := "div"; e != nested.Resolved.Value {
		t.Errorf("got: %s wanted: %s", nested.Resolved.Value, e)
	}
}

var imports = map[string]bool{
	`"../sass-spec/spec/basic/01_simple_css/input.scss"`: true,
}

func TestImports(t *testing.T) {
	for path, isValid := range imports {
		src := fmt.Sprintf("@import %s;", path)
		_, err := ParseFile(token.NewFileSet(), "", src, 0) // Trace
		switch {
		case err != nil && isValid:
			t.Errorf("ParseFile(%s): got %v; expected no error", src, err)
		case err == nil && !isValid:
			t.Errorf("ParseFile(%s): got no error; expected one", src)
		}
	}
}
