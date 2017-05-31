package calc

import (
	"testing"

	"github.com/wellington/sass/ast"
	"github.com/wellington/sass/token"
)

func TestBinary_simple_int(t *testing.T) {
	bin := &ast.BinaryExpr{
		X:  &ast.BasicLit{Kind: token.INT, Value: "1"},
		Op: token.ADD,
		Y:  &ast.BasicLit{Kind: token.INT, Value: "2"},
	}
	lit, err := binary(bin, true)
	if err != nil {
		t.Fatal(err)
	}

	if lit.Kind != token.INT {
		t.Fatalf("unexpected kind: %s", lit.Kind)
	}

	if e := "3"; lit.Value != e {
		t.Errorf("got: %s wanted: %s", lit.Value, e)
	}

}

func TestBinary_simple_str(t *testing.T) {
	bin := &ast.BinaryExpr{
		X:  &ast.BasicLit{Kind: token.STRING, Value: "a"},
		Op: token.ADD,
		Y:  &ast.BasicLit{Kind: token.STRING, Value: "b"},
	}
	lit, err := binary(bin, true)
	if err != nil {
		t.Fatal(err)
	}

	if lit.Kind != token.STRING {
		t.Fatal("unexpected kind")
	}

	if e := "ab"; lit.Value != e {
		t.Errorf("got: %s wanted: %s", lit.Value, e)
	}

	bin = &ast.BinaryExpr{
		X:  &ast.BasicLit{Kind: token.STRING, Value: "a"},
		Op: token.ADD,
		Y:  &ast.BasicLit{Kind: token.INT, Value: "1"},
	}
	lit, err = binary(bin, true)
	if err != nil {
		t.Fatal(err)
	}

	if lit.Kind != token.STRING {
		t.Fatal("unexpected kind")
	}

	if e := "a1"; lit.Value != e {
		t.Errorf("got: %s wanted: %s", lit.Value, e)
	}

}
