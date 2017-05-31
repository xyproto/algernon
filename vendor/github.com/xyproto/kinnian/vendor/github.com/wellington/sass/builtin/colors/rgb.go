package colors

import (
	"fmt"
	"image/color"
	"log"
	"strconv"

	"github.com/wellington/sass/ast"
	"github.com/wellington/sass/builtin"
	"github.com/wellington/sass/token"
)

func init() {
	builtin.Register("rgb($red:0, $green:0, $blue:0)", rgb)
	builtin.Register("rgba($red:0, $green:0, $blue:0, $alpha:0)", rgba)
	builtin.Register("mix($color1, $color2, $weight:0.5)", mix)
	builtin.Register("invert($color)", invert)
	builtin.Register("red($color)", red)
	builtin.Register("blue($color)", blue)
	builtin.Register("green($color)", green)
}

func resolveDecl(ident *ast.Ident) []*ast.BasicLit {
	var lits []*ast.BasicLit
	switch decl := ident.Obj.Decl.(type) {
	case *ast.AssignStmt:
		call := decl.Rhs[0].(*ast.CallExpr)
		for i := range call.Args {
			lits = append(lits, call.Args[i].(*ast.BasicLit))
		}
	case *ast.BasicLit:
		lits = append(lits, decl)
	default:
		log.Fatalf("can not resolve: % #v\n", decl)
	}
	return lits
}

func parseColors(args []*ast.BasicLit) (color.RGBA, error) {
	ints := make([]uint8, 4)
	var ret color.RGBA
	var u uint8
	for i := range args {
		v := args[i]
		switch v.Kind {
		case token.FLOAT:
			f, err := strconv.ParseFloat(args[i].Value, 8)
			if err != nil {
				return ret, err
			}
			// Has to be alpha, or bust
			u = uint8(f * 100)
		case token.INT:
			i, err := strconv.Atoi(v.Value)
			if err != nil {
				return ret, err
			}
			u = uint8(i)
		case token.COLOR:
			if i != 0 {
				return ret, fmt.Errorf("hex is only allowed as the first argumetn found: % #v", v)
			}
			var err error
			ret, err = ast.ColorFromHexString(v.Value)
			if err != nil {
				return ret, err
			}
			// This is only allowed as the first argument
			i = i + 2
		default:
			log.Fatalf("unsupported kind %s % #v\n", v.Kind, v)
		}
		ints[i] = u
	}
	if ints[0] > 0 {
		ret.R = ints[0]
	}
	if ints[1] > 0 {
		ret.G = ints[1]
	}
	if ints[2] > 0 {
		ret.B = ints[2]
	}
	if ints[3] > 0 {
		ret.A = ints[3]
	}
	return ret, nil
}

func onecolor(which string, args []*ast.BasicLit) (*ast.BasicLit, error) {
	c, err := parseColors(args)
	if err != nil {
		return nil, err
	}
	lit := &ast.BasicLit{
		Kind: token.INT,
	}
	switch which {
	case "red":
		lit.Value = strconv.Itoa(int(c.R))
	case "green":
		lit.Value = strconv.Itoa(int(c.G))
	case "blue":
		lit.Value = strconv.Itoa(int(c.B))
	default:
		panic("not a onecolor")
	}
	return lit, nil
}

func red(call *ast.CallExpr, args ...*ast.BasicLit) (*ast.BasicLit, error) {
	return onecolor("red", args)
}

func green(call *ast.CallExpr, args ...*ast.BasicLit) (*ast.BasicLit, error) {
	return onecolor("green", args)
}

func blue(call *ast.CallExpr, args ...*ast.BasicLit) (*ast.BasicLit, error) {
	return onecolor("blue", args)
}

func rgb(call *ast.CallExpr, args ...*ast.BasicLit) (*ast.BasicLit, error) {
	// log.Println("rgb call:", call.Args)
	// log.Printf("rgb args: red: %s green: %s blue: %s\n",
	// 	args[0].Value, args[1].Value, args[2].Value)
	c, err := parseColors(args)
	if err != nil {
		return nil, err
	}

	return colorOutput(c, &ast.BasicLit{}), nil
}

