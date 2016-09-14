package compiler

import "testing"

func TestMath_unit_convert(t *testing.T) {
	in := `
div {
  v: w + 4px;
  w: 4px + w;
  o: 3px + 3px + 3px;
  p: 4 + 1px;
  no: 15 / 3 / 5;
  yes: ( 15 / 3 / 5 );
}
`
	e := `div {
  v: w4px;
  w: 4pxw;
  o: 9px;
  p: 5px;
  no: 15/3/5;
  yes: 1; }
`
	runParse(t, in, e)
}

func TestMath_fractions(t *testing.T) {
	in := `
div {
  a: 1 + 2;
  b: 3 + 3/4;
  c: 1/2 + 1/2;
  d: 1/2;
}
`
	e := `div {
  a: 3;
  b: 3.75;
  c: 1;
  d: 1/2; }
`
	runParse(t, in, e)
}

func TestMath_list(t *testing.T) {
	in := `
div {
  e: 1 + (5/10 4 7 8);
  f: (5/10 2 3) + 1;
  g: (15 / 3) / 5;
}
`
	e := `div {
  e: 15/10 4 7 8;
  f: 5/10 2 31;
  g: 1; }
`
	runParse(t, in, e)
}

func TestMath_var(t *testing.T) {
	in := `
$three: 3;
div {
  k: 15 / $three;
  l: 15 / 5 / $three;
}
`
	e := `div {
  k: 5;
  l: 1; }
`
	runParse(t, in, e)
}

func TestMath_mixed_unit(t *testing.T) {

	in := `
div {
  r: 16em * 4;
  s: (10em / 2);
  t: 5em/2;
}
`
	e := `div {
  r: 64em;
  s: 5em;
  t: 5em/2; }
`
	runParse(t, in, e)
}

func TestMath_color(t *testing.T) {
	in := `
div {
   p01: #AbC;
  p02: #AAbbCC;
  p03: #AbC + hello;
  p04: #AbC + 1; // add 1 to each triplet
  p05: #AbC + #001; // triplet-wise addition
  p06: #0000ff + 1; // add 1 to each triplet; ignore overflow because it doesn't correspond to a color name
  p07: #0000ff + #000001; // convert overflow to name of color (blue)
  p08: #00ffff + #000101; // aqua
  p09: #000000;
  p10: #000000 - 1; // black
  p11: #000000 - #000001; // black
  p12: #ffff00 + #010100; // yellow
  p13: (#101010 / 7);
  p14: #000 + 0;
  p15a: 10 - #a2B;
  p15b: 10 - #aa22BB;
  p16: #000 - #001;
  p17: #f0F + #101;
  p18: 10 #a2B + 1;
  p19a: (10 / #a2B);
  p19b: (10 / #aa22BB);
  p20: rgb(10,10,10) + #010001;
  p21: #010000 + rgb(255, 255, 255);
}
`
	e := `div {
  p01: #AbC;
  p02: #AAbbCC;
  p03: #AbChello;
  p04: #abbccd;
  p05: #aabbdd;
  p06: #0101ff;
  p07: blue;
  p08: cyan;
  p09: #000000;
  p10: black;
  p11: black;
  p12: yellow;
  p13: #020202;
  p14: black;
  p15a: 10-#a2B;
  p15b: 10-#aa22BB;
  p16: black;
  p17: magenta;
  p18: 10 #ab23bc;
  p19a: 10/#a2B;
  p19b: 10/#aa22BB;
  p20: #0b0a0b;
  p21: white; }
`
	runParse(t, in, e)
}
