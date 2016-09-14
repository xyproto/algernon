package gcss

import (
	"bytes"
	"io"
)

// elementBase holds the common fields of an element.
type elementBase struct {
	ln     *line
	parent element
	sels   []*selector
	decs   []*declaration
	mixins []*mixinInvocation
	ctx    *context
}

// AppendChild appends a child element to the element.
func (eBase *elementBase) AppendChild(child element) {
	switch c := child.(type) {
	case *mixinInvocation:
		eBase.mixins = append(eBase.mixins, c)
	case *declaration:
		eBase.decs = append(eBase.decs, c)
	case *selector:
		eBase.sels = append(eBase.sels, c)
	}
}

// Base returns the element base.
func (eBase *elementBase) Base() *elementBase {
	return eBase
}

// SetContext sets the context to the element.
func (eBase *elementBase) SetContext(ctx *context) {
	eBase.ctx = ctx
}

// Context returns the top element's context.
func (eBase *elementBase) Context() *context {
	if eBase.parent != nil {
		return eBase.parent.Context()
	}

	return eBase.ctx
}

// hasMixinDecs returns true if the element has a mixin
// which has declarations.
func (eBase *elementBase) hasMixinDecs() bool {
	for _, mi := range eBase.mixins {
		if decs, _ := mi.decsParams(); len(decs) > 0 {
			return true
		}
	}

	return false
}

// hasMixinSels returns true if the element has a mixin
// which has selectors.
func (eBase *elementBase) hasMixinSels() bool {
	for _, mi := range eBase.mixins {
		if sels, _ := mi.selsParams(); len(sels) > 0 {
			return true
		}
	}

	return false
}

// writeDecsTo writes the element's declarations to w.
func (eBase *elementBase) writeDecsTo(w io.Writer, params map[string]string) (int64, error) {
	bf := new(bytes.Buffer)

	// Write the declarations.
	for _, dec := range eBase.decs {
		// Writing to the bytes.Buffer never returns an error.
		dec.writeTo(bf, params)
	}

	// Write the mixin's declarations.
	for _, mi := range eBase.mixins {
		decs, prms := mi.decsParams()

		for _, dec := range decs {
			// Writing to the bytes.Buffer never returns an error.
			dec.writeTo(bf, prms)
		}
	}

	n, err := w.Write(bf.Bytes())

	return int64(n), err
}

// newElementBase creates and returns an element base.
func newElementBase(ln *line, parent element) elementBase {
	return elementBase{
		ln:     ln,
		parent: parent,
	}
}