func rgba(call *ast.CallExpr, args ...*ast.BasicLit) (*ast.BasicLit, error) {
	// This is ugly. Instead there needs to be a 2 arg implementation of rgba()
	if len(call.Args) == 2 && args[3].Value == "0" {
		args[3] = args[1]
		args[1] = &ast.BasicLit{Kind: token.INT, Value: "0"}
	}
	// log.Printf("rgba args: red: %s green: %s blue: %s alpha: %s\n",
	// 	args[0].Value, args[1].Value, args[2].Value, args[3].Value)

	c, err := parseColors(args)
	if err != nil {
		return nil, err
	}
	return colorOutput(c, call), nil
}

// mix takes two colors and optional weight (50% assumed). mix evaluates the
// difference of alphas and factors this into the weight calculations
// For details see: http://sass-lang.com/documentation/Sass/Script/Functions.html#mix-instance_method
func mix(call *ast.CallExpr, args ...*ast.BasicLit) (*ast.BasicLit, error) {
	wt, err := strconv.ParseFloat(args[2].Value, 8)
	// Parse percentage ie. 50%
	if err != nil {
		var i float64
		_, err := fmt.Sscanf(args[2].Value, "%f%%", &i)
		if err != nil {
			log.Fatal(err)
		}
		wt = i / 100
	}

	c1, err := ast.ColorFromHexString(args[0].Value)
	if err != nil {
		return nil, fmt.Errorf("failed to read hex string %s: %s",
			args[0].Value, err)
	}
	c2, err := ast.ColorFromHexString(args[1].Value)
	if err != nil {
		return nil, fmt.Errorf("failed to read hex string %s: %s",
			args[1].Value, err)
	}

	var r, g, b, a float64

	a1, a2 := float64(c1.A)/100, float64(c2.A)/100
	w := wt*2 - 1
	a = a1 - a2

	var w1 float64
	// if w*a == -1, weight is w
	if w*a == -1 {
		w1 = w
	} else {
		w1 = (w + a) / (1 + w*a)
	}
	w1 = (w1 + 1) / 2
	w2 := 1 - w1

	r = w1*float64(c1.R) + w2*float64(c2.R)
	g = w1*float64(c1.G) + w2*float64(c2.G)
	b = w1*float64(c1.B) + w2*float64(c2.B)

	alpha := (float64(c1.A) + float64(c2.A)) / 2

	ret := color.RGBA{
		R: uint8(round(r, 0)),
		G: uint8(round(g, 0)),
		B: uint8(round(b, 0)),
		A: uint8(round(alpha, 2)),
	}

	return colorOutput(ret, call.Args[0]), nil
}

// https://gist.github.com/DavidVaini/10308388#gistcomment-1460571
func round(v float64, decimals int) float64 {
	var pow float64 = 1
	for i := 0; i < decimals; i++ {
		pow *= 10
	}
	return float64(int((v*pow)+0.5)) / pow
}

func invert(call *ast.CallExpr, args ...*ast.BasicLit) (*ast.BasicLit, error) {
	val := args[0].Value
	c, err := ast.ColorFromHexString(val)
	if err != nil {
		return nil, fmt.Errorf("invert failed to parse argument %s; %s", val, err)
	}

	c.R = 255 - c.R
	c.G = 255 - c.G
	c.B = 255 - c.B

	return colorOutput(c, call.Args[0]), nil
}

// colorOutput inspects the context to determine the appropriate output
func colorOutput(c color.RGBA, outTyp ast.Expr) *ast.BasicLit {
	ctx1 := outTyp
	lit := &ast.BasicLit{
		Kind: token.COLOR,
	}
	attemptLookup := true
	switch ctx := ctx1.(type) {
	case *ast.CallExpr:
		switch ctx.Fun.(*ast.Ident).Name {
		case "rgb":
			lit.Value = fmt.Sprintf("%s(%d, %d, %d)",
				"rgb", c.R, c.G, c.B,
			)
		case "rgba":
			attemptLookup = false
			i := int(c.A) * 10000
			f := float32(i) / 1000000
			lit.Value = fmt.Sprintf("%s(%d, %d, %d, %.2g)",
				"rgba", c.R, c.G, c.B, f,
			)
		default:
			log.Fatal("unsupported ident", ctx.Fun.(*ast.Ident).Name)
		}
	case *ast.BasicLit:
		lit = ast.BasicLitFromColor(c)
	}
	if attemptLookup {
		lit.Value = ast.LookupColor(lit.Value)
	}

	return lit
}
