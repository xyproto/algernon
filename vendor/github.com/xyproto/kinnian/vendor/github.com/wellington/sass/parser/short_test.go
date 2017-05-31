package parser

import "testing"

var valids = []string{
	"$color: red;",
	`$color: "black" !global;`,
	"p15a: 10 - #a2B;",
	"p18: 10 #a2b + 1;",
	"p20: rgb(10,10,10) + #010001;",
	"rgb(255, $blue: 0, $green: 255);",
	"mix(rgba(#f0e, $alpha: .5)+#111, #00f);",
	// interp
	"$a: h#{ello + world};",
	"a#{id} { a: b; }",
	"a#{id}d {}",
	"$a: 4; $b: #{$a+\"1\"};",
	// functions
	"@function grid-width($x){};;;",
	"b: type-of(12#{3});",
	"$a: 1; type-of($a);",
	// selectors
	"a+b,c{ d e, f~g, > i {}}",
	"a { & {}}",
	"div { /*comment*/ }",
	"g { @media print and (foo: 1 2 3) {}}",
	// Easy way to test resolveCall
	"div { a: #{rgb(4,5,6)+1}; }",
	"$a: 1 2; inspect($a);",
	// directives
	// "@mixin foo($a: one, $b) { p {$x: inside $a;} } @include foo(); @include foo(two);",
	// nested and root are treated ifferently
	"div { @each $i in (1 2 3) {} }",
	// "@mixin foo($a: one, $b) { $x: inside $a; } div { inner { @include foo(); @include foo(two); } }",
}

func TestValid(t *testing.T) {
	for _, src := range valids {
		checkErrors(t, src, src)
	}
}

var invalids = []string{
	"mix(#111);",
}
