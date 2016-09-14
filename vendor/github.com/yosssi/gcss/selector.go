package gcss

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// selector represents a selector of CSS.
type selector struct {
	elementBase
	name string
}

// WriteTo writes the selector to the writer.
func (sel *selector) WriteTo(w io.Writer) (int64, error) {
	return sel.writeTo(w, nil)
}

// writeTo writes the selector to the writer.
func (sel *selector) writeTo(w io.Writer, params map[string]string) (int64, error) {
	bf := new(bytes.Buffer)

	// Write the declarations.
	if len(sel.decs) > 0 || sel.hasMixinDecs() {
		bf.WriteString(sel.names())
		bf.WriteString(openBrace)

		// Writing to the bytes.Buffer never returns an error.
		sel.writeDecsTo(bf, params)

		bf.WriteString(closeBrace)
	}

	// Write the child selectors.
	for _, childSel := range sel.sels {
		// Writing to the bytes.Buffer never returns an error.
		childSel.writeTo(bf, params)
	}

	// Write the mixin's selectors.
	for _, mi := range sel.mixins {
		sels, prms := mi.selsParams()

		for _, sl := range sels {
			sl.parent = sel
			// Writing to the bytes.Buffer never returns an error.
			sl.writeTo(bf, prms)
		}
	}

	n, err := w.Write(bf.Bytes())

	return int64(n), err
}

// names returns the selector names.
func (sel *selector) names() string {
	bf := new(bytes.Buffer)

	switch parent := sel.parent.(type) {
	case nil, *atRule:
		for _, name := range strings.Split(sel.name, comma) {
			if bf.Len() > 0 {
				bf.WriteString(comma)
			}

			bf.WriteString(strings.TrimSpace(name))
		}
	case *selector:
		for _, parentS := range strings.Split(parent.names(), comma) {
			for _, s := range strings.Split(sel.name, comma) {
				if bf.Len() > 0 {
					bf.WriteString(comma)
				}

				s = strings.TrimSpace(s)

				if strings.Index(s, ampersand) != -1 {
					bf.WriteString(strings.Replace(s, ampersand, parentS, -1))
				} else {
					bf.WriteString(parentS)
					bf.WriteString(space)
					bf.WriteString(s)
				}
			}
		}
	}

	return bf.String()
}

// newSelector creates and returns a selector.
func newSelector(ln *line, parent element) (*selector, error) {
	name := strings.TrimSpace(ln.s)

	if strings.HasSuffix(name, openBrace) {
		return nil, fmt.Errorf("selector must not end with %q [line: %d]", openBrace, ln.no)
	}

	if strings.HasSuffix(name, closeBrace) {
		return nil, fmt.Errorf("selector must not end with %q [line: %d]", closeBrace, ln.no)
	}

	return &selector{
		elementBase: newElementBase(ln, parent),
		name:        name,
	}, nil
}
