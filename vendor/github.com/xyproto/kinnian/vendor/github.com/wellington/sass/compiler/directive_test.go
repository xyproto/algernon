package compiler

import (
	"testing"

	"github.com/wellington/sass/token"
)

func TestDirective_each_paran(t *testing.T) {
	ctx := NewContext()

	ctx.fset = token.NewFileSet()
	input := `div {
  @each $i in (1 2 3 4 5) {
   i: $i;
  }
}
`
	out, err := ctx.runString("", input)
	if err != nil {
		t.Fatal(err)
	}

	e := `div {
  i: 1;
  i: 2;
  i: 3;
  i: 4;
  i: 5; }
`
	if e != out {
		t.Fatalf("got:\n%s\nwanted:\n%s", out, e)
	}
}

func TestDirective_each(t *testing.T) {
	ctx := NewContext()

	ctx.fset = token.NewFileSet()
	input := `div {
  @each $i in a b c {
   i: $i;
  }
}
`
	out, err := ctx.runString("", input)
	if err != nil {
		t.Fatal(err)
	}

	e := `div {
  i: a;
  i: b;
  i: c; }
`
	if e != out {
		t.Fatalf("got:\n%s\nwanted:\n%s", out, e)
	}
}
