package gcss

import (
	"fmt"
	"io"
	"strings"
)

// variable represents a GCSS variable.
type variable struct {
	elementBase
	name  string
	value string
}

// WriteTo writes the variable to the writer.
func (v *variable) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write([]byte(v.value))

	return int64(n), err
}

// variableNV extracts a variable name and value
// from the line.
func variableNV(ln *line) (string, string, error) {
	s := strings.TrimSpace(ln.s)

	if !strings.HasPrefix(s, dollarMark) {
		return "", "", fmt.Errorf("variable must start with %q [line: %d]", dollarMark, ln.no)
	}

	nv := strings.SplitN(s, space, 2)

	if len(nv) < 2 {
		return "", "", fmt.Errorf("variable's name and value should be divided by a space [line: %d]", ln.no)
	}

	if !strings.HasSuffix(nv[0], colon) {
		return "", "", fmt.Errorf("variable's name should end with a colon [line: %d]", ln.no)
	}

	return strings.TrimSuffix(strings.TrimPrefix(nv[0], dollarMark), colon), nv[1], nil
}

// newVariable creates and returns a variable.
func newVariable(ln *line, parent element) (*variable, error) {
	name, value, err := variableNV(ln)

	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(value, semicolon) {
		return nil, fmt.Errorf("variable must not end with %q", semicolon)
	}

	return &variable{
		elementBase: newElementBase(ln, parent),
		name:        name,
		value:       value,
	}, nil
}
