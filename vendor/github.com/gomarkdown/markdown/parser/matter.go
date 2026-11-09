package parser

import (
	"bytes"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/internal/matter"
)

func (p *Parser) documentMatter(data []byte) int {
	if data[0] != '{' {
		return 0
	}

	for _, kind := range matter.Kinds {
		tag := matter.Tag(kind)
		if !bytes.HasPrefix(data, []byte(tag)) {
			continue
		}
		node := &ast.DocumentMatter{Matter: kind}
		p.AddBlock(node)
		p.Finalize(node)
		return len(tag)
	}
	return 0
}
