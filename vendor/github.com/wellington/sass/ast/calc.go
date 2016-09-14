package ast

import (
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/wellington/sass/token"
)

// ErrNoCombine indicates that combination is not necessary
// The result should be treated as a sasslist or other form
// and input whitespace may be essential to compiler output.
var ErrNoCombine = errors.New("no op to perform")

// ErrIllegalOp indicate parsing errors on operands
var ErrIllegalOp = errors.New("operand is illegal")

type kind struct {
	unit    token.Token
	combine func(op token.Token, x, y *BasicLit, combine bool) (*BasicLit, error)
}

var kinds []kind

// RegisterKind enables additional external Operations. These could
// be color math or other non-literal math unsupported directly
// in ast.
func RegisterKind(fn func(op token.Token, x, y *BasicLit, combine bool) (*BasicLit, error), units ...token.Token) {
	for _, unit := range units {
		kinds = append(kinds, kind{unit: unit, combine: fn})
	}
}

// Op processes x op y applying unit conversions as necessary.
// combine forces operations on unitless numbers. By default,
// unitless numbers are not combined.
func Op(op token.Token, x, y *BasicLit, combine bool) (*BasicLit, error) {
	fmt.Printf("kind: %s op: %s y: %s combine: %t x: % #v y: % #v\n",
		x.Kind, op, y.Kind, combine, x, y)
	if x.Kind == token.ILLEGAL || y.Kind == token.ILLEGAL {
		return nil, ErrIllegalOp
	}

	switch op {
	case token.MUL, token.ADD:
		// always combine these
		combine = true
	}

	kind := x.Kind
	var fn func(token.Token, *BasicLit, *BasicLit, bool) (*BasicLit, error)
	switch {
	case kind == token.INT:
		switch y.Kind {
		case token.INT:
			fn = intOp
		case token.STRING:
			fn = stringOp
		case token.FLOAT:
			fn = floatOp
		case token.UPCT:
			fn = pctOp
		}
		// Other Kinds that could act as Float
	case kind == token.FLOAT:
		switch y.Kind {
		case token.STRING:
			fn = stringOp
		default:
			fn = floatOp
		}
	case kind == token.STRING:
		fmt.Println("string op?", x.Value, y.Value)
		fn = stringOp
	case kind == token.UPCT:
		fn = pctOp
	}

	// math operations do not happen unless explicity enforced
	// on * and /
	if op == token.QUO || op == token.MUL {
		if fn != nil && !combine {
			fn = stringOp
		}
	}

	if fn == nil {
		switch {
		case x.Kind == token.QSSTRING || y.Kind == token.QSSTRING:
			fallthrough
		case x.Kind == token.QSTRING || y.Kind == token.QSTRING:
			fallthrough
		case x.Kind == token.STRING || y.Kind == token.STRING:
			fn = stringOp
		}
	}

	if fn == nil {

		units := []token.Token{token.UEM, token.UPX}
		for _, u := range units {
			// case kind == token.UEM:
			if x.Kind == u || y.Kind == u {
				return otherOp(op, x, y, combine)
			}
		}

		// no functions matched, check registered functions
		for _, k := range kinds {
			if k.unit == kind {
				log.Println("x match", kind)
				fn = k.combine
			}
			if k.unit == y.Kind {
				log.Println("y match", y.Kind)
				fn = k.combine
			}
		}

		if fn == nil {
			return nil, fmt.Errorf("unsupported Operands %s:%s",
				x.Kind, y.Kind)
		}
	}

	lit, err := fn(op, x, y, combine)
	return lit, err
}

func pctOp(op token.Token, x, y *BasicLit, combine bool) (*BasicLit, error) {
	xx := x
	xx.Value = strings.TrimSuffix(x.Value, "%")
	yy := y
	yy.Value = strings.TrimSuffix(y.Value, "%")
	// catch case where dividing % by % results in unitless
	if x.Kind == y.Kind {
		if op == token.QUO {
			xx.Kind = token.FLOAT
			yy.Kind = token.FLOAT
			return floatOp(op, xx, yy, combine)
		}
	}
	xx.Kind = token.FLOAT
	yy.Kind = token.FLOAT

	lit, err := floatOp(op, xx, yy, combine)
	lit.Kind = token.UPCT
	lit.Value += "%"
	return lit, err
}

