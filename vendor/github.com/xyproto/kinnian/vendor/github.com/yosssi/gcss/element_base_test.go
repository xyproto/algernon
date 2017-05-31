package gcss

import "testing"

func Test_elementBase_Base(t *testing.T) {
	e, err := newElement(newLine(1, "html"), nil)

	if err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
		return
	}

	if e.Base() == nil {
		t.Error("e.Base() should not be nil")
		return
	}
}

func Test_newElementBase(t *testing.T) {
	ln := newLine(1, "html")

	eBase := newElementBase(ln, nil)

	if eBase.ln != ln {
		t.Errorf("eBase.ln should be %+v [actual: %+v]", ln, eBase.ln)
		return
	}
}
