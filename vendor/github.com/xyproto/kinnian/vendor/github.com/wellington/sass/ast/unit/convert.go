package unit

import (
	"fmt"
	"math"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/wellington/sass/ast"
	"github.com/wellington/sass/token"
)

// Unit represents the CSS unit being described
type Unit int

// Why does this exist?
//
// var unitTypes = map[string]string{
// 	"in":   "distance",
// 	"cm":   "distance",
// 	"pc":   "distance",
// 	"mm":   "distance",
// 	"pt":   "distance",
// 	"px":   "distance",
// 	"deg":  "angle",
// 	"grad": "angle",
// 	"rad":  "angle",
// 	"turn": "angle",
// }

const (
	// INVALID unit can not be converted
	INVALID Unit = iota
	IN
	CM
	MM
	PC
	PX
	PT
	DEG
	GRAD
	RAD
	TURN
	// NOUNIT represents float or int that perform ops but are not
	// tied to a unit
	NOUNIT
)

func (u Unit) String() string {
	switch u {
	case IN:
		return "IN"
	case CM:
		return "CM"
	case MM:
		return "MM"
	case PC:
		return "PC"
	case PX:
		return "PX"
	case PT:
		return "PT"
	case DEG:
		return "DEG"
	case GRAD:
		return "GRAD"
	case RAD:
		return "RAD"
	case TURN:
		return "TURN"
	case NOUNIT:
		return "NOUNIT"
	}
	return "invalid"
}

var mlook = map[token.Token]Unit{
	token.FLOAT: NOUNIT,
	token.INT:   NOUNIT,
	token.UIN:   IN,
	token.UCM:   CM,
	token.UMM:   MM,
	token.UPC:   PC,
	token.UPX:   PX,
	token.UPT:   PT,
	token.DEG:   DEG,
	token.GRAD:  GRAD,
	token.RAD:   RAD,
	token.TURN:  TURN,
}

func unitLookup(tok token.Token) Unit {
	u, ok := mlook[tok]
	if !ok {
		return INVALID
	}
	return u
}

func tokLookup(u Unit) token.Token {
	for t, unit := range mlook {
		if unit == u {
			return t
		}
	}
	return token.ILLEGAL
}

var cnv = [...]float64{
	PC: 6,
	CM: 2.54,
	MM: 25.4,
	PT: 72,
	PX: 96,
}

var unitconv = [...][11]float64{
	IN: {
		IN:   1,
		CM:   2.54,
		PC:   6,
		MM:   25.4,
		PT:   72,
		PX:   96,
		DEG:  1,
		GRAD: 1,
		RAD:  1,
		TURN: 1,
	},
	PC: {
		IN:   1.0 / cnv[PC],
		CM:   2.54 / cnv[PC],
		PC:   6.0 / cnv[PC],
		MM:   25.4 / cnv[PC],
		PT:   72 / cnv[PC],
		PX:   96 / cnv[PC],
		DEG:  1,
		GRAD: 1,
		RAD:  1,
		TURN: 1,
	},
	CM: {
		IN:   1 / cnv[CM],
		CM:   2.54 / cnv[CM],
		PC:   6 / cnv[CM],
		MM:   25.4 / cnv[CM],
		PT:   72 / cnv[CM],
		PX:   96 / cnv[CM],
		DEG:  1,
		GRAD: 1,
		RAD:  1,
		TURN: 1,
	},
	MM: {
		IN:   1 / cnv[MM],
		CM:   2.54 / cnv[MM],
		PC:   6 / cnv[MM],
		MM:   25.4 / cnv[MM],
		PT:   72 / cnv[MM],
		PX:   96 / cnv[MM],
		DEG:  1,
		GRAD: 1,
		RAD:  1,
		TURN: 1,
	},
	PT: {
		IN:   1 / cnv[PT],
		CM:   2.54 / cnv[PT],
		PC:   6 / cnv[PT],
		MM:   25.4 / cnv[PT],
		PT:   72 / cnv[PT],
		PX:   96 / cnv[PT],
		DEG:  1,
		GRAD: 1,
		RAD:  1,
		TURN: 1,
	},
	PX: {
		IN:   1 / cnv[PX],
		CM:   2.54 / cnv[PX],
		PC:   6 / cnv[PX],
		MM:   25.4 / cnv[PX],
		PT:   72 / cnv[PX],
		PX:   96 / cnv[PX],
		DEG:  1,
		GRAD: 1,
		RAD:  1,
		TURN: 1,
	},
	// conversion not useful for these
	DEG: {
		IN:   1,
		CM:   1,
		PC:   1,
		MM:   1,
		PT:   1,
		PX:   1,
		DEG:  1,
		GRAD: 4 / 3.6,
		RAD:  math.Pi / 180.0,
		TURN: 1.0 / 360.0,
	},
	GRAD: {
		IN:   1,
		CM:   1,
		PC:   1,
		MM:   1,
		PT:   1,
		PX:   1,
		DEG:  3.6 / 4,
		GRAD: 1,
		RAD:  math.Pi / 200.0,
		TURN: 1.0 / 400.0,
	},
	RAD: {
		IN:   1,
		CM:   1,
		PC:   1,
		MM:   1,
		PT:   1,
		PX:   1,
		DEG:  180 / math.Pi,
		GRAD: 200 / math.Pi,
		RAD:  1,
		TURN: math.Pi / 2,
	},
	TURN: {
		IN:   1,
		CM:   1,
		PC:   1,
		MM:   1,
		PT:   1,
		PX:   1,
		DEG:  360,
		GRAD: 400,
		RAD:  2.0 * math.Pi,
		TURN: 1,
	},
	NOUNIT: {
		IN:   1,
		CM:   1,
		PC:   1,
		MM:   1,
		PT:   1,
		PX:   1,
		DEG:  1,
		GRAD: 1,
		RAD:  1,
		TURN: 1,
	},
}

