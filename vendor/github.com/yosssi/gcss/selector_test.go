package gcss

import (
	"io/ioutil"
	"testing"
)

func Test_selector_WriteTo(t *testing.T) {
	ln := newLine(1, "html")

	sel, err := newSelector(ln, nil)

	if err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
		return
	}

	if _, err := sel.WriteTo(ioutil.Discard); err != nil {
		t.Errorf("err should be nil [err: %s]", err.Error())
		return
	}
}

func Test_selector_AppendChild(t *testing.T) {
	ln := newLine(1, "html")

	sel, err := newSelector(ln, nil)

	if err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
		return
	}

	sel.AppendChild(nil)
}

func Test_newSelector_suffixCloseBraceErr(t *testing.T) {
	ln := newLine(1, "html {}")

	_, err := newSelector(ln, nil)

	if err == nil {
		t.Error("error should be occurred")
		return
	}

	if expected := "selector must not end with \"}\" [line: 1]"; err.Error() != expected {
		t.Errorf("err should be %q [actual: %q]", expected, err.Error())
		return
	}
}

func Test_newSelector(t *testing.T) {
	ln := newLine(1, "html")

	sel, err := newSelector(ln, nil)

	if err != nil {
		t.Errorf("err should be nil [err: %s]", err.Error())
		return
	}

	if sel.ln != ln {
		t.Errorf("sel.ln should be %+v [actual: %+v]", ln, sel.ln)
		return
	}
}
