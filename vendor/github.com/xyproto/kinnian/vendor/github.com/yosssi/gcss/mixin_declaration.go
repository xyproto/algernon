package gcss

import (
	"fmt"
	"io"
	"strings"
)

// mixinDeclaration represents a mixin declaration.
type mixinDeclaration struct {
	elementBase
	name       string
	paramNames []string
}

// WriteTo writes the selector to the writer.
func (md *mixinDeclaration) WriteTo(w io.Writer) (int64, error) {
	return 0, nil
}

// mixinNP extracts a mixin name and parameters from the line.
func mixinNP(ln *line, isDeclaration bool) (string, []string, error) {
	s := strings.TrimSpace(ln.s)

	if !strings.HasPrefix(s, dollarMark) {
		return "", nil, fmt.Errorf("mixin must start with %q [line: %d]", dollarMark, ln.no)
	}

	s = strings.TrimPrefix(s, dollarMark)

	np := strings.Split(s, openParenthesis)

	if len(np) != 2 {
		return "", nil, fmt.Errorf("mixin's format is invalid [line: %d]", ln.no)
	}

	paramsS := strings.TrimSpace(np[1])

	if !strings.HasSuffix(paramsS, closeParenthesis) {
		return "", nil, fmt.Errorf("mixin must end with %q [line: %d]", closeParenthesis, ln.no)
	}

	paramsS = strings.TrimSuffix(paramsS, closeParenthesis)

	if strings.Index(paramsS, closeParenthesis) != -1 {
		return "", nil, fmt.Errorf("mixin's format is invalid [line: %d]", ln.no)
	}

	var params []string

	if paramsS != "" {
		params = strings.Split(paramsS, comma)
	}

	for i, p := range params {
		p = strings.TrimSpace(p)

		if isDeclaration {
			if !strings.HasPrefix(p, dollarMark) {
				return "", nil, fmt.Errorf("mixin's parameter must start with %q [line: %d]", dollarMark, ln.no)
			}

			p = strings.TrimPrefix(p, dollarMark)
		}

		params[i] = p
	}

	return np[0], params, nil
}

// newMixinDeclaration creates and returns a mixin declaration.
func newMixinDeclaration(ln *line, parent element) (*mixinDeclaration, error) {
	name, paramNames, err := mixinNP(ln, true)

	if err != nil {
		return nil, err
	}

	return &mixinDeclaration{
		elementBase: newElementBase(ln, parent),
		name:        name,
		paramNames:  paramNames,
	}, nil
}