func init() {
	ast.RegisterKind(Combine,
		token.UIN, token.UCM, token.UPC, token.UMM,
		token.UPT, token.UPX, token.DEG, token.GRAD,
		token.RAD, token.TURN)
}

// Combine lit with specified kind rules
func Combine(op token.Token, x, y *ast.BasicLit, combine bool) (*ast.BasicLit, error) {
	// So we have some non-standard units, convert to INT/FLOAT
	// and send to another handler. Return always matches the unit
	// of x
	// var unitx, unity token.Token

	m, err := NewNum(x)
	if err != nil {
		return nil, err
	}
	n, err := NewNum(y)
	if err != nil {
		return nil, err
	}

	m.Op(op, m, n)
	return m.Lit()
}

// Precision used when performing calculations
var Precision int32 = 6

// Num represents a float with units. This isn't useful for
// unitless operations, since "calc" already does this.
type Num struct {
	pos token.Pos
	dec decimal.Decimal
	Unit
}

// NewNum initializes a Num from a BasicLit. Kind will hold the unit
// the number portion is always treated as a float.
func NewNum(lit *ast.BasicLit) (*Num, error) {
	val := lit.Value
	// TODO: scanner should remove unit
	kind := lit.Kind
	val = strings.TrimSuffix(lit.Value, token.Tokens[kind])
	dec, err := decimal.NewFromString(val)
	return &Num{dec: dec, Unit: unitLookup(kind)}, err
}

func (z *Num) String() string {
	d := z.dec.Round(Precision).String()
	// s := strconv.FormatFloat(z.f, 'G', -1, 64)
	if z.Unit == NOUNIT {
		return d
	}
	return d + tokLookup(z.Unit).String()
}

// Lit attempts to convert Num back into a Lit.
func (z *Num) Lit() (*ast.BasicLit, error) {
	return &ast.BasicLit{
		Kind:     tokLookup(z.Unit),
		Value:    z.String(),
		ValuePos: z.pos,
	}, nil
}

// Convert src to z, applying proper conversion to src
func (z *Num) Convert(src *Num) *Num {
	u := z.Unit
	var cv float64
	if u == NOUNIT {
		// nounit inherits unit of src
		u = src.Unit
		cv = 1
	} else {
		cv = unitconv[src.Unit][z.Unit]
	}

	dcv := decimal.NewFromFloat(cv)
	return &Num{
		Unit: z.Unit,
		dec:  src.dec.Mul(dcv).Round(Precision),
	}
}

// Op returns the sum of x and y using the specified Op
func (z *Num) Op(op token.Token, x, y *Num) *Num {
	switch op {
	case token.ADD:
		return z.Add(x, y)
	case token.MUL:
		return z.Mul(x, y)
	case token.QUO:
		return z.Div(x, y)
	case token.SUB:
		return z.Sub(x, y)
	default:
		panic(fmt.Errorf("unsupported unit op: %s", op))
	}
}

// Add returns the sum of x and y
func (z *Num) Add(x, y *Num) *Num {
	if z.Unit == NOUNIT {
		// Look around for a unit!
		// Only one of these can be true
		if x.Unit != NOUNIT {
			z.Unit = x.Unit
		} else if y.Unit != NOUNIT {
			z.Unit = y.Unit
		}

	}
	// n controls output unit
	// fmt.Printf("adding %s:%s + %s:%s ", x.Unit, x.dec, y.Unit, y.dec)
	a, b := z.Convert(x), z.Convert(y)
	z.dec = a.dec.Add(b.dec).Round(Precision)
	fmt.Printf("res    %s + %s = %s\n", a, b, z)

	return z
}

// Sub returns the subtraction of x and y
func (z *Num) Sub(x, y *Num) *Num {
	// n controls output unit
	a, b := z.Convert(x), z.Convert(y)
	z.dec = a.dec.Sub(b.dec).Round(Precision)
	return z
}

// Mul returns the multiplication of x and y
func (z *Num) Mul(x, y *Num) *Num {
	// n controls output unit
	a, b := z.Convert(x), z.Convert(y)
	z.dec = a.dec.Mul(b.dec).Round(Precision)
	return z
}

// Div returns the rounded division of x and y
func (z *Num) Div(x, y *Num) *Num {
	// n controls output unit
	a, b := z.Convert(x), z.Convert(y)
	z.dec = a.dec.Div(b.dec).Round(Precision)
	return z
}
