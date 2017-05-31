package gcss

import (
	"io/ioutil"
	"testing"
)

func Test_mixinDeclaration_WriteTo(t *testing.T) {
	ln := newLine(1, "$test()")

	md, err := newMixinDeclaration(ln, nil)

	if err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
		return
	}

	if _, err := md.WriteTo(ioutil.Discard); err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
		return
	}
}

func Test_mixinDeclaration_mixinNP_errPrefixDollar(t *testing.T) {
	ln := newLine(1, "test()")

	_, _, err := mixinNP(ln, true)

	if err == nil {
		t.Error("error should occur")
		return
	}

	if expected, actual := "mixin must start with \"$\" [line: 1]", err.Error(); actual != expected {
		t.Errorf("error should be %q [actual: %q]", expected, actual)
		return
	}
}

func Test_mixinDeclaration_mixinNP_errNoOpenParenthesis(t *testing.T) {
	ln := newLine(1, "$test")

	_, _, err := mixinNP(ln, true)

	if err == nil {
		t.Error("error should occur")
		return
	}

	if expected, actual := "mixin's format is invalid [line: 1]", err.Error(); actual != expected {
		t.Errorf("error should be %q [actual: %q]", expected, actual)
		return
	}
}

func Test_mixinDeclaration_mixinNP_errNoCloseParenthesis(t *testing.T) {
	ln := newLine(1, "$test(")

	_, _, err := mixinNP(ln, true)

	if err == nil {
		t.Error("error should occur")
		return
	}

	if expected, actual := "mixin must end with \")\" [line: 1]", err.Error(); actual != expected {
		t.Errorf("error should be %q [actual: %q]", expected, actual)
		return
	}
}

func Test_mixinDeclaration_mixinNP_errMultiCloseParentheses(t *testing.T) {
	ln := newLine(1, "$test())")

	_, _, err := mixinNP(ln, true)

	if err == nil {
		t.Error("error should occur")
		return
	}

	if expected, actual := "mixin's format is invalid [line: 1]", err.Error(); actual != expected {
		t.Errorf("error should be %q [actual: %q]", expected, actual)
		return
	}
}

func Test_mixinDeclaration_mixinNP_noParamNames(t *testing.T) {
	ln := newLine(1, "$test()")

	_, _, err := mixinNP(ln, true)

	if err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
		return
	}
}

func Test_mixinDeclaration_mixinNP(t *testing.T) {
	ln := newLine(1, "$test($param1)")

	_, _, err := mixinNP(ln, true)

	if err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
		return
	}
}

func Test_mixinDeclaration_mixinNP_errInvalidParamNames(t *testing.T) {
	ln := newLine(1, "$test(param1)")

	_, _, err := mixinNP(ln, true)

	if err == nil {
		t.Error("error should occur")
		return
	}

	if expected, actual := "mixin's parameter must start with \"$\" [line: 1]", err.Error(); actual != expected {
		t.Errorf("error should be %q [actual: %q]", expected, actual)
		return
	}
}

func Test_newMixinDeclaration(t *testing.T) {
	ln := newLine(1, "$test($param1, $param2)")

	_, err := newMixinDeclaration(ln, nil)

	if err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
		return
	}
}

func Test_newMixinDeclaration_errInvalidParamNames(t *testing.T) {
	ln := newLine(1, "$test(param1)")

	_, err := newMixinDeclaration(ln, nil)

	if err == nil {
		t.Error("error should occur")
		return
	}

	if expected, actual := "mixin's parameter must start with \"$\" [line: 1]", err.Error(); actual != expected {
		t.Errorf("error should be %q [actual: %q]", expected, actual)
		return
	}
}

func Test_newMixinDeclaration_fromFile(t *testing.T) {
	_, err := CompileFile("test/0014.gcss")

	if err != nil {
		t.Error("error occurred [error: %q]", err.Error())
		return
	}
}
