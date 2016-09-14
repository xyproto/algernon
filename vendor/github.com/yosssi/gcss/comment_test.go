package gcss

import (
	"io/ioutil"
	"testing"
)

func Test_comment_WriteTo(t *testing.T) {
	ln := newLine(1, "// test")

	c := newComment(ln, nil)

	if _, err := c.WriteTo(ioutil.Discard); err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
	}
}

func Test_newComment(t *testing.T) {
	ln := newLine(1, "// test")

	c := newComment(ln, nil)

	if c == nil || c.ln != ln {
		t.Error("c is invalid")
	}
}
