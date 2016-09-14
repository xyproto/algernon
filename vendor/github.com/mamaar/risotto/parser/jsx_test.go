package parser

import (
	"github.com/mamaar/risotto/ast"
	"github.com/stretchr/testify/assert"
	"testing"
)

var validJSX = []string{
	"<div />",
	"<div param=\"value\"></div>",
	"<div><div /></div>",
	"<div prop={name} />",
}

func p(jsx string) (*ast.Program, error) {
	p := newParser("", jsx)
	return p.parse()
}

func TestJSX(t *testing.T) {
	for _, jsx := range validJSX {
		_, err := p(jsx)

		assert.NoError(t, err)

	}
}
