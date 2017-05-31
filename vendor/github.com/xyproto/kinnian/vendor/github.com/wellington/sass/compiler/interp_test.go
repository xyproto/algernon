package compiler

import (
	"testing"

	"github.com/wellington/sass/token"
)

func TestInterp(t *testing.T) {
	ctx := NewContext()

	ctx.fset = token.NewFileSet()
	input := `div {
  hello: #{123+321};
}
`
	out, err := ctx.runString("", input)
	if err != nil {
		t.Fatal(err)
	}

	e := `div {
  hello: 444; }
`
	if e != out {
		t.Fatalf("got:\n%s\nwanted:\n%s", out, e)
	}
}

func TestInterp_merge_front(t *testing.T) {
	ctx := NewContext()

	ctx.fset = token.NewFileSet()
	input := `div {
  hello: before#{123+321};
}
`
	out, err := ctx.runString("", input)
	if err != nil {
		t.Fatal(err)
	}

	e := `div {
  hello: before444; }
`
	if e != out {
		t.Fatalf("got:\n%s\nwanted:\n%s", out, e)
	}
}

func TestInterp_merge_back(t *testing.T) {
	ctx := NewContext()

	ctx.fset = token.NewFileSet()
	input := `div {
  hello: #{123+321}after;
}
`
	out, err := ctx.runString("", input)
	if err != nil {
		t.Fatal(err)
	}

	e := `div {
  hello: 444after; }
`
	if e != out {
		t.Fatalf("got:\n%s\nwanted:\n%s", out, e)
	}
}

func TestInterp_merge_both(t *testing.T) {
	ctx := NewContext()

	ctx.fset = token.NewFileSet()
	input := `div {
  hello: before#{123+321}after;
}
`
	out, err := ctx.runString("", input)
	if err != nil {
		t.Fatal(err)
	}

	e := `div {
  hello: before444after; }
`
	if e != out {
		t.Fatalf("got:\n%s\nwanted:\n%s", out, e)
	}
}

func TestInterp_copy(t *testing.T) {
	ctx := NewContext()

	ctx.fset = token.NewFileSet()
	input := `div {
  @each $i in 1 2 {
    hello: text#{$i};
  }
}
`
	out, err := ctx.runString("", input)
	if err != nil {
		t.Fatal(err)
	}

	e := `div {
  hello: text1;
  hello: text2; }
`
	if e != out {
		t.Fatalf("got:\n%s\nwanted:\n%s", out, e)
	}
}

func TestInterp_math(t *testing.T) {
	ctx := NewContext()

	ctx.fset = token.NewFileSet()
	input := `div {
  $i: 123;
  hello: #{123+321};
  there: #{$i+321};
}
`
	out, err := ctx.runString("", input)
	if err != nil {
		t.Fatal(err)
	}

	e := `div {
  hello: 444;
  there: 444; }
`
	if e != out {
		t.Fatalf("got:\n%s\nwanted:\n%s", out, e)
	}
}
