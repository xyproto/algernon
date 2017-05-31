package gcss

import "io"

// mixinInvocation represents a mixin invocation.
type mixinInvocation struct {
	elementBase
	name        string
	paramValues []string
}

// WriteTo writes the selector to the writer.
func (mi *mixinInvocation) WriteTo(w io.Writer) (int64, error) {
	return 0, nil
}

// decsParams returns the mixin's declarations and params.
func (mi *mixinInvocation) decsParams() ([]*declaration, map[string]string) {
	md, ok := mi.Context().mixins[mi.name]

	if !ok {
		return nil, nil
	}

	params := make(map[string]string)

	l := len(mi.paramValues)

	for i, name := range md.paramNames {
		if i < l {
			params[name] = mi.paramValues[i]
		}
	}

	return md.decs, params
}

// selsParams returns the mixin's selectors and params.
func (mi *mixinInvocation) selsParams() ([]*selector, map[string]string) {
	md, ok := mi.Context().mixins[mi.name]

	if !ok {
		return nil, nil
	}

	params := make(map[string]string)

	l := len(mi.paramValues)

	for i, name := range md.paramNames {
		if i < l {
			params[name] = mi.paramValues[i]
		}
	}

	return md.sels, params
}

// newMixinInvocation creates and returns a mixin invocation.
func newMixinInvocation(ln *line, parent element) (*mixinInvocation, error) {
	name, paramValues, err := mixinNP(ln, false)

	if err != nil {
		return nil, err
	}

	return &mixinInvocation{
		elementBase: newElementBase(ln, parent),
		name:        name,
		paramValues: paramValues,
	}, nil
}
