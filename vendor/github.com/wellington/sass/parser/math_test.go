package parser

import (
	"testing"

	"github.com/wellington/sass/ast"
	"github.com/wellington/sass/token"
)

func TestBinary_math(t *testing.T) {
	const src = `
$a: inspect(1 + 3);
$b: inspect(3/1);
$c: inspect(1/2 + 1/2);
$d: inspect(2*2);
$d: inspect(1*1/2);
$e: inspect(1/2*1/2);
$f: inspect(2*2/2*2);

//$o: inspect(3px + 3px + 3px);
`
	f, err := ParseFile(token.NewFileSet(), "", src, 0|Trace)
	if err != nil {
		t.Fatal(err)
	}

	lits := []string{
		"4",
		"3",
		"1",
		"4",
		"0.5",
		"0.25",
		"4",
	}
	var pos int
	ast.Inspect(f, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			// ident := call.Fun.(*ast.Ident)
			lit := call.Resolved.(*ast.BasicLit)
			val := lits[pos]
			pos++

			if val != lit.Value {
				t.Errorf("useful ident here got: %s wanted %s", lit.Value, val)
			}
		}
		return true
	})

}
