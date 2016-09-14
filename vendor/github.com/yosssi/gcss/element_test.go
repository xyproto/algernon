package gcss

import (
	"reflect"
	"testing"
)

func Test_newElement_selector(t *testing.T) {
	ln := newLine(1, "html")

	e, err := newElement(ln, nil)

	if err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
		return
	}

	if _, ok := e.(*selector); !ok {
		t.Errorf(`e's type should be "*selector" [actual: %q]`, reflect.TypeOf(e))
		return
	}
}

func Test_newElement_mixinDeclaration(t *testing.T) {
	ln := newLine(1, "$test($params1, $params2)")

	e, err := newElement(ln, nil)

	if err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
		return
	}

	if _, ok := e.(*mixinDeclaration); !ok {
		t.Errorf(`e's type should be "*mixinDeclaration" [actual: %q]`, reflect.TypeOf(e))
		return
	}
}
