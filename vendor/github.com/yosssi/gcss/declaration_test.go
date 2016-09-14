package gcss

import (
	"io/ioutil"
	"testing"
)

func Test_declaration_WriteTo(t *testing.T) {
	ln := newLine(1, "font-size: 12px")

	dec, err := newDeclaration(ln, nil)

	if err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
	}

	_, err = dec.WriteTo(ioutil.Discard)

	if err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
	}
}

func Test_newDeclaration_semicolonSuffixErr(t *testing.T) {
	ln := newLine(1, "color: blue;")

	_, err := newDeclaration(ln, nil)

	if err == nil {
		t.Error("error should be occurred")
	}

	if expected := "declaration must not end with \";\" [line: 1]"; expected != err.Error() {
		t.Errorf("err should be %q [actual: %q]", expected, err.Error())
	}
}

func Test_newDeclaration(t *testing.T) {
	ln := newLine(1, "html")

	_, err := newDeclaration(ln, nil)

	if err == nil {
		t.Error("error should be occurred")
	}

	if expected := "declaration's property and value should be divided by a space [line: 1]"; expected != err.Error() {
		t.Errorf("err should be %q [actual: %q]", expected, err.Error())
	}
}
