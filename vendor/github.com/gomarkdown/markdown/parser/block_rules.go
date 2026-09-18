package parser

import (
	"bytes"
	"strconv"

	"github.com/gomarkdown/markdown/ast"
)

type blockRule struct {
	extension Extensions
	parse     func(*Parser, []byte) int
}

// standardBlockRules defines block precedence. The first matching rule consumes
// input. It is function-local because several rules recursively invoke Block.
func standardBlockRules() [17]blockRule {
	return [17]blockRule{
		{parse: parsePrefixHeading},
		{parse: parseSpecialHeading},
		{parse: parseHTMLBlock},
		{extension: Titleblock, parse: parseTitleBlock},
		{parse: parseBlankLines},
		{parse: parseIndentedCode},
		{extension: FencedCode, parse: parseFencedCode},
		{parse: parseHorizontalRule},
		{parse: parseBlockQuote},
		{extension: Mmark, parse: parseAside},
		{extension: Mmark, parse: parseFigure},
		{extension: Tables, parse: parseTable},
		{parse: parseUnorderedList},
		{parse: parseOrderedList},
		{extension: DefinitionLists, parse: parseDefinitionList},
		{extension: MathJax, parse: parseMathBlock},
		{extension: Mmark, parse: parseDocumentMatter},
	}
}

func (p *Parser) parseBlock(data []byte) int {
	for _, rule := range standardBlockRules() {
		if rule.extension != 0 && p.extensions&rule.extension == 0 {
			continue
		}
		if consumed := rule.parse(p, data); consumed > 0 {
			return consumed
		}
	}
	return p.paragraph(data)
}

func parsePrefixHeading(p *Parser, data []byte) int {
	if p.isPrefixHeading(data) {
		return p.prefixHeading(data)
	}
	return 0
}

func parseSpecialHeading(p *Parser, data []byte) int {
	if p.isPrefixSpecialHeading(data) {
		return p.prefixSpecialHeading(data)
	}
	return 0
}

func parseHTMLBlock(p *Parser, data []byte) int {
	if data[0] == '<' {
		return p.html(data, true)
	}
	return 0
}

func parseTitleBlock(p *Parser, data []byte) int { return p.titleBlock(data) }
func parseBlankLines(_ *Parser, data []byte) int { return IsEmpty(data) }

func parseIndentedCode(p *Parser, data []byte) int {
	if codePrefix(data) > 0 {
		return p.code(data)
	}
	return 0
}

func parseFencedCode(p *Parser, data []byte) int { return p.fencedCodeBlock(data, true) }

func parseHorizontalRule(p *Parser, data []byte) int {
	if !isHRule(data) {
		return 0
	}
	end := skipUntilChar(data, 0, '\n')
	p.AddBlock(&ast.HorizontalRule{Leaf: ast.Leaf{Literal: bytes.Trim(data[:end], " \n")}})
	return end
}

func parseBlockQuote(p *Parser, data []byte) int {
	if quotePrefix(data) > 0 {
		return p.quote(data)
	}
	return 0
}

func parseAside(p *Parser, data []byte) int {
	if asidePrefix(data) > 0 {
		return p.aside(data)
	}
	return 0
}

func parseFigure(p *Parser, data []byte) int { return p.figureBlock(data, true) }
func parseTable(p *Parser, data []byte) int  { return p.table(data) }

func parseUnorderedList(p *Parser, data []byte) int {
	if uliPrefix(data) > 0 {
		return p.list(data, 0, 0, '.')
	}
	return 0
}

func parseOrderedList(p *Parser, data []byte) int {
	prefix := oliPrefix(data)
	if prefix == 0 {
		return 0
	}
	start := 0
	delimiter := byte('.')
	if prefix > 2 {
		if p.extensions&OrderedListStart != 0 {
			start, _ = strconv.Atoi(string(data[:prefix-2]))
			if start == 1 {
				start = 0
			}
		}
		delimiter = data[prefix-2]
	}
	return p.list(data, ast.ListTypeOrdered, start, delimiter)
}

func parseDefinitionList(p *Parser, data []byte) int {
	if dliPrefix(data) > 0 {
		return p.list(data, ast.ListTypeDefinition, 0, '.')
	}
	return 0
}

func parseMathBlock(p *Parser, data []byte) int      { return p.blockMath(data) }
func parseDocumentMatter(p *Parser, data []byte) int { return p.documentMatter(data) }
