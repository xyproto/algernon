package gcss

import (
	"bytes"
	"io"
	"strings"
)

// atRule represents an at-rule of CSS.
type atRule struct {
	elementBase
}

// WriteTo writes the at-rule to the writer.
func (ar *atRule) WriteTo(w io.Writer) (int64, error) {
	bf := new(bytes.Buffer)

	bf.WriteString(strings.TrimSpace(ar.ln.s))

	if len(ar.sels) == 0 && len(ar.decs) == 0 && !ar.hasMixinDecs() && !ar.hasMixinSels() {
		bf.WriteString(semicolon)
		n, err := w.Write(bf.Bytes())
		return int64(n), err
	}

	bf.WriteString(openBrace)

	// Writing to the bytes.Buffer never returns an error.
	ar.writeDecsTo(bf, nil)

	for _, sel := range ar.sels {
		// Writing to the bytes.Buffer never returns an error.
		sel.WriteTo(bf)
	}

	// Write the mixin's selectors.
	for _, mi := range ar.mixins {
		sels, prms := mi.selsParams()

		for _, sl := range sels {
			sl.parent = ar
			// Writing to the bytes.Buffer never returns an error.
			sl.writeTo(bf, prms)
		}
	}

	bf.WriteString(closeBrace)

	n, err := w.Write(bf.Bytes())

	return int64(n), err
}

// newAtRule creates and returns a at-rule.
func newAtRule(ln *line, parent element) *atRule {
	return &atRule{
		elementBase: newElementBase(ln, parent),
	}
}
