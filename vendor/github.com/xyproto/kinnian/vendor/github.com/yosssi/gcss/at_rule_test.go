package gcss

import (
	"io/ioutil"
	"testing"
)

func Test_atRule_WriteTo(t *testing.T) {
	ln := newLine(1, "html")

	ar := newAtRule(ln, nil)

	if _, err := ar.WriteTo(ioutil.Discard); err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
		return
	}
}

func Test_atRule_WriteTo_fromFile(t *testing.T) {
	if _, err := CompileFile("test/0009.gcss"); err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
		return
	}
}

func Test_newAtRule(t *testing.T) {
	ln := newLine(1, "html")

	ar := newAtRule(ln, nil)

	if ar.ln != ln {
		t.Errorf("ar.ln should be %+v [actual: %+v]", ln, ar.ln)
		return
	}
}
