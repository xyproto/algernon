package compiler

import (
	"testing"

	"github.com/wellington/sass/token"
)

func TestSelector_nesting(t *testing.T) {
	ctx := NewContext()

	ctx.fset = token.NewFileSet()
	input := `a {
d { color: red; }
}
`
	out, err := ctx.runString("", input)
	if err != nil {
		t.Fatal(err)
	}

	e := `a d {
  color: red; }
`
	if e != out {
		t.Errorf("got:\n%s\nwanted:\n%s", out, e)
	}
}

func TestSelector_inplace_nesting(t *testing.T) {
	ctx := NewContext()

	ctx.fset = token.NewFileSet()
	input := `hey, ho {
  foo &.goo {
    color: blue;
  }
}
`
	out, err := ctx.runString("", input)
	if err != nil {
		t.Fatal(err)
	}

	e := `foo hey.goo, foo ho.goo {
  color: blue; }
`
	if e != out {
		t.Fatalf("got:\n%s\nwanted:\n%s", out, e)
	}
}

func TestSelector_deep_nesting(t *testing.T) {
	ctx := NewContext()

	ctx.fset = token.NewFileSet()
	input := `a {
	c, d, e {
	  f, g, h {
      m, n, o {
        color: blue;
      }
    }
	}
}`
	out, err := ctx.runString("", input)
	if err != nil {
		t.Fatal(err)
	}

	e := `a c f m, a c f n, a c f o, a c g m, a c g n, a c g o, a c h m, a c h n, a c h o, a d f m, a d f n, a d f o, a d g m, a d g n, a d g o, a d h m, a d h n, a d h o, a e f m, a e f n, a e f o, a e g m, a e g n, a e g o, a e h m, a e h n, a e h o {
  color: blue; }
`
	if e != out {
		t.Fatalf("got:\n%s\nwanted:\n%s", out, e)
	}
}

func TestSelector_selector_interp(t *testing.T) {
	ctx := NewContext()

	ctx.fset = token.NewFileSet()
	input := `$x: oo, ba;
$y: az, hu;

f#{$x}r {
  p: 1;
  b#{$y}x {
    q: 2;
    mumble#{length($x) + length($y)} {
      r: 3;
    }
  }
}
`
	out, err := ctx.runString("", input)
	if err != nil {
		t.Fatal(err)
	}

	e := `foo, bar {
  p: 1; }
  foo baz, foo hux, bar baz, bar hux {
    q: 2; }
    foo baz mumble4, foo hux mumble4, bar baz mumble4, bar hux mumble4 {
      r: 3; }
`

	if e != out {
		t.Fatalf("got:\n%s\nwanted:\n%s", out, e)
	}

}

func TestSelector_nesting_implicit_unary(t *testing.T) {

	ctx := NewContext()
	ctx.fset = token.NewFileSet()
	input := `a {
  > e {
    color: blue;
  }
}
`
	out, err := ctx.runString("", input)
	if err != nil {
		t.Fatal(err)
	}

	e := `a > e {
  color: blue; }
`
	if e != out {
		t.Fatalf("got:\n%s\nwanted:\n%s", out, e)
	}
}

func TestSelector_nesting_unary(t *testing.T) {

	// This is bizarre, may never support this odd syntax
	ctx := NewContext()

	ctx.fset = token.NewFileSet()
	input := `a {
  & > e {
    color: blue;
  }
}
`
	out, err := ctx.runString("", input)
	if err != nil {
		t.Fatal(err)
	}

	e := `a > e {
  color: blue; }
`
	if e != out {
		t.Fatalf("got:\n%s\nwanted:\n%s", out, e)
	}
}

func TestSelector_nesting_parent_group(t *testing.T) {

	ctx := NewContext()

	ctx.fset = token.NewFileSet()
	input := `a, b {
d { color: red; }
}
`
	out, err := ctx.runString("", input)
	if err != nil {
		t.Fatal(err)
	}

	e := `a d, b d {
  color: red; }
`
	if e != out {
		t.Fatalf("got:\n%s\nwanted:\n%s", out, e)
	}
}

func TestSelector_nesting_child_group(t *testing.T) {

	ctx := NewContext()

	ctx.fset = token.NewFileSet()
	input := `a {
b, c { color: red; }
}
`
	out, err := ctx.runString("", input)
	if err != nil {
		t.Fatal(err)
	}

	e := `a b, a c {
  color: red; }
`
	if e != out {
		t.Fatalf("got:\n%s\nwanted:\n%s", out, e)
	}
}

func TestSelector_many_nests(t *testing.T) {
	ctx := NewContext()

	ctx.fset = token.NewFileSet()
	input := `a, b {
c, d { color: red; }
}
`
	out, err := ctx.runString("", input)
	if err != nil {
		t.Fatal(err)
	}

	e := `a c, a d, b c, b d {
  color: red; }
`
	if e != out {
		t.Fatalf("got:\n%s\nwanted:\n%s", out, e)
	}
}

func TestSelector_combinators(t *testing.T) {
	ctx := NewContext()

	ctx.fset = token.NewFileSet()
	input := `a + b ~ c { color: red; }
`
	out, err := ctx.runString("", input)
	if err != nil {
		t.Fatal(err)
	}

	e := `a + b ~ c {
  color: red; }
`
	if e != out {
		t.Fatalf("got:\n%s\nwanted:\n%s", out, e)
	}

}

func TestSelector_singleampersand(t *testing.T) {
	ctx := NewContext()

	ctx.fset = token.NewFileSet()
	input := `div {
& { color: red; }
}
`
	out, err := ctx.runString("", input)
	if err != nil {
		t.Fatal(err)
	}

	e := `div {
  color: red; }
`
	if e != out {
		t.Fatalf("got:\n%s\nwanted:\n%s", out, e)
	}

}

func TestSelector_comboampersand(t *testing.T) {
	ctx := NewContext()

	ctx.fset = token.NewFileSet()
	input := `div ~ b {
& + & { color: red; }
}
`
	out, err := ctx.runString("", input)
	if err != nil {
		t.Fatal("compilation fail", err)
	}

	e := `div ~ b + div ~ b {
  color: red; }
`
	if e != out {
		t.Fatalf("got:\n%s\nwanted:\n%s", out, e)
	}

}
