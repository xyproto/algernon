package parser

import (
	"bytes"
	"strconv"

	"github.com/gomarkdown/markdown/ast"
)

type blockRule struct {
	extension Extensions
	// match and prefix, when set, must accept data before parse runs; prefix
	// accepts it by returning a length above 0.
	match  func(*Parser, []byte) bool
	prefix func([]byte) int
	parse  func(*Parser, []byte) int
	// render, set instead of parse, is a parser that can run without
	// rendering; a rule runs it with rendering.
	render func(*Parser, []byte, bool) int
}

// standardBlockRules defines block precedence. The first matching rule consumes
// input. It is function-local because several rules recursively invoke Block.
func standardBlockRules() [17]blockRule {
	return [17]blockRule{
		{match: (*Parser).isPrefixHeading, parse: (*Parser).prefixHeading},
		{match: (*Parser).isPrefixSpecialHeading, parse: (*Parser).prefixSpecialHeading},
		{match: startsHTML, render: (*Parser).html},
		{extension: Titleblock, parse: (*Parser).titleBlock},
		{parse: parseBlankLines},
		{prefix: codePrefix, parse: (*Parser).code},
		{extension: FencedCode, render: (*Parser).fencedCodeBlock},
		{parse: parseHorizontalRule},
		{prefix: quotePrefix, parse: (*Parser).quote},
		{extension: Mmark, prefix: asidePrefix, parse: (*Parser).aside},
		{extension: Mmark, render: (*Parser).figureBlock},
		{extension: Tables, parse: (*Parser).table},
		{prefix: uliPrefix, parse: parseUnorderedList},
		{parse: parseOrderedList},
		{extension: DefinitionLists, prefix: dliPrefix, parse: parseDefinitionList},
		{extension: MathJax, parse: (*Parser).blockMath},
		{extension: Mmark, parse: (*Parser).documentMatter},
	}
}

func (p *Parser) parseBlock(data []byte) int {
	for _, rule := range standardBlockRules() {
		if rule.extension != 0 && p.extensions&rule.extension == 0 {
			continue
		}
		if rule.match != nil && !rule.match(p, data) || rule.prefix != nil && rule.prefix(data) == 0 {
			continue
		}
		var consumed int
		if rule.render != nil {
			consumed = rule.render(p, data, true)
		} else {
			consumed = rule.parse(p, data)
		}
		if consumed > 0 {
			return consumed
		}
	}
	return p.paragraph(data)
}

func startsHTML(_ *Parser, data []byte) bool { return data[0] == '<' }

func parseBlankLines(_ *Parser, data []byte) int { return IsEmpty(data) }

func parseHorizontalRule(p *Parser, data []byte) int {
	if !isHRule(data) {
		return 0
	}
	end := skipUntilChar(data, 0, '\n')
	p.AddBlock(&ast.HorizontalRule{Leaf: ast.Leaf{Literal: bytes.Trim(data[:end], " \n")}})
	return end
}

func parseUnorderedList(p *Parser, data []byte) int { return p.list(data, 0, 0, '.') }

func parseOrderedList(p *Parser, data []byte) int {
	prefix := oliPrefix(data)
	if prefix == 0 {
		return 0
	}
	start := 0
	delimiter := byte('.')
	if prefix > 2 {
		if p.extensions&OrderedListStart != 0 {
			start, _ = strconv.Atoi(string(bytes.TrimLeft(data[:prefix-2], " ")))
			if start == 1 {
				start = 0
			}
		}
		delimiter = data[prefix-2]
	}
	return p.list(data, ast.ListTypeOrdered, start, delimiter)
}

func parseDefinitionList(p *Parser, data []byte) int {
	return p.list(data, ast.ListTypeDefinition, 0, '.')
}
