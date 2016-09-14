package gcss

import (
	"io/ioutil"
	"testing"
)

func Test_mixinInvocation_WriteTo(t *testing.T) {
	ln := newLine(1, "$test(10px, blue)")

	mi, err := newMixinInvocation(ln, nil)

	if err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
	}

	if _, err := mi.WriteTo(ioutil.Discard); err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
	}
}

func Test_newMixinInvocation(t *testing.T) {
	ln := newLine(1, "$test(10px, blue)")

	_, err := newMixinInvocation(ln, nil)

	if err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
	}
}

func Test_newMixinInvocation_err(t *testing.T) {
	ln := newLine(1, "test(10px, blue)")

	_, err := newMixinInvocation(ln, nil)

	if err == nil {
		t.Error("error should occur")
	}

	if expected, actual := "mixin must start with \"$\" [line: 1]", err.Error(); actual != expected {
		t.Errorf("error should be %q [actual: %q]", expected, actual)
	}
}
