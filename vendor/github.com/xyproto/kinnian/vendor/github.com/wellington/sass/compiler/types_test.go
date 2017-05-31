package compiler

import "testing"

func TestType_simple_list(t *testing.T) {
	runParse(t, `
$x: 1 2 3;
div {
  x: $x;
}`,
		`div {
  x: 1 2 3; }
`)
}
