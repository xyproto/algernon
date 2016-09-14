package unit

import (
	"testing"

	"github.com/shopspring/decimal"
)

var samp = []*Num{
	&Num{Unit: IN, dec: decimal.NewFromFloat(1)},
	&Num{Unit: CM, dec: decimal.NewFromFloat(2.54)},
	&Num{Unit: MM, dec: decimal.NewFromFloat(25.4)},
	&Num{Unit: PT, dec: decimal.NewFromFloat(72)},
	&Num{Unit: PX, dec: decimal.NewFromFloat(96)},
}

func copy(n *Num) *Num {
	x := *n
	return &x
}

func TestConvert(t *testing.T) {

	for i := range samp {
		x := copy(samp[0])
		x.Convert(samp[i])
		if x.Unit != samp[0].Unit {
			t.Errorf("got: %s wanted: %s", samp[0].Unit, x.Unit)
		}
		if e := samp[0].dec; e.Cmp(x.dec) != 0 {
			t.Errorf("got: %s wanted: %s", samp[0].dec, e)
			// if e := samp[i].dec; e != x.dec {
		}
	}
}

func TestAdd(t *testing.T) {
	for i := range samp {
		x := copy(samp[0])
		x.Add(samp[0], samp[i])
		if x.Unit != samp[0].Unit {
			t.Errorf("got: %s wanted: %s", samp[0].Unit, x.Unit)
		}
		if e := decimal.NewFromFloat(2.0); e.Cmp(x.dec) != 0 {
			t.Errorf("%s got: %s wanted: %s", samp[i].Unit, x.dec, e)
		}
	}
}

func TestSub(t *testing.T) {
	for i := range samp {
		x := copy(samp[0])
		x.Sub(samp[0], samp[i])
		if x.Unit != samp[0].Unit {
			t.Errorf("got: %s wanted: %s", samp[0].Unit, x.Unit)
		}
		if e := decimal.NewFromFloat(0.0); e.Cmp(x.dec) != 0 {
			t.Errorf("got: %f wanted: %f", x.dec, e)
		}
	}
}

func TestMul(t *testing.T) {
	for i := range samp {
		x := copy(samp[0])
		x.Mul(samp[0], samp[i])
		if x.Unit != samp[0].Unit {
			t.Errorf("got: %s wanted: %s", samp[0].Unit, x.Unit)
		}
		if e := decimal.NewFromFloat(1.0); e.Cmp(x.dec) != 0 {
			t.Errorf("got: %f wanted: %f", x.dec, e)
		}
	}
}

func TestDiv(t *testing.T) {
	for i := range samp {
		x := copy(samp[0])
		x.Div(samp[0], samp[i])
		if x.Unit != samp[0].Unit {
			t.Errorf("got: %s wanted: %s", samp[0].Unit, x.Unit)
		}
		if e := decimal.NewFromFloat(1.0); e.Cmp(x.dec) != 0 {
			t.Errorf("got: %f wanted: %f", x.dec, e)
		}
	}
}