func otherOp(op token.Token, x, y *BasicLit, combine bool) (*BasicLit, error) {
	if !combine && op == token.QUO {
		return stringOp(op, x, y, combine)
	}

	a, b := x, y
	// Order for non-string operations is not important. Reorder for simpler logic
	switch {
	case x.Kind == token.INT || x.Kind == token.FLOAT:
		a = y
		b = x
	case y.Kind == token.INT || y.Kind == token.FLOAT:
	default:
		if op == token.MUL {
			return nil, fmt.Errorf("unsupported operation: %s %s %s", x, op, y)
		}
	}

	var kind token.Token
	switch a.Kind {
	case token.ILLEGAL:
	default:
		kind = a.Kind
	}

	switch b.Kind {
	case token.ILLEGAL:
	default:
		switch {
		case kind == token.ILLEGAL:
			kind = b.Kind
		case b.Kind == token.INT || b.Kind == token.FLOAT:
		case kind != b.Kind:
			return nil, fmt.Errorf("illegal unit operation %s %s",
				a.Kind, b.Kind)
		}
	}
	xx := &BasicLit{
		Value: strings.TrimSuffix(a.Value, a.Kind.String()),
	}
	yy := &BasicLit{
		Value: strings.TrimSuffix(b.Value, b.Kind.String()),
	}

	f, err := floatOp(op, xx, yy, combine)
	return &BasicLit{
		Kind:  kind,
		Value: f.Value + kind.String(),
	}, err
}

func floatOp(op token.Token, x, y *BasicLit, combine bool) (*BasicLit, error) {
	out := &BasicLit{
		Kind: token.FLOAT,
	}
	if !combine {
		out.Value = x.Value + op.String() + y.Value
		return out, nil
	}

	l, err := strconv.ParseFloat(x.Value, 64)
	if err != nil {
		return out, err
	}
	r, err := strconv.ParseFloat(y.Value, 64)
	if err != nil {
		return out, err
	}
	var t float64
	switch op {
	case token.ADD:
		t = l + r
	case token.SUB:
		t = l - r
	case token.QUO:
		// Sass division can create floats, so much treat
		// ints as floats then test for fitting inside INT
		t = l / r
	case token.MUL:
		t = l * r
	default:
		panic("unsupported intOp" + op.String())
	}
	out.Value = strconv.FormatFloat(t, 'G', -1, 64)
	if math.Remainder(t, 1) == 0 {
		out.Kind = token.INT
	}
	return out, nil

}

func intOp(op token.Token, x, y *BasicLit, combine bool) (*BasicLit, error) {
	out := &BasicLit{
		Kind: x.Kind,
	}
	l, err := strconv.Atoi(x.Value)
	if err != nil {
		return out, err
	}
	r, err := strconv.Atoi(y.Value)
	if err != nil {
		return out, err
	}
	var t int
	switch op {
	case token.ADD:
		t = l + r
	case token.SUB:
		t = l - r
	case token.QUO:
		// Sass division can create floats, so much treat
		// ints as floats then test for fitting inside INT
		fl, fr := float64(l), float64(r)
		if math.Remainder(fl, fr) != 0 {
			return floatOp(op, x, y, combine)
		}
		t = l / r
	case token.MUL:
		t = l * r
	default:
		panic("unsupported intOp" + op.String())
	}
	out.Value = strconv.Itoa(t)
	return out, nil
}

func stringOp(op token.Token, x, y *BasicLit, combine bool) (*BasicLit, error) {
	kind := token.STRING
	if op == token.ADD {
		return &BasicLit{
			Kind:  kind,
			Value: x.Value + y.Value,
		}, nil
	}

	return &BasicLit{
		Kind:  kind,
		Value: x.Value + op.String() + y.Value,
	}, nil
}
