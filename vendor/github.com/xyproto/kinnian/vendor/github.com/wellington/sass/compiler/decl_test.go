package compiler

import (
	"testing"

	"github.com/wellington/sass/parser"
	"github.com/wellington/sass/token"
)

func TestDecl_if(t *testing.T) {
	ctx := NewContext()

	ctx.fset = token.NewFileSet()

	input := `$x: 1 2;
@if type-of(nth($x, 2)) == number {
  div {
    background: gray;
  }
}
@else if type-of(nth($x, 2)) == string {
  div {
    background: blue;
  }
}
@else {
  div {
    background: green;
  }
}
`
	ctx.SetMode(parser.Trace)
	out, err := ctx.runString("", input)
	if err != nil {
		t.Fatal(err)
	}

	e := `div {
  background: gray; }
`
	if e != out {
		t.Fatalf("got:\n%s\nwanted:\n%s", out, e)
	}
}

func TestDecl_func_if(t *testing.T) {
	ctx := NewContext()

	ctx.fset = token.NewFileSet()
	input := `$x: true;

@function foobar() {
  @if $x {
    $x: false !global;
    @return foo;
  }
}

div {
  content: foobar();
}
`
	ctx.SetMode(parser.Trace)
	out, err := ctx.runString("", input)
	if err != nil {
		t.Fatal(err)
	}

	e := `div {
  content: foo; }
`
	if e != out {
		t.Fatalf("got:\n%s\nwanted:\n%s", out, e)
	}
}
