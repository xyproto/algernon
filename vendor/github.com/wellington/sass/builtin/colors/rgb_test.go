package colors

import (
	"image/color"
	"testing"

	"github.com/wellington/sass/ast"
	"github.com/wellington/sass/token"
)

func runParseColors(t *testing.T, in []*ast.BasicLit, e color.RGBA) {
	c, err := parseColors(in)
	if err != nil {
		t.Fatal(err)
	}
	if c != e {
		t.Errorf("got: %v wanted: %v", c, e)
	}
}

func TestParseColors_rgb(t *testing.T) {
	in := []*ast.BasicLit{
		{0, token.INT, "255"},
		{0, token.INT, "255"},
		{0, token.INT, "0"},
	}
	runParseColors(t, in, color.RGBA{
		R: 255,
		G: 255,
	})

	in = []*ast.BasicLit{
		{0, token.COLOR, "cyan"},
		{0, token.INT, "0"},
		{30, token.INT, "0"},
		{222, token.FLOAT, "0.7"},
	}
	runParseColors(t, in, color.RGBA{
		R: 0,
		G: 255,
		B: 255,
		A: 70,
	})
	in = []*ast.BasicLit{
		{0, token.COLOR, "#7b2d06"},
	}
	runParseColors(t, in, color.RGBA{
		R: 123,
		G: 45,
		B: 6,
		A: 100,
	})

	in = []*ast.BasicLit{
		{149, token.COLOR, "#f0e"},
		{0, token.INT, "0"},
		{0, token.INT, "0"},
		{163, token.FLOAT, ".5"},
	}
	runParseColors(t, in, color.RGBA{
		R: 255,
		G: 0,
		B: 238,
		A: 50,
	})
}

func runOneColor(t *testing.T, which string, in []*ast.BasicLit, e ast.BasicLit) {
	lit, err := onecolor(which, in)
	if err != nil {
		t.Fatal(err)
	}
	if e != *lit {
		t.Errorf("got: %v wanted: %v", lit, e)
	}
}

func TestOneColor(t *testing.T) {
	in := []*ast.BasicLit{
		{0, token.INT, "255"},
		{0, token.INT, "255"},
		{0, token.INT, "0"},
	}
	runOneColor(t, "red", in, ast.BasicLit{0, token.INT, "255"})
	runOneColor(t, "green", in, ast.BasicLit{0, token.INT, "255"})
	runOneColor(t, "blue", in, ast.BasicLit{0, token.INT, "0"})

	in = []*ast.BasicLit{
		{0, token.COLOR, "cyan"},
		{0, token.INT, "0"},
		{30, token.INT, "0"},
		{222, token.FLOAT, "0.7"},
	}
	runOneColor(t, "red", in, ast.BasicLit{0, token.INT, "0"})
	runOneColor(t, "green", in, ast.BasicLit{0, token.INT, "255"})
	runOneColor(t, "blue", in, ast.BasicLit{0, token.INT, "255"})

	in = []*ast.BasicLit{
		{0, token.COLOR, "#7b2d06"},
	}
	runOneColor(t, "red", in, ast.BasicLit{0, token.INT, "123"})
	runOneColor(t, "green", in, ast.BasicLit{0, token.INT, "45"})
	runOneColor(t, "blue", in, ast.BasicLit{0, token.INT, "6"})

	in = []*ast.BasicLit{
		{149, token.COLOR, "#f0e"},
		{0, token.INT, "0"},
		{0, token.INT, "0"},
		{163, token.FLOAT, ".5"},
	}
	runOneColor(t, "red", in, ast.BasicLit{0, token.INT, "255"})
	runOneColor(t, "green", in, ast.BasicLit{0, token.INT, "0"})
	runOneColor(t, "blue", in, ast.BasicLit{0, token.INT, "238"})

}
